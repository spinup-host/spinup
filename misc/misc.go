package misc

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/spinup-host/config"
)

func minMax(array []int) (int, int) {
	var max int = array[0]
	var min int = array[0]
	for _, value := range array {
		if max < value {
			max = value
		}
		if min > value {
			min = value
		}
	}
	return min, max
}

func Portcheck() (int, error) {
	min, endingPort := minMax(config.Cfg.Common.Ports)
	for port := min; port <= endingPort; port++ {
		target := fmt.Sprintf("%s:%d", "localhost", port)
		conn, err := net.DialTimeout("tcp", target, 3*time.Second)
		if err == nil {
			// we were able to connect, post is already used
			log.Printf("INFO: port %d in use", port)
			continue
		} else {
			if strings.Contains(err.Error(), "connect: connection refused") {
				// could not reach port (probably because port is not in use)
				log.Printf("INFO: port %d is unused", port)
				return port, nil
			} else {
				// could not reach port because of some other error
				log.Printf("INFO: failed to reach port %d and checking next port: %d", port, port+1)
			}
			defer conn.Close()
		}
	}
	log.Printf("WARN: all allocated ports are occupied")
	return 0, fmt.Errorf("error all allocated ports are occupied")
}

func GetContainerIdByName(name string) (string, error) {
	name = "name=" + name
	command := "docker ps -f name=" + name + " --format '{{.ID}}'"
	cmd := exec.Command("/bin/bash", "-c", command)
	// trying to directly run docker is not working correctly. ref https://stackoverflow.com/questions/53640424/exit-code-125-from-docker-when-trying-to-run-container-programmatically
	// cmd := exec.Command("docker", "ps", "-f", name, "--format '{{.ID}}'")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return string(out.String()), nil
}

// SliceContainsString returns true if str present in s
func SliceContainsString(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func ErrorResponse(w http.ResponseWriter, msg string, statuscode int) {
	w.WriteHeader(statuscode)
	w.Write([]byte(msg))
}

func StringToDockerEnvVal(key, val string) string {
	keyval := key + "=" + val
	return keyval
}
