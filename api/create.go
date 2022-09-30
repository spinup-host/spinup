package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"

	"github.com/spinup-host/spinup/config"
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

func (c ClusterHandler) CreateService(w http.ResponseWriter, req *http.Request) {
	if (*req).Method != "POST" {
		respond(http.StatusMethodNotAllowed, w, map[string]string{"message": "Invalid Method"})
		return
	}
	authHeader := req.Header.Get("Authorization")
	apiKeyHeader := req.Header.Get("x-api-key")

	_, err := config.ValidateUser(authHeader, apiKeyHeader)
	if err != nil {
		c.logger.Error(err.Error())
		respond(http.StatusUnauthorized, w, map[string]string{"message": "error validating user"})
		return
	}
	var s Cluster

	byteArray, err := io.ReadAll(req.Body)
	if err != nil {
		c.logger.Error("error reading request body", zap.Error(err))
		respond(http.StatusInternalServerError, w, map[string]string{"message": "error reading request body"})
		return
	}
	err = json.Unmarshal(byteArray, &s)
	if err != nil {
		c.logger.Error("parsing request", zap.Error(err))
		respond(http.StatusBadRequest, w, map[string]string{"message": "error reading request body"})
		return
	}

	if s.Db.Type != "postgres" {
		c.logger.Error("unsupported database type")
		respond(http.StatusBadRequest, w, map[string]string{"message": "provided database type is not supported"})
		return
	}
	port, err := misc.Portcheck()
	if err != nil {
		c.logger.Error("port issue", zap.Error(err))
		respond(http.StatusInternalServerError, w, map[string]string{"message": "port issue"})
		return
	}
	s.Architecture = config.Cfg.Common.Architecture

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
