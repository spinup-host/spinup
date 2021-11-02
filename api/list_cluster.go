package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/spinup-host/config"
)

func ListCluster(w http.ResponseWriter, req *http.Request) {
	log.Println("listcluster")
	if (*req).Method != "GET" {
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}
	authHeader := req.Header.Get("Authorization")
	var err error
	config.Cfg.UserID, err = config.ValidateToken(authHeader)
	if err != nil {
		log.Printf("error validating token %v", err)
		http.Error(w, "error validating token", 500)
	}
	dbPath := config.Cfg.Common.ProjectDir + "/" + config.Cfg.UserID
	clusterInfos := ReadClusterInfo(dbPath, config.Cfg.UserID)
	clusterByte, err := json.Marshal(clusterInfos)
	if err != nil {
		log.Printf("ERROR: marshalling clusterInfos %v", err)
		http.Error(w, "Internal server error ", 500)
		return
	}
	w.Write(clusterByte)
}

type clusterInfo struct {
	ID        int    `json:"id"`
	ClusterID string `json:"cluster_id"`
	Name      string `json:"name"`
	Port      int    `json:"port"`
	Username  string `json:"username"`
}

func ReadClusterInfo(path, dbName string) []clusterInfo {
	dsn := path + "/" + dbName + ".db"
	if _, err := os.Stat(dsn); errors.Is(err, fs.ErrNotExist) {
		log.Printf("INFO: no sqlite database")
		return nil
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query("select clusterId, name, port from clusterInfo")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var clusterIds []string
	var clusterInfos []clusterInfo
	var cluster clusterInfo
	for rows.Next() {
		err = rows.Scan(&cluster.ClusterID, &cluster.Name, &cluster.Port)
		if err != nil {
			log.Fatal(err)
		}
		clusterInfos = append(clusterInfos, cluster)
	}
	fmt.Println(clusterIds)
	return clusterInfos
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
	var err error
	config.Cfg.UserID, err = config.ValidateToken(authHeader)
	if err != nil {
		log.Printf("error validating token %v", err)
		respond(http.StatusInternalServerError, w, map[string]interface{}{
			"message": "could not validate token",
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

func getClusterFromDb(clusterId string) (clusterInfo, error) {
	var ci clusterInfo
	path := config.Cfg.Common.ProjectDir + "/" + config.Cfg.UserID
	dsn := path + "/" + config.Cfg.UserID + ".db"
	if _, err := os.Stat(dsn); errors.Is(err, fs.ErrNotExist) {
		return ci, err
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatal(err)
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
