package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spinup-host/api"
	"github.com/spinup-host/config"
	"github.com/spinup-host/metrics"

	"github.com/golang-jwt/jwt"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	cfgFile string
	uiPath  string
	apiOnly bool

	apiPort = ":4434"
	uiPort  = ":3000"
)

func apiHandler() http.Handler {
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
	mux.HandleFunc("/cluster", api.GetCluster)
	mux.HandleFunc("/metrics", metrics.HandleMetrics)
	mux.HandleFunc("/createbackup", api.CreateBackup)
	mux.HandleFunc("/altauth", api.AltAuth)
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"https://app.spinup.host", "http://localhost:3000"},
		AllowedHeaders: []string{"authorization", "content-type", "x-api-key"},
	})

	return c.Handler(mux)
}

func uiHandler() http.Handler {
	fs := http.FileServer(http.Dir(uiPath))
	http.Handle("/", fs)

	return http.DefaultServeMux
}

func startCmd() *cobra.Command {
	sc := &cobra.Command{
		Use:   "start",
		Short: "start the spinup API and frontend servers",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println(fmt.Sprintf("INFO: using config file: %s", cfgFile))
			if err := validateConfig(cfgFile); err != nil {
				log.Fatalf("FATAL: validating config: %v", err)
			}
			log.Println("INFO: initial validations successful")

			apiListener, err := net.Listen("tcp", apiPort)
			if err != nil {
				log.Fatalf("FATAL: starting API server %v", err)
			}
			apiServer := &http.Server{
				Handler: apiHandler(),
			}
			defer stop(apiServer)

			stopCh := make(chan os.Signal, 1)
			go func() {
				log.Println(fmt.Sprintf("INFO: starting Spinup API on port %s", apiPort))
				apiServer.Serve(apiListener)
			}()

			if apiOnly == false {
				uiListener, err := net.Listen("tcp", uiPort)
				if err != nil {
					log.Fatalf("FATAL: starting UI server %v", err)
				}

				uiServer := &http.Server{
					Handler: uiHandler(),
				}
				go func() {
					log.Println(fmt.Sprintf("INFO: starting Spinup UI on port %s", uiPort))
					uiServer.Serve(uiListener)
				}()
				defer stop(uiServer)
			}

			signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)
			log.Println(fmt.Sprint(<-stopCh))
			log.Println("stopping spinup apiServer")
		},
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("FATAL: obtaining home directory: %v", err)
	}
	sc.Flags().StringVar(&cfgFile, "config",
		fmt.Sprintf("%s/.local/spinup/config.yaml", home), "Path to spinup configuration")
	sc.Flags().StringVar(&uiPath, "ui-path",
		fmt.Sprintf("%s/.local/spinup/spinup-dash", home), "Path to spinup frontend")
	sc.Flags().BoolVar(&apiOnly, "api-only", false, "Only run the API server (without the UI server). Useful for development")

	return sc
}

func validateConfig(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	d := yaml.NewDecoder(file)
	if err = d.Decode(&config.Cfg); err != nil {
		return err
	}

	signBytes, err := ioutil.ReadFile(config.Cfg.Common.ProjectDir + "/app.rsa")
	if err != nil {
		return err
	}

	if config.Cfg.SignKey, err = jwt.ParseRSAPrivateKeyFromPEM(signBytes); err != nil {
		return err
	}

	verifyBytes, err := ioutil.ReadFile(config.Cfg.Common.ProjectDir + "/app.rsa.pub")
	if err != nil {
		return err
	}

	if config.Cfg.VerifyKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyBytes); err != nil {
		return err
	}

	return nil
}

func stop(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) //nolint
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Can't stop Spinup API correctly: %v", err)
	}
}
