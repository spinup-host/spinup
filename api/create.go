package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/spinup-host/internal"
)

var authToken, zoneID, projectDir, architecture string
var api *cloudflare.API

func init() {
	var ok bool
	var err error
	if projectDir, ok = os.LookupEnv("SPINUP_PROJECT_DIR"); !ok {
		log.Fatalf("FATAL: getting environment variable SPINUP_PROJECT_DIR")
	}
	if architecture, ok = os.LookupEnv("ARCHITECTURE"); !ok {
		log.Fatalf("FATAL: getting environment variable ARCHITECTURE")
	}
	if authToken, ok = os.LookupEnv("CF_AUTHORIZATION_TOKEN"); !ok {
		log.Fatalf("FATAL: getting environment variable CF_AUTHORIZATION_TOKEN")
	}
	if zoneID, ok = os.LookupEnv("CF_ZONE_ID"); !ok {
		log.Fatalf("FATAL: getting environment variable CF_ZONE_ID")
	}
	api, err = cloudflare.NewWithAPIToken(authToken)
	if err != nil {
		log.Fatalf("FATAL: creating new cloudflare client %v", err)
	}
	log.Println("INFO: initial validations successful")
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
	enableCors(&w)
	if (*req).Method == "OPTIONS" {
		return
	}
	if (*req).Method != "POST" {
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
		return
	}
	if err = startService(s); err != nil {
		log.Printf("ERROR: starting service for %s %v", s.UserID, err)
		http.Error(w, "Error starting service", 500)
		return
	}
	log.Printf("INFO: created service for user %s", s.UserID)
	err = connectService(s)
	if err != nil {
		log.Printf("ERROR: connecting service for %s %v", s.UserID, err)
		http.Error(w, "Error connecting service", 500)
		return
	}
	err = internal.UpdateTunnelClient()
	if err != nil {
		log.Printf("ERROR: updating tunnel client for %s %v", s.UserID, err)
		http.Error(w, "Error updating tunnel client", 500)
		return
	}
	w.WriteHeader(http.StatusOK)
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

func startService(s service) error {
	err := ValidateSystemRequirements()
	if err != nil {
		return err
	}
	err = ValidateDockerCompose(projectDir + "/" + s.UserID)
	if err != nil {
		return err
	}
	cmd := exec.Command("docker-compose", "-f", projectDir+"/"+s.UserID+"/docker-compose.yml", "up", "-d")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func ValidateDockerCompose(path string) error {
	cmd := exec.Command("docker-compose", "-f", path+"/docker-compose.yml", "config")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("validating docker-compose file %v", err)
	}
	return nil
}

func ValidateSystemRequirements() error {
	cmd := exec.Command("which", "docker-compose")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker-compose doesn't exist %v", err)
	}
	cmd = exec.Command("which", "docker")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func connectService(s service) error {
	_, err := api.CreateDNSRecord(context.Background(), zoneID, cloudflare.DNSRecord{
		Type:    "A",
		Name:    s.UserID,
		Content: "34.203.202.32",
	})
	if err != nil {
		return err
	}
	log.Printf("INFO: DNS record created for %s ", s.UserID)
	return nil
}

func enableCors(w *http.ResponseWriter) {
	// TODO: to remove the wildcard and control it to specific host
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
}
