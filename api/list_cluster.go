package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/spinup-host/spinup/internal/monitor"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"

	"github.com/spinup-host/spinup/config"
	"github.com/spinup-host/spinup/internal/metastore"

	_ "modernc.org/sqlite"
)

type ClusterHandler struct {
	db      metastore.Db
	monitor *monitor.Runtime
}

func NewClusterHandler(monitor *monitor.Runtime) (ClusterHandler, error) {
	path := filepath.Join(config.Cfg.Common.ProjectDir, "metastore.db")
	db, err := metastore.NewDb(path)
	if err != nil {
		return ClusterHandler{}, err
	}
	return ClusterHandler{
		db:      db,
		monitor: monitor,
	}, nil
}

func (c ClusterHandler) ListCluster(w http.ResponseWriter, req *http.Request) {
	log.Println("listcluster")
	if (*req).Method != "GET" {
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}
	authHeader := req.Header.Get("Authorization")
	apiKeyHeader := req.Header.Get("x-api-key")
	var err error
	config.Cfg.UserID, err = config.ValidateUser(authHeader, apiKeyHeader)
	if err != nil {
		log.Println("ERROR: validating user: ", err)
		http.Error(w, "error validating user", http.StatusUnauthorized)
		return
	}
	clustersInfo, err := metastore.AllClusters(c.db)
	if err != nil {
		log.Println("ERROR: reading from clusterInfo table: ", err)
		http.Error(w, "reading from clusterInfo", http.StatusUnauthorized)
		return
	}
	clusterByte, err := json.Marshal(clustersInfo)
	if err != nil {
		log.Printf("ERROR: marshalling clusterInfos %v", err)
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
	config.Cfg.UserID, err = config.ValidateUser(authHeader, apiKeyHeader)
	if err != nil {
		log.Printf(err.Error())
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

	ci, err := metastore.GetClusterByID(c.db, clusterId)
	if errors.Is(err, fs.ErrNotExist) {
		log.Println(err)
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
		log.Println(err)
		respond(http.StatusInternalServerError, w, map[string]interface{}{
			"message": "could not get cluster details",
		})
		return
	}

	respond(http.StatusOK, w, map[string]interface{}{
		"data": ci,
	})
}

