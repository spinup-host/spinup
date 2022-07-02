package api

import (
	"encoding/json"
	"github.com/spinup-host/spinup/config"
	"github.com/spinup-host/spinup/internal/metastore"
	"github.com/spinup-host/spinup/misc"
	"github.com/spinup-host/spinup/utils"
	"go.uber.org/zap"
	"io/ioutil"
	"log"
	_ "modernc.org/sqlite"
	"net/http"
)

// Service is used to parse request from JSON payload
// todo merge with metastore.ClusterInfo
type Service struct {
	UserID   string
	// one of arm64v8 or arm32v7 or amd64
	Architecture string
	//Port         uint
	Db            dbCluster
	DockerNetwork string
	Version       version
}

type version struct {
	Maj uint
	Min uint
}
type dbCluster struct {
	Name     string
	ID       string
	Type     string
	Port     int
	Username string
	Password string

	Memory     int64
	CPU        int64
	Monitoring string
}


func (c ClusterHandler) CreateService(w http.ResponseWriter, req *http.Request) {
	if (*req).Method != "POST" {
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}
	authHeader := req.Header.Get("Authorization")
	apiKeyHeader := req.Header.Get("x-api-key")

	_, err := config.ValidateUser(authHeader, apiKeyHeader)
	if err != nil {
		log.Printf(err.Error())
		http.Error(w, "error validating user", http.StatusUnauthorized)
		return
	}
	var s Service

	byteArray, err := ioutil.ReadAll(req.Body)
	if err != nil {
		utils.Logger.Error("error reading request body", zap.Error(err))
		http.Error(w,"error reading request body", http.StatusInternalServerError)
		return
	}
	err = json.Unmarshal(byteArray, &s)
	if err != nil {
		utils.Logger.Error("parsing request", zap.Error(err))
		http.Error(w,"error reading request body", http.StatusBadRequest)
		return
	}

	if s.Db.Type != "postgres" {
		utils.Logger.Error("unsupported database type")
		http.Error(w, "provided database type is not supported", http.StatusBadRequest)
		return
	}
	s.Db.Port, err = misc.Portcheck()
	if err != nil {
		log.Printf("ERROR: port issue for %s %v", s.UserID, err)
		http.Error(w, "port issue", 500)
		return
	}
	s.Architecture = config.Cfg.Common.Architecture

	cluster := metastore.ClusterInfo{
		Architecture: s.Architecture,
		Type: s.Db.Type,
		Host: "localhost",
		Name: s.Db.Name,
		Username: s.Db.Username,
		Password: s.Db.Password,
		Port: s.Db.Port,
		MajVersion: int(s.Version.Maj),
		MinVersion: int(s.Version.Min),
		Monitoring: s.Db.Monitoring,
	}
	if err := c.svc.CreateService(req.Context(), &cluster); err != nil {
		utils.Logger.Error("failed to add create service", zap.Error(err))
	}

	jsonBody, err := json.Marshal(cluster)
	if err != nil {
		log.Printf("ERROR: marshalling service response struct serviceResponse %v", err)
		misc.ErrorResponse(w, "Internal server error ", 500)
	} else {
		w.Header().Set("Content-type", "application/json")
		w.Write(jsonBody)
	}
	return
}
