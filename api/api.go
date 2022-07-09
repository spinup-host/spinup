package api

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"

	"github.com/spinup-host/spinup/config"
	"github.com/spinup-host/spinup/internal/dockerservice"
	"github.com/spinup-host/spinup/internal/metastore"
	"github.com/spinup-host/spinup/internal/monitor"
	"github.com/spinup-host/spinup/internal/service"
	"go.uber.org/zap"
)


type ClusterHandler struct {
	db      metastore.Db
	svc service.Service
	logger *zap.Logger
}

func NewClusterHandler(client dockerservice.Docker, monitor *monitor.Runtime, logger *zap.Logger, cfg config.Configuration) (ClusterHandler, error) {
	path := filepath.Join(config.Cfg.Common.ProjectDir, "metastore.db")
	db, err := metastore.NewDb(path)
	if err != nil {
		return ClusterHandler{}, err
	}
	return ClusterHandler{
		db:      db,
		svc: service.NewService(client, db, monitor, logger, cfg),
	}, nil
}

// respond converts its data parameter to JSON and send it as an HTTP response
func respond(statusCode int, w http.ResponseWriter, data interface{}) {
	if statusCode == http.StatusNoContent {
		w.WriteHeader(statusCode)
		return
	}

	// Convert the response value to JSON.
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if _, err := w.Write(jsonData); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func Hello(w http.ResponseWriter, req *http.Request) {
	respond(200, w, map[string]string{
		"message": "Welcome to Spinup!",
	})
}
