package dockerservice

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/spinup-host/spinup/misc"
)

type Docker struct {
	Cli *client.Client
}

// NewDocker returns a Docker struct
func NewDocker(opts ...client.Opt) (Docker, error) {
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		fmt.Printf("error creating client %v", err)
	}
	//TODO: Check. Does this initialize and assign
	return Docker{Cli: cli}, nil
}

// Container represents a docker container
type Container struct {
	ID      string
	Name    string
	Options types.ContainerStartOptions
	// portable docker config
	Config container.Config
	// non-portable docker config
	HostConfig    container.HostConfig
	NetworkConfig network.NetworkingConfig
	Warning       []string
}

// NewContainer returns a container with provided name, ctx.
// Rest of the fields default value does makes sense. We should look to remove those since they aren't adding any value
func NewContainer(name string, config container.Config, hostConfig container.HostConfig, networkConfig network.NetworkingConfig) Container {
	return Container{
		Name:          name,
		Config:        config,
		HostConfig:    hostConfig,
		NetworkConfig: networkConfig,
	}
}

func (d Docker) GetContainer(ctx context.Context, name string) (*Container, error) {
	listFilters := filters.NewArgs()
	listFilters.Add("name", name)
	containers, err := d.Cli.ContainerList(ctx, types.ContainerListOptions{Filters: listFilters})
	if err != nil {
		return &Container{}, fmt.Errorf("error listing containers %w", err)
	}
	for _, match := range containers {
		// TODO: name of the container has prefixed with "/"
		// I have hardcoded here; perhaps there is a better way to handle this
		if misc.SliceContainsString(match.Names, "/"+name) {
			c := &Container{
				ID:   match.ID,
				Name: name,
			}
			return c, nil
		}
	}
	return nil, nil
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

func (c *Container) Start(ctx context.Context, d Docker) (container.ContainerCreateCreatedBody, error) {
	exists, err := imageExistsLocally(context.Background(), d, c.Config.Image)
	if err != nil {
		return container.ContainerCreateCreatedBody{}, fmt.Errorf("error checking whether the image exists locally %w", err)
	}
	if !exists {
		log.Printf("INFO: docker image %s doesn't exist on the host. Will attempt to pull in the background \n", c.Config.Image)
		if err := pullImageFromDockerRegistry(d, c.Config.Image); err != nil {
			return container.ContainerCreateCreatedBody{}, fmt.Errorf("unable to pull image from docker registry %w", err)
		}
	}
	body, err := d.Cli.ContainerCreate(ctx, &c.Config, &c.HostConfig, &c.NetworkConfig, nil, c.Name)
	if err != nil {
		return container.ContainerCreateCreatedBody{}, fmt.Errorf("unable to create container with image %s %w", c.Config.Image, err)
	}
	err = d.Cli.ContainerStart(ctx, body.ID, c.Options)
	if err != nil {
		return container.ContainerCreateCreatedBody{}, fmt.Errorf("unable to start container for image %s %w", c.Config.Image, err)
	}

	c.ID = body.ID
	log.Printf("started %s container with ID: %s", c.Name, c.ID)
	return body, nil
}

// imageExistsLocally returns a boolean indicating if an image with the
// requested name exists in the local docker image store
func imageExistsLocally(ctx context.Context, d Docker, imageName string) (bool, error) {

	listFilters := filters.NewArgs()
	listFilters.Add("reference", imageName)

	imageListOptions := types.ImageListOptions{
		Filters: listFilters,
	}

	images, err := d.Cli.ImageList(ctx, imageListOptions)
	if err != nil {
		return false, err
	}

	if len(images) > 0 {

		for _, v := range images {
			_, _, err := d.Cli.ImageInspectWithRaw(ctx, v.ID)
			if err != nil {
				return false, err
			}
			return true, nil

		}
		return false, nil
	}

	return false, nil
}

func pullImageFromDockerRegistry(d Docker, image string) error {
	rc, err := d.Cli.ImagePull(context.Background(), image, types.ImagePullOptions{
		//		Platform: "linux/amd64",
	})
	if err != nil {
		return fmt.Errorf("unable to pull docker image %s %w", image, err)
	}
	defer rc.Close()
	_, err = ioutil.ReadAll(rc)
	if err != nil {
		return fmt.Errorf("unable to download docker image %s %w", image, err)
	}
	return nil
}

func removeDockerImage(d Docker, image string) error {
	_, err := d.Cli.ImageRemove(context.Background(), image, types.ImageRemoveOptions{
		Force: true,
	})
	return err
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

func (c Container) Stop(ctx context.Context, d Docker, opts types.ContainerStartOptions) error {
	timeout := 20 * time.Second
	log.Println("stopping ", c.ID)
	return d.Cli.ContainerStop(ctx, c.ID, &timeout)
}

func (c Container) Remove(ctx context.Context, d Docker) error {
	log.Println("stopping ", c.ID)
	return d.Cli.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{})
}

func CreateVolume(ctx context.Context, d Docker, opt volume.VolumeCreateBody) (types.Volume, error) {
	log.Println("INFO: volume created successfully ", opt.Name)
	return d.Cli.VolumeCreate(ctx, opt)
}

func RemoveVolume(ctx context.Context, d Docker, volumeID string) error {
	return d.Cli.VolumeRemove(ctx, volumeID, true)
}

func CreateNetwork(ctx context.Context, d Docker, name string, opt types.NetworkCreate) (types.NetworkCreateResponse, error) {
	networkResponse, err := d.Cli.NetworkCreate(ctx, name, opt)
	if err != nil {
		return types.NetworkCreateResponse{}, err
	}
	log.Println("INFO: network created successfully ", name)
	return networkResponse, nil
}

func RemoveNetwork(ctx context.Context, d Docker, networkID string) error {
	return d.Cli.NetworkRemove(ctx, networkID)
}

func NewDockerClient(ops ...client.Opt) (*client.Client, error) {
	cli, err := client.NewClientWithOpts(ops...)
	if err != nil {
		return nil, err
	}
	return cli, nil
}
