package internal

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog"
	"log"
	"os"
	"os/exec"
	"strings"
)

type LinePrefixLogger struct {
	prefix string
	logger zerolog.Logger
}

type DockerService struct
{
	DockerClient *client.Client
	Name string `yaml:"name"`
	NetworkName   string      `yaml:"network_name"`
	RestartPolicy string      `yaml:"restart"`
	Ports         map[int]int `yaml:"ports"`
	Environment map[string]string `yaml:"environment"`
	Volumes []string `yaml:"volumes"`
	Image string `yaml:"image"`
}

func (ds DockerService) buildArgs() []string {
	args := []string{"run", "--net=" + ds.NetworkName, "--name=" + ds.NetworkName + "-" + ds.Name, "--hostname=" + ds.Name}

	// Environment variables
	for name, value := range ds.Environment {
		args = append(args, "-e", name+"="+value)
	}

	// Published ports
	for dst, src := range ds.Ports {
		args = append(args, "-p", fmt.Sprintf("%d:%d", dst, src))
	}

	// Restart policy
	args = append(args, "--restart", ds.RestartPolicy)

	args = append(args, ds.Image)

	return args
}

func (ds DockerService) Start() (err error) {
	logger := zerolog.New(os.Stderr).With().Str("name", ds.Name).Timestamp().Logger()
	_, err = ds.DockerClient.NetworkCreate(context.TODO(), ds.NetworkName, types.NetworkCreate{CheckDuplicate: true})
	if err != nil {
		networkExistsError := fmt.Sprintf("network with name %s already exists", ds.NetworkName)
		if !strings.Contains(err.Error(), networkExistsError) {
			return err
		}
	}

	cmd := exec.Command("docker", ds.buildArgs()...)
	cmd.Stdout = logger
	cmd.Stderr = logger

	if err = cmd.Start(); err != nil {
		log.Print(err)
		return err
	}
	return nil
}

func NewPgExporterService(cli *client.Client, networkName, postgresUsername, postgresPassword string) DockerService {
	exporterSvc := DockerService{
		DockerClient: cli,
		Name:          "postgres_exporter",
		NetworkName:   networkName,
		RestartPolicy: "unless-stopped",
		Ports: map[int]int {
			9187: 9187,
		},
		Environment: map[string]string {
			"DATA_SOURCE_NAME": fmt.Sprintf("postgresql://%s:%s@%s:5432/postgres?sslmode=disable",
				postgresUsername,
				postgresPassword,
				networkName + "_postgres_1",
			),
		},
		Image: "quay.io/prometheuscommunity/postgres-exporter",
	}
	return exporterSvc
}