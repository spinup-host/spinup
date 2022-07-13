package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"

	"github.com/spinup-host/spinup/config"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

func (c ClusterHandler) ListCluster(w http.ResponseWriter, req *http.Request) {
	if (*req).Method != "GET" {
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}
	authHeader := req.Header.Get("Authorization")
	apiKeyHeader := req.Header.Get("x-api-key")
	var err error
	_, err = config.ValidateUser(authHeader, apiKeyHeader)
	if err != nil {
		c.logger.Error("validating user", zap.Error(err))
		http.Error(w, "error validating user", http.StatusUnauthorized)
		return
	}
	clustersInfo, err := c.svc.ListClusters(req.Context())
	if err != nil {
		c.logger.Error("reading from clusterInfo table: ", zap.Error(err))
		http.Error(w, "reading from clusterInfo", http.StatusUnauthorized)
		return
	}
	clusterByte, err := json.Marshal(clustersInfo)
	if err != nil {
		c.logger.Error("parsing cluster info", zap.Error(err))
		http.Error(w, "Internal server error ", 500)
		return
	}
	w.Write(clusterByte)
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
	_, err = config.ValidateUser(authHeader, apiKeyHeader)
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
