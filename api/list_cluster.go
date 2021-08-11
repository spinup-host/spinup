package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func ListCluster(w http.ResponseWriter, req *http.Request) {
	if (*req).Method != "GET" {
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}
	userId, err := validateToken(*req)
	if err != nil {
		log.Printf("error validating token %v", err)
		http.Error(w, "error validating token", 500)
	}
	dbPath := projectDir + "/" + userId
	clusterInfos := ReadClusterInfo(dbPath, userId)
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
	db, err := sql.Open("sqlite3", path+"/"+dbName+".db")
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
