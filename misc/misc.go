package misc

import (
	"bytes"
	"net/http"
	"os/exec"
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
