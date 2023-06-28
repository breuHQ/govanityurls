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
	"log"
	"net/http"
	"os"
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

	vhandler, err := NewVanityHandler(vanity)
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/", vhandler)
	http.Handle("/healthz", http.HandlerFunc(healthz))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Listening on 0.0.0.0:%s", port)
	if err := http.ListenAndServe("0.0.0.0:"+port, LoggingHandler(os.Stdout, http.DefaultServeMux)); err != nil {
		log.Fatal(err)
	}
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
