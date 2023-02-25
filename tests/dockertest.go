package tests

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"strings"

	ds "github.com/spinup-host/spinup/internal/dockerservice"
)

// DockerTest wraps docker service to be used in tests
type DockerTest struct {
	ds.Docker
}

func NewDockerTest(ctx context.Context, networkName string) (DockerTest, error) {
	dc, err := ds.NewDocker(networkName)
	if err != nil {
		return DockerTest{}, err
	}

	_, err = dc.CreateNetwork(ctx)
	if err != nil {
		return DockerTest{}, errors.Wrap(err, "create network")
	}
	return DockerTest{
		Docker: dc,
	}, nil
}

// Cleanup removes all containers and volumes in the docker network, and removes the network itself.
func (dt DockerTest) Cleanup() error {
	ctx := context.Background()
	filter := filters.NewArgs()
	filter.Add("network", dt.NetworkName)

	containers, err := dt.Cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: filter})
	if err != nil {
		return errors.Wrap(err, "list containers")
	}

	var cleanupErr error
	for _, c := range containers {
		if err = dt.Cli.ContainerStop(ctx, c.ID, container.StopOptions{Timeout: nil}); err != nil {
			if strings.Contains(err.Error(), "No such container") {
				continue
			}
			cleanupErr = multierr.Append(cleanupErr, errors.Wrap(err, "stop container"))
		}
		if err = dt.Cli.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{}); err != nil {
			if strings.Contains(err.Error(), "No such container") {
				continue
			}
			cleanupErr = multierr.Append(cleanupErr, errors.Wrap(err, "remove container"))
		}

		// cleanup its volumes
		for _, mount := range c.Mounts {
			if mount.Type == "volume" {
				if err = dt.Cli.VolumeRemove(ctx, mount.Name, true); err != nil {
					cleanupErr = multierr.Append(cleanupErr, errors.Wrap(err, "remove volume"))
				}
			}
		}
	}

	if err = dt.Cli.NetworkRemove(ctx, dt.NetworkName); err != nil {
		cleanupErr = multierr.Append(cleanupErr, errors.Wrap(err, "remove network"))
	}
	return nil
}
