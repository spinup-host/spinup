package main

import (
	"embed"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/rs/cors"
	"github.com/spinup-host/api"
	"github.com/spinup-host/config"
	"github.com/spinup-host/metrics"
	"gopkg.in/yaml.v3"
)

func init() {
	file, err := os.Open("config.yaml")
	fatal(err)
	defer file.Close()
	d := yaml.NewDecoder(file)
	err = d.Decode(&config.Cfg)
	fatal(err)
	signBytes, err := ioutil.ReadFile(config.Cfg.Common.ProjectDir + "/app.rsa")
	fatal(err)

	config.Cfg.SignKey, err = jwt.ParseRSAPrivateKeyFromPEM(signBytes)
	fatal(err)
	verifyBytes, err := ioutil.ReadFile(config.Cfg.Common.ProjectDir + "/app.rsa.pub")
	fatal(err)

	config.Cfg.VerifyKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
	fatal(err)
	log.Println("INFO: initial validations successful")
}

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
	mux.HandleFunc("/metrics", metrics.HandleMetrics)
	mux.HandleFunc("/createbackup", api.CreateBackup)
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"https://app.spinup.host", "http://localhost:3000"},
		AllowedHeaders: []string{"authorization", "content-type"},
	})
	err := http.ListenAndServe(":4434", c.Handler(mux))
	if err != nil {
		log.Fatalf("FATAL: starting server %v", err)
	}
}

//go:embed templates/*
var DockerTempl embed.FS

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
