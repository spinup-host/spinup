package postgres

import (
	"context"
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
	"github.com/spinup-host/spinup/internal/dockerservice"
	"github.com/spinup-host/spinup/misc"
)

const (
	PREFIXPGCONTAINER = "spinup-postgres-"
	PGDATADIR         = "/var/lib/postgresql/data/"
)

func NewPostgresContainer(image, name, postgresUsername, postgresPassword string, port int) (postgresContainer dockerservice.Container, err error) {
	dockerClient, err := dockerservice.NewDocker()
	if err != nil {
		fmt.Printf("error creating client %v", err)
	}
	newVolume, err := dockerservice.CreateVolume(context.Background(), dockerClient, volume.VolumeCreateBody{
		Driver: "local",
		Labels: map[string]string{"purpose": "postgres data"},
		Name:   name,
	})
	if err != nil {
		return dockerservice.Container{}, err
	}
	// defer for cleaning volume removal
	defer func() {
		if err != nil {
			if errVolRemove := dockerservice.RemoveVolume(context.Background(), dockerClient, newVolume.Name); errVolRemove != nil {
				err = fmt.Errorf("error removing volume during failed service creation %w", err)
			}
		}
	}()
	networkResponse, err := dockerservice.CreateNetwork(context.Background(), dockerClient, name+"_default", types.NetworkCreate{CheckDuplicate: true})
	if err != nil {
		return dockerservice.Container{}, err
	}
	// defer for cleaning network removal
	defer func() {
		if err != nil {
			if errNetworkRemove := dockerservice.RemoveNetwork(context.Background(), dockerClient, networkResponse.ID); errNetworkRemove != nil {
				err = fmt.Errorf("error removing network during failed service creation %w", err)
			}
		}
	}()
	containerName := PREFIXPGCONTAINER + name

	newHostPort, err := nat.NewPort("tcp", strconv.Itoa(port))
	if err != nil {
		return dockerservice.Container{}, err
	}
	newContainerport, err := nat.NewPort("tcp", "5432")
	if err != nil {
		return dockerservice.Container{}, err
	}
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: newVolume.Name,
			Target: "/var/lib/postgresql/data",
		},
	}
	hostConfig := container.HostConfig{
		PortBindings: nat.PortMap{
			newContainerport: []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: newHostPort.Port(),
				},
			},
		},
		NetworkMode: "default",
		AutoRemove:  false,
		Mounts:      mounts,
	}

	endpointConfig := map[string]*network.EndpointSettings{}
	networkName := name + "_default"
	// setting key and value for the map. networkName=$dbname_default (eg: viggy_default)
	endpointConfig[networkName] = &network.EndpointSettings{}
	nwConfig := network.NetworkingConfig{EndpointsConfig: endpointConfig}
	env := []string{}
	env = append(env, misc.StringToDockerEnvVal("POSTGRES_USER", postgresUsername))
	env = append(env, misc.StringToDockerEnvVal("POSTGRES_PASSWORD", postgresPassword))

	postgresContainer = dockerservice.NewContainer(
		containerName,
		container.Config{
			Image: image,
			Env:   env,
		},
		hostConfig,
		nwConfig,
	)
	return postgresContainer, nil
}

func ReloadPostgres(d dockerservice.Docker, execpath, datapath, containerName string) error {
	execConfig := types.ExecConfig{
		User:         "postgres",
		Tty:          false,
		WorkingDir:   execpath,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{"pg_ctl", "-D", datapath, "reload"},
	}
	pgContainer, err := d.GetContainer(context.Background(), containerName)
	if err != nil {
		return fmt.Errorf("error getting container for container name %s %v", containerName, err)
	}
	if _, err := pgContainer.ExecCommand(context.Background(), d, execConfig); err != nil {
		return fmt.Errorf("error executing command %s %w", execConfig.Cmd[0], err)
	}
	return nil
}
