package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

var accountID, zoneID, projectDir, architecture string

func init() {
	var ok bool
	if projectDir, ok = os.LookupEnv("SPINUP_PROJECT_DIR"); !ok {
		log.Fatalf("FATAL: getting environment variable SPINUP_PROJECT_DIR")
	}
	if architecture, ok = os.LookupEnv("ARCHITECTURE"); !ok {
		log.Fatalf("FATAL: getting environment variable ARCHITECTURE")
	}
}

type service struct {
	Name     string
	Duration time.Duration
	Resource map[string]interface{}
	UserID   string
	// one of arm64v8 or arm32v7 or amd64
	Architecture string
	Port         uint
}

func Hello(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "hello !! Welcome to spinup \n")
}

func CreateService(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}
	var s service
	byteArray, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Fatalf("fatal: reading from readall body %v", req.Body)
	}
	err = json.Unmarshal(byteArray, &s)
	if err != nil {
		log.Fatalf("fatal: reading from readall body %v", req.Body)
	}
	if s.Name != "postgres" {
		fmt.Fprintf(w, "currently we don't support %s", s.Name)
		return
	}
	s.Port = 5432
	s.Architecture = architecture
	if err = prepareService(s); err != nil {
		log.Printf("ERROR: preparing service for %s %v", s.UserID, err)
		http.Error(w, "Error preparing service", 500)
	}
	return
}

func prepareService(s service) error {
	err := os.Mkdir(projectDir+"/"+s.UserID, 0755)
	if err != nil {
		return fmt.Errorf("ERROR: creating project directory at %s", projectDir+"/"+s.UserID)
	}
	if err := createDockerComposeFile(projectDir+"/"+s.UserID+"/", s); err != nil {
		return fmt.Errorf("ERROR: creating service docker-compose file %v", err)
	}
	return nil
}
