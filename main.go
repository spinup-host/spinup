package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/spinup-host/api"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	var a http.HandlerFunc
	a = api.Hello
	http.HandleFunc("/hello", a)
	http.HandleFunc("/createservice", api.CreateService)
	http.HandleFunc("/githubAuth", api.GithubAuth)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello Secure World")
	})
	err := http.ListenAndServe("localhost:3001", nil)
	if err != nil {
		log.Fatalf("FATAL: starting server %v", err)
	}
}
