// Copyright 2017 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// govanityurls serves Go vanity URLs.
package main

import (
	"embed"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v2"
)

var (
	//go:embed templates
	templates embed.FS
)

type (
	VanityHandler struct {
		host         string
		CacheControl string
		paths        PathConfigSet
	}

	PathConfigSet []PathConfig

	PathConfig struct {
		path    string
		repo    string
		display string
		vcs     string
	}

	VanityTemplate struct {
		Import  string
		SubPath string
		Repo    string
		Display string
		VCS     string
	}

	VanityConfig struct {
		Host     string                `yaml:"host,omitempty"`
		CacheAge *int64                `yaml:"cache_max_age,omitempty"`
		Paths    map[string]VanityPath `yaml:"paths,omitempty"`
	}

	VanityPath struct {
		Repo    string `yaml:"repo,omitempty"`
		Display string `yaml:"display,omitempty"`
		VCS     string `yaml:"vcs,omitempty"`
	}
)

func (h *VanityHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	current := r.URL.Path
	pc, subpath := h.paths.Find(current)

	if pc == nil && current == "/" {
		h.ServeIndex(w, r)
		return
	}

	if pc == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Cache-Control", h.CacheControl)

	vanityTmpl := template.Must(template.ParseFS(templates, "templates/vanity.html.tmpl"))
	if err := vanityTmpl.Execute(w, VanityTemplate{
		Import:  h.Host(r) + pc.path,
		SubPath: subpath,
		Repo:    pc.repo,
		Display: pc.display,
		VCS:     pc.vcs,
	}); err != nil {
		http.Error(w, "cannot render the page", http.StatusInternalServerError)
	}
}

func (h *VanityHandler) ServeIndex(w http.ResponseWriter, r *http.Request) {
	host := h.Host(r)
	handlers := make([]string, len(h.paths))

	for i, h := range h.paths {
		handlers[i] = host + h.path
	}

	indexTmpl := template.Must(template.ParseFS(templates, "templates/index.html.tmpl"))
	if err := indexTmpl.Execute(w, struct {
		Host     string
		Handlers []string
	}{
		Host:     host,
		Handlers: handlers,
	}); err != nil {
		http.Error(w, "cannot render the page", http.StatusInternalServerError)
	}
}

func (h *VanityHandler) Host(r *http.Request) string {
	host := h.host
	if host == "" {
		host = DefaultHost(r)
	}

	return host
}

func (pset PathConfigSet) Len() int {
	return len(pset)
}

func (pset PathConfigSet) Less(i, j int) bool {
	return pset[i].path < pset[j].path
}

func (pset PathConfigSet) Swap(i, j int) {
	pset[i], pset[j] = pset[j], pset[i]
}

func (pset PathConfigSet) Find(path string) (pc *PathConfig, subpath string) {
	// Fast path with binary search to retrieve exact matches
	// e.g. given pset ["/", "/abc", "/xyz"], path "/def" won't match.
	i := sort.Search(len(pset), func(i int) bool {
		return pset[i].path >= path
	})

	if i < len(pset) && pset[i].path == path {
		return &pset[i], ""
	}

	if i > 0 && strings.HasPrefix(path, pset[i-1].path+"/") {
		return &pset[i-1], path[len(pset[i-1].path)+1:]
	}

	// Slow path, now looking for the longest prefix/shortest subpath i.e.
	// e.g. given pset ["/", "/abc/", "/abc/def/", "/xyz"/]
	//  * query "/abc/foo" returns "/abc/" with a subpath of "foo"
	//  * query "/x" returns "/" with a subpath of "x"
	lenShortestSubpath := len(path)

	var bestMatchConfig *PathConfig

	// After binary search with the >= lexicographic comparison,
	// nothing greater than i will be a prefix of path.
	max := i
	for i := 0; i < max; i++ {
		ps := pset[i]

		if len(ps.path) >= len(path) {
			// We previously didn't find the path by search, so any
			// route with equal or greater length is NOT a match.
			continue
		}

		sSubpath := strings.TrimPrefix(path, ps.path)

		if len(sSubpath) < lenShortestSubpath {
			subpath = sSubpath
			lenShortestSubpath = len(sSubpath)
			bestMatchConfig = &pset[i]
		}
	}

	return bestMatchConfig, subpath
}

func NewVanityHandler(config []byte) (*VanityHandler, error) {
	var parsed VanityConfig

	if err := yaml.Unmarshal(config, &parsed); err != nil {
		return nil, err
	}

	handler := &VanityHandler{host: parsed.Host}
	cacheAge := int64(86400) // 24 hours (in seconds)

	if parsed.CacheAge != nil {
		cacheAge = *parsed.CacheAge
		if cacheAge < 0 {
			return nil, errors.New("cache_max_age is negative")
		}
	}

	handler.CacheControl = fmt.Sprintf("public, max-age=%d", cacheAge)

	for path, e := range parsed.Paths {
		pc := PathConfig{
			path:    strings.TrimSuffix(path, "/"),
			repo:    e.Repo,
			display: e.Display,
			vcs:     e.VCS,
		}

		switch {
		case e.Display != "":
			// Already filled in.
		case strings.HasPrefix(e.Repo, "https://github.com/"):
			pc.display = fmt.Sprintf("%v %v/tree/master{/dir} %v/blob/master{/dir}/{file}#L{line}", e.Repo, e.Repo, e.Repo)
		case strings.HasPrefix(e.Repo, "https://bitbucket.org"):
			pc.display = fmt.Sprintf("%v %v/src/default{/dir} %v/src/default{/dir}/{file}#{file}-{line}", e.Repo, e.Repo, e.Repo)
		}

		switch {
		case e.VCS != "":
			// Already filled in.
			if e.VCS != "bzr" && e.VCS != "git" && e.VCS != "hg" && e.VCS != "svn" {
				return nil, fmt.Errorf("configuration for %v: unknown VCS %s", path, e.VCS)
			}
		case strings.HasPrefix(e.Repo, "https://github.com/"):
			pc.vcs = "git"
		default:
			return nil, fmt.Errorf("configuration for %v: cannot infer VCS from %s", path, e.Repo)
		}

		handler.paths = append(handler.paths, pc)
	}

	sort.Sort(handler.paths)

	return handler, nil
}

func DefaultHost(r *http.Request) string {
	return r.Host
}
