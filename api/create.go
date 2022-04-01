package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/spinup-host/spinup/config"
	"github.com/spinup-host/spinup/internal/dockerservice"
	"github.com/spinup-host/spinup/internal/metastore"
	"github.com/spinup-host/spinup/internal/monitoring"
	"github.com/spinup-host/spinup/internal/postgres"
	"github.com/spinup-host/spinup/misc"
	_ "modernc.org/sqlite"
)

func Hello(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "hello !! Welcome to spinup \n")
}

func CreateService(w http.ResponseWriter, req *http.Request) {
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
	log.Printf("%d %d", s.Version.Maj, s.Version.Min)
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
	s.DockerNetwork = fmt.Sprintf("%s_default", s.Db.Name) // following docker-compose naming format for compatibility
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
	postgresContainer, err := postgres.NewPostgresContainer(image, s.Db.Name, s.Db.Username, s.Db.Password, s.Db.Port)
	if err != nil {
		log.Printf("ERROR: creating new docker service for %s %v", s.UserID, err)
		http.Error(w, "Error creating postgres docker service", 500)
		return
	}
	body, err := postgresContainer.Start(dockerClient)
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
		target := monitoring.Target{
			DockerNetwork: s.DockerNetwork,
			ContainerName: postgresContainer.Name,
			UserName:      s.Db.Username,
			Password:      s.Db.Password,
		}
		_, err := target.Enable()
		if err != nil {
			log.Printf("ERROR: enabling monitoring %v", err)
			http.Error(w, "Error enabling monitoring", 500)
			return
		}
	}
}
