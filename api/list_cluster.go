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
	clusterIds := ReadClusterId(dbPath, userId)
	clusterByte, err := json.Marshal(clusterIds)
	if err != nil {
		log.Printf("ERROR: marshalling clusterIds %v", err)
		http.Error(w, "Internal server error ", 500)
		return
	}
	w.Write(clusterByte)
}

func ReadClusterId(path, dbName string) []string {
	db, err := sql.Open("sqlite3", path+"/"+dbName+".db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query("select clusterId from clusterInfo")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var clusterId string
	var clusterIds []string
	for rows.Next() {
		err = rows.Scan(&clusterId)
		if err != nil {
			log.Fatal(err)
		}
		clusterIds = append(clusterIds, clusterId)
	}
	fmt.Println(clusterIds)
	return clusterIds
}
