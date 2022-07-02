package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/spinup-host/spinup/api"
	"github.com/spinup-host/spinup/config"
	"github.com/spinup-host/spinup/internal/backup"
	"github.com/spinup-host/spinup/internal/dockerservice"
	"github.com/spinup-host/spinup/internal/monitor"
	"github.com/spinup-host/spinup/metrics"
	"github.com/spinup-host/spinup/utils"
)

var (
	cfgFile string
	uiPath  string
	apiOnly bool

	apiPort = ":4434"
	uiPort  = ":3000"

	monitorRuntime *monitor.Runtime
)

func apiHandler() http.Handler {
	dockerClient, err := dockerservice.NewDocker(config.DefaultNetworkName)
	if err != nil {
		utils.Logger.Error("could not create docker client", zap.Error(err))
	}

	ch, err := api.NewClusterHandler(dockerClient, monitorRuntime)
	if err != nil {
		utils.Logger.Fatal("unable to create NewClusterHandler")
	}
	mh, err := metrics.NewMetricsHandler()
	if err != nil {
		utils.Logger.Fatal("unable to create NewClusterHandler")
	}
	rand.Seed(time.Now().UnixNano())
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", api.Hello)
	mux.HandleFunc("/createservice", ch.CreateService)
	mux.HandleFunc("/githubAuth", api.GithubAuth)
	mux.HandleFunc("/logs", api.Logs)
	mux.HandleFunc("/jwt", api.JWT)
	mux.HandleFunc("/jwtdecode", api.JWTDecode)
	mux.HandleFunc("/streamlogs", api.StreamLogs)
	mux.HandleFunc("/listcluster", ch.ListCluster)
	mux.HandleFunc("/cluster", ch.GetCluster)
	mux.HandleFunc("/metrics", mh.ServeHTTP)
	mux.HandleFunc("/createbackup", backup.CreateBackup)
	mux.HandleFunc("/altauth", api.AltAuth)
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"https://app.spinup.host", "http://localhost:3000"},
		AllowedHeaders: []string{"authorization", "content-type", "x-api-key"},
	})

	return c.Handler(mux)
}

func startCmd() *cobra.Command {
	sc := &cobra.Command{
		Use:   "start",
		Short: "start the spinup API and frontend servers",
		Run: func(cmd *cobra.Command, args []string) {
			utils.InitializeLogger("", "")
			if !isDockerdRunning(context.Background()) {
				log.Fatalf("FATAL: docker daemon is not running. Start docker daemon")
			}
			log.Println(fmt.Sprintf("INFO: Using config file: %s", cfgFile))
			if err := validateConfig(cfgFile); err != nil {
				log.Fatalf("FATAL: failed to validate config: %v", err)
			}
			log.Println("INFO: Initial Validations successful")
			utils.InitializeLogger(config.Cfg.Common.LogDir, config.Cfg.Common.LogFile)

			dockerClient, err := dockerservice.NewDocker(config.DefaultNetworkName)
			if err != nil {
				utils.Logger.Error("could not create docker client", zap.Error(err))
			}
			ctx := context.TODO()
			_, err = dockerClient.CreateNetwork(ctx, dockerClient.NetworkName, types.NetworkCreate{CheckDuplicate: true})
			if err != nil {
				if errors.Is(err, dockerservice.ErrDuplicateNetwork) {
					utils.Logger.Fatal(fmt.Sprintf("found multiple docker networks with name: '%s', remove them and restart Spinup.", dockerClient.NetworkName))
				} else {
					utils.Logger.Fatal("unable to create docker network", zap.Error(err))
				}
			}

			if config.Cfg.Common.Monitoring {
				monitorRuntime = monitor.NewRuntime(dockerClient, utils.Logger)
				if err := monitorRuntime.BootstrapServices(ctx); err != nil {
					utils.Logger.Error("could not start monitoring services", zap.Error(err))
				} else {
					utils.Logger.Info("started spinup monitoring services")
				}
			}

			apiListener, err := net.Listen("tcp", apiPort)
			if err != nil {
				utils.Logger.Fatal("failed to start listener", zap.Error(err))
			}
			apiServer := &http.Server{
				Handler: apiHandler(),
			}
			defer stop(apiServer)

			stopCh := make(chan os.Signal, 1)
			go func() {
				utils.Logger.Info("starting Spinup API ", zap.String("port", apiPort))
				if err = apiServer.Serve(apiListener); err != nil {
					utils.Logger.Fatal("failed to start API server", zap.Error(err))
				}
			}()

			if apiOnly == false {
				uiListener, err := net.Listen("tcp", uiPort)
				if err != nil {
					utils.Logger.Fatal("failed to start UI server", zap.Error(err))
					return
				}

				r := chi.NewRouter()
				r.Use(middleware.Logger)

				fs := http.FileServer(http.Dir(uiPath))
				http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/" {
						fullPath := filepath.Join(uiPath, strings.TrimPrefix(path.Clean(r.URL.Path), "/"))
						_, err := os.Stat(fullPath)
						if err != nil {
							if !os.IsNotExist(err) {
								utils.Logger.Error("could not find asset", zap.Error(err))
							}
							// Requested file does not exist so we return the default (resolves to index.html)
							r.URL.Path = "/"
						}
					}
					fs.ServeHTTP(w, r)
				})

				uiServer := &http.Server{
					Handler: http.DefaultServeMux,
				}
				go func() {
					utils.Logger.Info("starting Spinup UI", zap.String("port", uiPort))
					if err = uiServer.Serve(uiListener); err != nil {
						utils.Logger.Fatal("failed to start UI server", zap.Error(err))
					}
				}()
				defer stop(uiServer)
			}

			signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)
			log.Println(fmt.Sprint(<-stopCh))
			utils.Logger.Info(fmt.Sprint(<-stopCh))
			utils.Logger.Info("stopping spinup apiServer")

		},
	}

	home, err := os.UserHomeDir()
	if err != nil {
		utils.Logger.Fatal("obtaining home directory: ", zap.Error(err))
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
		utils.Logger.Info("Can't stop Spinup API correctly:", zap.Error(err))
	}
}

// isDockerdRunning returns true if docker daemon process is running on the host
// ref: https://docs.docker.com/config/daemon/#check-whether-docker-is-running
func isDockerdRunning(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	stdout, err := exec.CommandContext(ctx, "docker", "info").CombinedOutput()
	if err != nil {
		return false
	}
	if strings.Contains(string(stdout), "ERROR") {
		return false
	}
	return true
}
