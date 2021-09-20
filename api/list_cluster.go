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
	ClusterID string
	Name      string
	Port      int
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
