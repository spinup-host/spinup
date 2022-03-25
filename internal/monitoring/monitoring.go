package monitoring

import (
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/spinup-host/spinup/internal/dockerservice"
	"github.com/spinup-host/spinup/internal/postgres"
	"github.com/spinup-host/spinup/misc"
)

// Target represents a postgres host (target) for monitoring
type Target struct {
	DockerNetwork string
	ContainerName string
	UserName      string
	Password      string
}

func (t Target) Enable() (container.ContainerCreateCreatedBody, error) {
	postgresExporterContainer, err := newPostgresExporterContainer(t)
	if err != nil {
		return container.ContainerCreateCreatedBody{}, err
	}
	dockerClient, err := dockerservice.NewDocker()
	if err != nil {
		fmt.Printf("error creating client %v", err)
		return container.ContainerCreateCreatedBody{}, err
	}
	body, err := postgresExporterContainer.Start(dockerClient)
	if err != nil {
		return container.ContainerCreateCreatedBody{}, err
	}
	return body, nil
}

func newPostgresExporterContainer(t Target) (*dockerservice.Container, error) {
	networkName := t.ContainerName + "_default"
	endpointConfig := map[string]*network.EndpointSettings{}
	endpointConfig[networkName] = &network.EndpointSettings{}
	nwConfig := network.NetworkingConfig{EndpointsConfig: endpointConfig}
	containerName := "spinup-" + t.ContainerName + "-postgres-exporter"
	image := "quay.io/prometheuscommunity/postgres-exporter"
	env := []string{}
	dsn := fmt.Sprintf("postgresql://%s:%s@%s:5432/postgres?sslmode=disable", t.UserName, t.Password, postgres.PREFIXPGCONTAINER+t.ContainerName)
	env = append(env, misc.StringToDockerEnvVal("DATA_SOURCE_NAME", dsn))
	config := container.Config{
		Image: image,
		Env:   env,
	}
	postgresExporterContainer := dockerservice.NewContainer(
		containerName,
		config,
		container.HostConfig{},
		nwConfig,
		&types.ContainerStartOptions{},
	)
	return postgresExporterContainer, nil
}
