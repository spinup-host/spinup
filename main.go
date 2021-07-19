package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/rs/cors"
	"github.com/spinup-host/api"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", api.Hello)
	mux.HandleFunc("/createservice", api.CreateService)
	mux.HandleFunc("/githubAuth", api.GithubAuth)
	mux.HandleFunc("/logs", api.Logs)
	mux.HandleFunc("/streamlogs", api.StreamLogs)
	// TODO: remove http version
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"https://spinup.host", "http://spinup.host", "http://app.spinup.host", "https://app.spinup.host", "localhost:3000"},
	})
	err := http.ListenAndServe("localhost:443", c.Handler(mux))
	if err != nil {
		log.Fatalf("FATAL: starting server %v", err)
	}
}
