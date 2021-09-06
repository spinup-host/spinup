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
	mux.HandleFunc("/jwt", api.JWT)
	mux.HandleFunc("/jwtdecode", api.JWTDecode)
	mux.HandleFunc("/streamlogs", api.StreamLogs)
	mux.HandleFunc("/listcluster", api.ListCluster)
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"https://app.spinup.host", "localhost:3000"},
		AllowedHeaders: []string{"authorization", "content-type"},
	})
	err := http.ListenAndServe(":4434", c.Handler(mux))
	if err != nil {
		log.Fatalf("FATAL: starting server %v", err)
	}
}
