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
		host      string
		paths     PathConfigSet
		cachectrl string
	}

	PathConfigSet []PathConfig

	PathConfig struct {
		Path    string
		Repo    string
		Display string
		VCS     string
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
	pc, subpath := h.paths.find(current)

	w.Header().Set("Cache-Control", h.cachectrl)

	if pc == nil && current == "/" {
		h.index(w, r)
		return
	}

	if pc == nil {
		http.NotFound(w, r)
		return
	}

	h.vanity(pc, subpath)(w, r)
}

// index renders the index page.
func (h *VanityHandler) index(w http.ResponseWriter, r *http.Request) {
	host := h.Host(r)
	handlers := make([]string, len(h.paths))

	for i, h := range h.paths {
		handlers[i] = host + h.Path
	}

	indexTmpl := template.Must(template.ParseFS(templates, "templates/index.html.tmpl"))
	if err := indexTmpl.Execute(w, struct {
		Host     string
		Handlers []string
	}{
		Host:     host,
		Handlers: handlers,
	}); err != nil {
		http.Error(w, ErrUnableToRender.Error(), http.StatusInternalServerError)
	}
}

// vanity renders the vanity url.
func (h *VanityHandler) vanity(pc *PathConfig, subpath string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vanityTmpl := template.Must(template.ParseFS(templates, "templates/vanity.html.tmpl"))
		if err := vanityTmpl.Execute(w, VanityTemplate{
			Import:  h.Host(r) + pc.Path,
			SubPath: subpath,
			Repo:    pc.Repo,
			Display: pc.Display,
			VCS:     pc.VCS,
		}); err != nil {
			http.Error(w, ErrUnableToRender.Error(), http.StatusInternalServerError)
		}
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
	return pset[i].Path < pset[j].Path
}

func (pset PathConfigSet) Swap(i, j int) {
	pset[i], pset[j] = pset[j], pset[i]
}

func (pset PathConfigSet) find(path string) (pc *PathConfig, subpath string) {
	// Fast path with binary search to retrieve exact matches
	// e.g. given pset ["/", "/abc", "/xyz"], path "/def" won't match.
	i := sort.Search(len(pset), func(i int) bool {
		return pset[i].Path >= path
	})

	if i < len(pset) && pset[i].Path == path {
		return &pset[i], ""
	}

	if i > 0 && strings.HasPrefix(path, pset[i-1].Path+"/") {
		return &pset[i-1], path[len(pset[i-1].Path)+1:]
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

		if len(ps.Path) >= len(path) {
			// We previously didn't find the path by search, so any
			// route with equal or greater length is NOT a match.
			continue
		}

		sSubpath := strings.TrimPrefix(path, ps.Path)

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
		return nil, ErrInvalidConfig
	}

	handler := &VanityHandler{host: parsed.Host}
	cacheAge := int64(86400) // 24 hours (in seconds)

	if parsed.CacheAge != nil {
		cacheAge = *parsed.CacheAge
		if cacheAge < 0 {
			return nil, ErrCacheMaxAgeNegative
		}
	}

	handler.cachectrl = fmt.Sprintf("public, max-age=%d", cacheAge)

	for path, e := range parsed.Paths {
		pc := PathConfig{
			Path:    strings.TrimSuffix(path, "/"),
			Repo:    e.Repo,
			Display: e.Display,
			VCS:     e.VCS,
		}

		switch {
		case e.Display != "":
			// Already filled in.
		case strings.HasPrefix(e.Repo, "https://github.com/"):
			pc.Display = fmt.Sprintf("%v %v/tree/master{/dir} %v/blob/master{/dir}/{file}#L{line}", e.Repo, e.Repo, e.Repo)
		case strings.HasPrefix(e.Repo, "https://bitbucket.org"):
			pc.Display = fmt.Sprintf("%v %v/src/default{/dir} %v/src/default{/dir}/{file}#{file}-{line}", e.Repo, e.Repo, e.Repo)
		}

		switch {
		case e.VCS != "":
			// Already filled in.
			if e.VCS != "bzr" && e.VCS != "git" && e.VCS != "hg" && e.VCS != "svn" {
				return nil, NewInvalidVCSError(path, e.Repo)
			}
		case strings.HasPrefix(e.Repo, "https://github.com/"):
			pc.VCS = "git"
		default:
			return nil, NewInvalidVCSError(path, e.Repo)
		}

		handler.paths = append(handler.paths, pc)
	}

	sort.Sort(handler.paths)

	return handler, nil
}

func DefaultHost(r *http.Request) string {
	return r.Host
}
