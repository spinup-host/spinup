package dockerservice

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/spinup-host/misc"
)

type Docker struct {
	Cli *client.Client
}

func NewDocker(opts ...client.Opt) (Docker, error) {
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		fmt.Printf("error creating client %v", err)
	}
	//TODO: Check. Does this initialize and assign
	return Docker{Cli: cli}, nil
}

type DockerService struct {
	DockerClient    *client.Client
	Name            string            `yaml:"name"`
	NetworkName     string            `yaml:"network_name"`
	RestartPolicy   string            `yaml:"restart"`
	Ports           map[int]int       `yaml:"ports"`
	Environment     map[string]string `yaml:"environment"`
	Volumes         []string          `yaml:"volumes"`
	Image           string            `yaml:"image"`
	RemoveContainer bool              `yaml:"remove_container"`
}

type Container struct {
	ID      string
	Name    string
	Options *types.ContainerStartOptions
	Ctx     context.Context
	// portable docker config
	Config container.Config
	// non-portable docker config
	HostConfig    container.HostConfig
	NetworkConfig network.NetworkingConfig
}

func NewContainer(name string, config container.Config, hostConfig container.HostConfig, networkConfig network.NetworkingConfig, options *types.ContainerStartOptions) *Container {
	return &Container{
		Name:          name,
		Ctx:           context.Background(),
		Config:        config,
		HostConfig:    hostConfig,
		NetworkConfig: networkConfig,
		Options:       options,
	}
}

func (d Docker) GetContainer(ctx context.Context, name string) (Container, error) {
	c := Container{}
	containers, err := d.Cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return Container{}, fmt.Errorf("error listing containers %w", err)
	}
	for _, container := range containers {
		// TODO: name of the container has prefixed with "/"
		// I have hardcoded here; perhaps there is a better way to handle this
		if misc.SliceContainsString(container.Names, "/"+name) {
			c.ID = container.ID
			c.Config.Image = container.Image
			break
		}
	}
	return c, nil
}

func (d Docker) LastContainerID(ctx context.Context) (string, error) {
	containers, err := d.Cli.ContainerList(ctx, types.ContainerListOptions{Latest: true})
	if err != nil {
		return "", err
	}
	var containerID string
	for _, container := range containers {
		containerID = container.ID
	}
	return containerID, nil
}

func (c *Container) Start(d Docker) (container.ContainerCreateCreatedBody, error) {
	body, err := d.Cli.ContainerCreate(c.Ctx, &c.Config, &c.HostConfig, &c.NetworkConfig, nil, c.Name)
	if err != nil {
		log.Println("error creating container")
		return container.ContainerCreateCreatedBody{}, err
	}
	err = d.Cli.ContainerStart(c.Ctx, body.ID, types.ContainerStartOptions{})
	if err != nil {
		log.Println("error starting container")
		return container.ContainerCreateCreatedBody{}, err
	}
	return body, nil
}

// ExecCommand executes a given bash command through execConfig and displays the output in stdout and stderr
// This function doesn't return an error for the failure of the command itself
func (c Container) ExecCommand(ctx context.Context, d Docker, execConfig types.ExecConfig) (types.IDResponse, error) {
	if c.ID == "" {
		return types.IDResponse{}, errors.New("container id is empty")
	}
	if _, err := d.Cli.ContainerInspect(ctx, c.ID); err != nil {
		return types.IDResponse{}, errors.New("container doesn't exist")
	}
	execResponse, err := d.Cli.ContainerExecCreate(ctx, c.ID, execConfig)
	if err != nil {
		return types.IDResponse{}, fmt.Errorf("creating container exec %w", err)
	}
	resp, err := d.Cli.ContainerExecAttach(ctx, execResponse.ID, types.ExecStartCheck{Tty: false})
	if err != nil {
		return types.IDResponse{}, fmt.Errorf("creating container exec attach %w", err)
	}
	defer resp.Close()
	// show the output to stdout and stderr
	if _, err := stdcopy.StdCopy(os.Stdout, os.Stderr, resp.Reader); err != nil {
		return types.IDResponse{}, fmt.Errorf("unable to copy the output of container, %w", err)
	}
	if err = d.Cli.ContainerExecStart(ctx, execResponse.ID, types.ExecStartCheck{}); err != nil {
		return types.IDResponse{}, fmt.Errorf("starting container exec %v", err)
	}
	return execResponse, nil
}

func (c *Container) Stop(d Docker, opts types.ContainerStartOptions) {
	timeout := 20 * time.Second
	d.Cli.ContainerStop(c.Ctx, c.ID, &timeout)
}

func CreateVolume(ctx context.Context, d Docker, opt volume.VolumeCreateBody) (types.Volume, error) {
	log.Println("creating volume: ", opt.Name)
	return d.Cli.VolumeCreate(ctx, opt)
}

func CreateNetwork(ctx context.Context, d Docker, name string, opt types.NetworkCreate) (types.NetworkCreateResponse, error) {
	networkResponse, err := d.Cli.NetworkCreate(ctx, name, opt)
	if err != nil {
		return types.NetworkCreateResponse{}, err
	}
	return networkResponse, nil
}

func NewDockerClient(ops ...client.Opt) (*client.Client, error) {
	cli, err := client.NewClientWithOpts(ops...)
	if err != nil {
		return nil, err
	}
	return cli, nil
}
