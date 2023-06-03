package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"

	"go.uber.org/zap"
	_ "modernc.org/sqlite"

	"github.com/spinup-host/spinup/internal/dockerservice"
	"github.com/spinup-host/spinup/internal/metastore"
	"github.com/spinup-host/spinup/misc"
)

// Cluster is used to parse request from JSON payload
// todo merge with metastore.ClusterInfo
type Cluster struct {
	UserID string `json:"userId"`
	// one of arm64v8 or arm32v7 or amd64
	Architecture string    `json:"architecture"`
	Db           dbCluster `json:"db"`
	Version      version   `json:"version"`
}

type version struct {
	Maj uint `json:"maj"`
	Min uint `json:"min"`
}
type dbCluster struct {
	Name     string `json:"name"`
	ID       string `json:"id,omitempty"`
	Type     string `json:"type"`
	Username string `json:"username"`
	Password string `json:"password"`

	Memory     int64  `json:"memory,omitempty"`
	CPU        int64  `json:"cpu,omitempty"`
	Monitoring string `json:"monitoring"`
}

// CreateCluster creates a new database with the provided parameters.
func (c ClusterHandler) CreateCluster(w http.ResponseWriter, req *http.Request) {
	if (*req).Method != "POST" {
		respond(http.StatusMethodNotAllowed, w, map[string]string{"message": "Invalid Method"})
		return
	}
	authHeader := req.Header.Get("Authorization")
	apiKeyHeader := req.Header.Get("x-api-key")
	var err error
	_, err = ValidateUser(c.appConfig, authHeader, apiKeyHeader)
	if err != nil {
		c.logger.Error("Failed to validate user", zap.Error(err))
		respond(http.StatusUnauthorized, w, map[string]string{
			"message": "Unauthorized",
		})
		return
	}

	var s Cluster

	byteArray, err := io.ReadAll(req.Body)
	if err != nil {
		c.logger.Error("error reading request body", zap.Error(err))
		respond(http.StatusInternalServerError, w, map[string]string{"message": "Error reading request body"})
		return
	}
	err = json.Unmarshal(byteArray, &s)
	if err != nil {
		c.logger.Error("parsing request", zap.Error(err))
		respond(http.StatusBadRequest, w, map[string]string{"message": "Error reading request body"})
		return
	}

	if s.Db.Type != "postgres" {
		c.logger.Error("unsupported database type")
		respond(http.StatusBadRequest, w, map[string]string{"message": "Provided database type is not supported"})
		return
	}
	port, err := misc.PortCheck(c.appConfig.Common.Ports[0], c.appConfig.Common.Ports[len(c.appConfig.Common.Ports)-1])
	if err != nil {
		c.logger.Error("port issue", zap.Error(err))
		respond(http.StatusInternalServerError, w, map[string]string{"message": "Could not find an open port"})
		return
	}
	s.Architecture = c.appConfig.Common.Architecture

	cluster := metastore.ClusterInfo{
		Architecture: s.Architecture,
		Type:         s.Db.Type,
		Host:         "localhost",
		Name:         s.Db.Name,
		Username:     s.Db.Username,
		Password:     s.Db.Password,
		Port:         port,
		MajVersion:   int(s.Version.Maj),
		MinVersion:   int(s.Version.Min),
		Monitoring:   s.Db.Monitoring,
	}

	if cluster.MajVersion <= 9 {
		respond(http.StatusBadRequest, w, map[string]string{"message": "Unsupported Postgres version. Minimum supported major version is v9"})
		return
	}
	if err := c.svc.CreateService(req.Context(), &cluster); err != nil {
		c.logger.Error("failed to add create service", zap.Error(err))
		if errors.Is(err, dockerservice.ErrDuplicateContainerName) {
			respond(http.StatusBadRequest, w, map[string]string{"message": "container with provided name already exists"})
		} else {
			respond(http.StatusBadRequest, w, map[string]string{"message": "failed to add service"})
		}
		return
	}
	respond(http.StatusOK, w, cluster)
	return
}

func (c ClusterHandler) ListCluster(w http.ResponseWriter, req *http.Request) {
	if (*req).Method != "GET" {
		respond(http.StatusMethodNotAllowed, w, map[string]string{
			"message": "Invalid Method"})
		return
	}
	authHeader := req.Header.Get("Authorization")
	apiKeyHeader := req.Header.Get("x-api-key")
	var err error
	_, err = ValidateUser(c.appConfig, authHeader, apiKeyHeader)
	if err != nil {
		c.logger.Error("validating user", zap.Error(err))
		respond(http.StatusUnauthorized, w, map[string]string{
			"message": "unauthorized",
		})
		return
	}
	clustersInfo, err := c.svc.ListClusters(req.Context())
	if err != nil {
		c.logger.Error("failed to list clusters", zap.Error(err))
		respond(http.StatusInternalServerError, w, map[string]string{
			"message": "Failed to list clusters",
		})
		return
	}
	respond(http.StatusOK, w, clustersInfo)
	return
}

func (c ClusterHandler) GetCluster(w http.ResponseWriter, r *http.Request) {
	if (*r).Method != "GET" {
		respond(http.StatusMethodNotAllowed, w, map[string]interface{}{
			"message": "method not allowed",
		})
		return
	}
	// todo (idoqo): move auth stuff to a "middleware"
	authHeader := r.Header.Get("Authorization")
	apiKeyHeader := r.Header.Get("x-api-key")
	var err error
	_, err = ValidateUser(c.appConfig, authHeader, apiKeyHeader)
	if err != nil {
		c.logger.Error("validating user", zap.Error(err))
		respond(http.StatusInternalServerError, w, map[string]interface{}{
			"message": "could not validate user",
		})
		return
	}

	clusterId := r.URL.Query().Get("cluster_id")
	if clusterId == "" {
		respond(http.StatusBadRequest, w, map[string]interface{}{
			"message": "cluster_id not present",
		})
		return
	}

	ci, err := c.svc.GetClusterByID(r.Context(), clusterId)
	if errors.Is(err, fs.ErrNotExist) {
		c.logger.Error("no database file", zap.Error(err))
		respond(http.StatusInternalServerError, w, map[string]interface{}{
			"message": "sqlite database was not found",
		})
	}

	if err == sql.ErrNoRows {
		respond(http.StatusNotFound, w, map[string]interface{}{
			"message": "no cluster found with matching id",
		})
		return
	} else if err != nil {
		c.logger.Error("getting cluster info")
		respond(http.StatusInternalServerError, w, map[string]interface{}{
			"message": "could not get cluster details",
		})
		return
	}

	respond(http.StatusOK, w, map[string]interface{}{
		"data": ci,
	})
}
