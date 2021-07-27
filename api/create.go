package api

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/golang-jwt/jwt"
	"github.com/spinup-host/internal"
)

var authToken, zoneID, projectDir, architecture string
var api *cloudflare.API
var privKeyPath, pubKeyPath string
var (
	verifyKey *rsa.PublicKey
	signKey   *rsa.PrivateKey
)

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

	signBytes, err := ioutil.ReadFile(projectDir + "/app.rsa")
	fatal(err)

	signKey, err = jwt.ParseRSAPrivateKeyFromPEM(signBytes)
	fatal(err)
	verifyBytes, err := ioutil.ReadFile(projectDir + "/app.rsa.pub")
	fatal(err)

	verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
	fatal(err)
	log.Println("INFO: initial validations successful")
}

type service struct {
	Duration time.Duration
	UserID   string
	// one of arm64v8 or arm32v7 or amd64
	Architecture string
	//Port         uint
	Db dbCluster
}

type dbCluster struct {
	Name       string
	Type       string
	Port       uint
	MajVersion uint
	MinVersion uint
	Memory     string
	Storage    string
}

func Hello(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "hello !! Welcome to spinup \n")
}

func CreateService(w http.ResponseWriter, req *http.Request) {
	if (*req).Method != "POST" {
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}
	userId, err := validateToken(*req)
	if err != nil {
		log.Printf("error validating token %v", err)
		http.Error(w, "error validating token", 500)
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
	if s.UserID != userId {
		log.Printf("user %s trying to access /createservice using jwt userId %s", s.UserID, userId)
		http.Error(w, "userid doesn't match", http.StatusInternalServerError)
		return
	}
	if s.Db.Type != "postgres" {
		fmt.Fprintf(w, "currently we don't support %s", s.Db.Type)
		http.Error(w, "db type is currently not supported", 500)
		return
	}
	s.Db.Port = nextAvailablePort()
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
	servicePath := projectDir + "/" + s.UserID + "/" + s.Db.Name
	err := os.Mkdir(servicePath, 0755)
	if err != nil {
		return fmt.Errorf("ERROR: creating project directory at %s", servicePath)
	}
	if err := createDockerComposeFile(servicePath, s); err != nil {
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
		Name:    s.UserID + "-" + s.Db.Name,
		Content: "34.203.202.32",
	})
	if err != nil {
		return err
	}
	log.Printf("INFO: DNS record created for %s ", s.UserID+"-"+s.Db.Name)
	return nil
}

func nextAvailablePort() uint {
	var port uint
	endingPort := 5440
	for startingPort := 5432; startingPort < endingPort; startingPort++ {
		_, err := net.DialTimeout("tcp", ":"+string(startingPort), 3*time.Second)
		if err != nil {
			log.Printf("INFO: port %d already taken", startingPort)
			continue
		}
		port = uint(startingPort)
		break
	}
	return port
}

func validateToken(r http.Request) (string, error) {
	reqToken := r.Header.Get("Authorization")
	splitToken := strings.Split(reqToken, "Bearer ")
	reqToken = splitToken[1]
	userID, err := JWTToString(reqToken)
	if err != nil {
		return "", err
	}
	return userID, nil
}
