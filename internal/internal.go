package internal

import (
	"fmt"
	"log"
	"os"
)

// TODO: vicky this entire code to manipulate tunnel client yml should be removed.
// This is just commented for reference.

/* type ClientTunnel struct {
	Server_addr string
	Tunnels     struct {
		Api        map[string]string
		Prometheus map[string]string `tag:"omitempty"`
		Postgres   map[string]string `tag:"omitempty"`
		Postgresa  map[string]string `tag:"omitempty"`
	}
}

type NewClientTunnel struct {
	Server_addr string
	Tunnels     struct {
		Api        map[string]string
		Postgres   map[string]string
		Postgresa  map[string]string `yaml:"postgresa,omitempty"`
		Prometheus map[string]string `yaml:"prometheus,omitempty"`
	}
}

func UpdateTunnelClient(port uint) error {
	yamlbytes, err := ioutil.ReadFile("/home/pi/tunnel/.tunnel/tunnel.yml")
	if err != nil {
		return fmt.Errorf("error reading file %v", err)
	}
	var c ClientTunnel
	err = yaml.Unmarshal(yamlbytes, &c)
	if err != nil {
		log.Printf("error Unmarshal file %v", err)
	}
	nc := NewClientTunnel{
		Server_addr: c.Server_addr,
	}
	_, ok := c.Tunnels.Api["addr"]
	if ok {
		nc.Tunnels.Api = make(map[string]string, 1)
		nc.Tunnels.Api["proto"] = c.Tunnels.Api["proto"]
		nc.Tunnels.Api["addr"] = c.Tunnels.Api["addr"]
		nc.Tunnels.Api["host"] = c.Tunnels.Api["host"]
	}
	nc.Tunnels.Postgres = make(map[string]string, 1)
	nc.Tunnels.Postgres["proto"] = "tcp"
	nc.Tunnels.Postgres["addr"] = "10.0.0.244:" + fmt.Sprintf("%d", port)
	nc.Tunnels.Postgres["remote_addr"] = "0.0.0.0:" + fmt.Sprintf("%d", port)
	outYamlBytes, err := yaml.Marshal(nc)
	if err != nil {
		log.Printf("error marshalling %v", err)
	}
	ioutil.WriteFile("/home/pi/tunnel/.tunnel/tunnel.yml", outYamlBytes, 0644)
	file, _ := os.OpenFile("/home/pi/tunnel/.tunnel/tunnel.yml", os.O_RDWR, 0644)
	file.WriteAt([]byte("postgres"), 0)
	return nil
} */

func UpdateTunnelClientYml(service string, port int) error {
	filePath := "/home/pi/tunnel/.tunnel/tunnel.yml"
	fileByte, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("failed to read tunnel client file: %s", err)
		return err
	}
	fileLength := len(fileByte)
	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		log.Printf("failed to open tunnel client file: %s", err)
		return err
	}
	defer file.Close()
	bytesWritten, err := file.WriteAt([]byte("    "+service+":"), int64(fileLength))
	if err != nil {
		log.Printf("failed to write on tunnel client file: %s", err)
		return err
	}
	fileLength = fileLength + bytesWritten
	bytesWritten, err = file.WriteAt([]byte("\n        addr: 10.0.0.244:"+fmt.Sprintf("%d", port)), int64(fileLength))
	if err != nil {
		log.Printf("failed to write on tunnel client file: %s", err)
		return err
	}
	fileLength = fileLength + bytesWritten
	bytesWritten, err = file.WriteAt([]byte("\n        proto: tcp"), int64(fileLength))
	if err != nil {
		log.Printf("failed to write on tunnel client file: %s", err)
		return err
	}
	fileLength = fileLength + bytesWritten
	_, err = file.WriteAt([]byte("\n        remote_addr: 0.0.0.0:"+fmt.Sprintf("%d", port)+"\n"), int64(fileLength))
	if err != nil {
		log.Printf("failed to write on tunnel client file: %s", err)
		return err
	}
	return nil
}

/* func RestartTunnelClient() error {
	cmd := exec.Command("/bin/sh", "-c", "sudo systemctl restart tunnel-client")
	// https://stackoverflow.com/questions/18159704/how-to-debug-exit-status-1-error-when-running-exec-command-in-golang/18159705
	// To print the actual error instead of just printing the exit status
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return err
	}
	return nil
} */
