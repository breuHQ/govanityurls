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

package main

import (
	"embed"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	//go:embed static
	static embed.FS
)

func main() {
	var configPath string

	switch len(os.Args) {
	case 1:
		configPath = "vanity.yaml"
	case 2:
		configPath = os.Args[1]
	default:
		log.Fatal("usage: govanityurls [CONFIG]")
	}

	vanity, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}

	handler, err := NewVanityHandler(vanity)
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/", handler)
	http.Handle("/favicon.ico", http.HandlerFunc(favico))
	http.Handle("/healthz", http.HandlerFunc(healthz))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Listening on 0.0.0.0:%s", port)

	server := &http.Server{
		Addr:              "0.0.0.0:" + port,
		Handler:           LoggingHandler(os.Stdout, http.DefaultServeMux),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func favico(w http.ResponseWriter, r *http.Request) {
	f, err := static.ReadFile("static/favicon.ico")
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	}

	w.Header().Set("Content-Type", "image/x-icon")
	_, _ = w.Write(f)
}
