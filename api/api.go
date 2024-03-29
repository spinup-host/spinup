package api

import (
	"encoding/json"
	"log"
	"net/http"

	"go.uber.org/zap"

	"github.com/spinup-host/spinup/config"
)

type ClusterHandler struct {
	svc       clusterService
	logger    *zap.Logger
	appConfig config.Configuration
}

func NewClusterHandler(clusterService clusterService, cfg config.Configuration, logger *zap.Logger) (ClusterHandler, error) {
	return ClusterHandler{
		svc:       clusterService,
		logger:    logger,
		appConfig: cfg,
	}, nil
}

// respond converts its data parameter to JSON and send it as an HTTP response.
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
