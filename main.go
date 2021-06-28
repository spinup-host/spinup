package main

import (
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
	http.ListenAndServe("localhost:8090", nil)
}
