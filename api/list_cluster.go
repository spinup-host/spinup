package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/spinup-host/config"
	"github.com/spinup-host/internal/metastore"
	"github.com/spinup-host/misc"

	_ "modernc.org/sqlite"
)

func ListCluster(w http.ResponseWriter, req *http.Request) {
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
	path := config.Cfg.Common.ProjectDir + "/" + config.Cfg.UserID + "/" + config.Cfg.UserID + ".db"
	db, err := metastore.NewDb(path)
	if err != nil {
		misc.ErrorResponse(w, "error accessing sqlite database", 500)
		return
	}
	clustersInfo, err := metastore.ClustersInfo(db)
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

func GetCluster(w http.ResponseWriter, r *http.Request) {
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

	ci, err := getClusterFromDb(clusterId)
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

func getClusterFromDb(clusterId string) (config.ClusterInfo, error) {
	var ci config.ClusterInfo
	path := config.Cfg.Common.ProjectDir + "/" + config.Cfg.UserID
	dsn := path + "/" + config.Cfg.UserID + ".db"
	if _, err := os.Stat(dsn); errors.Is(err, fs.ErrNotExist) {
		return ci, err
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return ci, err
	}
	defer db.Close()

	query := `SELECT id, clusterId, name, username, port FROM clusterInfo WHERE clusterId = ? LIMIT 1`
	err = db.QueryRow(query, clusterId).Scan(
		&ci.ID,
		&ci.ClusterID,
		&ci.Name,
		&ci.Username,
		&ci.Port,
	)
	return ci, err
}
