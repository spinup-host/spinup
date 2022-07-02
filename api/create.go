package api

import (
	"encoding/json"
	"github.com/spinup-host/spinup/config"
	"github.com/spinup-host/spinup/internal/service"
	"github.com/spinup-host/spinup/misc"
	"github.com/spinup-host/spinup/utils"
	"go.uber.org/zap"
	"io/ioutil"
	"log"
	_ "modernc.org/sqlite"
	"net/http"
)

func (c ClusterHandler) CreateService(w http.ResponseWriter, req *http.Request) {
	if (*req).Method != "POST" {
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}
	authHeader := req.Header.Get("Authorization")
	apiKeyHeader := req.Header.Get("x-api-key")

	_, err := config.ValidateUser(authHeader, apiKeyHeader)
	if err != nil {
		log.Printf(err.Error())
		http.Error(w, "error validating user", http.StatusUnauthorized)
		return
	}
	var s service.ServiceInfo
	byteArray, err := ioutil.ReadAll(req.Body)
	if err != nil {
		utils.Logger.Error("error reading request body", zap.Error(err))
		http.Error(w,"error reading request body", http.StatusInternalServerError)
		return
	}
	err = json.Unmarshal(byteArray, &s)
	if err != nil {
		utils.Logger.Error("parsing request", zap.Error(err))
		http.Error(w,"error reading request body", http.StatusBadRequest)
		return
	}

	if s.Db.Type != "postgres" {
		utils.Logger.Error("unsupported database type")
		http.Error(w, "provided database type is not supported", http.StatusBadRequest)
		return
	}
	s.Architecture = config.Cfg.Common.Architecture

	if err := c.svc.CreateService(req.Context(), &s); err != nil {
		utils.Logger.Error("failed to add create service", zap.Error(err))
	}

	serviceResponse := config.ClusterInfo{
		Host:       "localhost",
		ClusterID:  s.Db.ID,
		Name:       s.Db.Name,
		Port:       s.Db.Port,
		Username:   s.Db.Username,
		Password:   s.Db.Password,
		MajVersion: int(s.Version.Maj),
		MinVersion: int(s.Version.Min),
	}
	jsonBody, err := json.Marshal(serviceResponse)
	if err != nil {
		log.Printf("ERROR: marshalling service response struct serviceResponse %v", err)
		misc.ErrorResponse(w, "Internal server error ", 500)
	} else {
		w.Header().Set("Content-type", "application/json")
		w.Write(jsonBody)
	}
	return
}
