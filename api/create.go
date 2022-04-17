package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	_ "modernc.org/sqlite"

	"github.com/spinup-host/spinup/config"
	"github.com/spinup-host/spinup/internal/dockerservice"
	"github.com/spinup-host/spinup/internal/metastore"
	"github.com/spinup-host/spinup/internal/monitor"
	"github.com/spinup-host/spinup/internal/postgres"
	"github.com/spinup-host/spinup/misc"
)

func Hello(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "hello !! Welcome to spinup \n")
}

func (c ClusterHandler) CreateService(w http.ResponseWriter, req *http.Request) {
	if (*req).Method != "POST" {
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}
	authHeader := req.Header.Get("Authorization")
	apiKeyHeader := req.Header.Get("x-api-key")

	userId, err := config.ValidateUser(authHeader, apiKeyHeader)
	if err != nil {
		log.Printf(err.Error())
		http.Error(w, "error validating user", http.StatusUnauthorized)
		return
	}
	var s config.Service

	byteArray, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Fatalf("fatal: reading from readall body %v", err)
	}

	err = json.Unmarshal(byteArray, &s)
	log.Printf("%d %d %d %d", s.Version.Maj, s.Version.Min, s.Db.CPU, s.Db.Memory)
	if s.UserID == "" && apiKeyHeader != "" {
		s.UserID = "testuser"
	}

	if err != nil {
		log.Fatalf("fatal: reading from readall body %v", err)
	}

	if s.UserID != "testuser" && userId != s.UserID {
		log.Printf("user %s trying to access /createservice using userId %s", s.UserID, userId)
		http.Error(w, "userid doesn't match", http.StatusInternalServerError)
		return
	}

	if s.Db.Type != "postgres" {
		fmt.Fprintf(w, "currently we don't support %s", s.Db.Type)
		http.Error(w, "db type is currently not supported", 500)
		return
	}
	s.Db.Port, err = misc.Portcheck()
	if err != nil {
		log.Printf("ERROR: port issue for %s %v", s.UserID, err)
		http.Error(w, "port issue", 500)
		return
	}
	s.Architecture = config.Cfg.Common.Architecture
	s.DockerNetwork = "spinup_services"
	image := s.Architecture + "/" + s.Db.Type + ":" + strconv.Itoa(int(s.Version.Maj))
	if s.Version.Min > 0 {
		image += "." + strconv.Itoa(int(s.Version.Min))
	} else {
		image += ".0"
	}
	dockerClient, err := dockerservice.NewDocker()
	if err != nil {
		fmt.Printf("error creating client %v", err)
	}
	postgresContainerProp := postgres.ContainerProps{
		Name:      s.Db.Name,
		Username:  s.Db.Username,
		Password:  s.Db.Password,
		Port:      s.Db.Port,
		Memory:    s.Db.Memory,
		CPUShares: s.Db.CPU,
		Image:     image,
	}
	postgresContainer, err := postgres.NewPostgresContainer(postgresContainerProp)
	if err != nil {
		log.Printf("ERROR: creating new docker service for %s %v", s.UserID, err)
		http.Error(w, "Error creating postgres docker service", 500)
		return
	}
	body, err := postgresContainer.Start(req.Context(), dockerClient)
	if err != nil {
		log.Printf("ERROR: starting new docker service for %s %v", s.UserID, err)
		http.Error(w, "Error starting postgres docker service", 500)
		return
	}
	postgresContainer.ID = body.ID
	postgresContainer.Warning = body.Warnings
	log.Printf("INFO: created service for user %s %s", s.UserID, postgresContainer.ID)
	if err != nil {
		log.Printf("ERROR: getting container id %v", err)
		http.Error(w, "Error getting container id", 500)
		return
	}
	path := filepath.Join(config.Cfg.Common.ProjectDir, "metastore.db")
	db, err := metastore.NewDb(path)
	if err != nil {
		misc.ErrorResponse(w, "error accessing sqlite database", 500)
		return
	}
	insertSql := "insert into clusterInfo(clusterId, name, username, password, port, majVersion, minVersion) values(?, ?, ?, ?, ?, ?, ?)"
	if err := metastore.InsertServiceIntoMeta(db, insertSql, postgresContainer.ID, s.Db.Name, s.Db.Username, s.Db.Password, s.Db.Port, int(s.Version.Maj), int(s.Version.Min)); err != nil {
		log.Printf("ERROR: executing insert into cluster info table %v", err)
		misc.ErrorResponse(w, "internal server error", 500)
		return
	}

	if s.Db.Monitoring == "enable" {
		target := &monitor.Target{
			DockerNetwork: s.DockerNetwork,
			ContainerName: postgresContainer.Name,
			UserName:      s.Db.Username,
			Password:      s.Db.Password,
			Port:          s.Db.Port,
		}

		if c.monitor != nil {
			if err = c.monitor.AddTarget(req.Context(), target); err != nil {
				log.Printf("ERROR: setting up monitoring for service: %v", err)
				http.Error(w, "error enabling monitoring", 500)
			}
		} else {
			// this might take more time especially if the image doesn't exist locally, so we wrap it in a goroutine
			go func() {
				dockerClient, err := dockerservice.NewDocker()
				if err != nil {
					log.Printf("error creating client %v", err)
					return
				}
				c.monitor = monitor.NewRuntime(dockerClient)
				if err := c.monitor.BootstrapServices(); err != nil {
					log.Println(err)
				} else {
					log.Println("started monitoring services")
				}
				if err = c.monitor.AddTarget(req.Context(), target); err != nil {
					log.Printf("ERROR: setting up monitoring for service: %v", err)
				}
			}()

			// todo: find a way to send "info" messages to the client without making them an error
			log.Println("monitoring services are not running, Spinup will start them in the background")
		}
		return
	}

	serviceResponse := struct {
		HostName    string
		Port        int
		ContainerID string
	}{
		HostName:    "localhost",
		Port:        s.Db.Port,
		ContainerID: postgresContainer.ID,
	}
	jsonBody, err := json.Marshal(serviceResponse)
	if err != nil {
		log.Printf("ERROR: marshalling service response struct serviceResponse %v", err)
		http.Error(w, "Internal server error ", 500)
		return
	}

	w.Header().Set("Content-type", "application/json")
	w.Write(jsonBody)
	return
}
