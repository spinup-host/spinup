package service

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/pkg/errors"
	"log"
	"testing"

	"github.com/spinup-host/spinup/internal/dockerservice"
)

type testDocker struct {
	dockerservice.Docker
}

func newTestDocker(networkName string) (testDocker, error) {
	dc, err := dockerservice.NewDocker(networkName)
	if err != nil {
		return testDocker{}, fmt.Errorf("could not create docker client: %s", err.Error())
	}
	return testDocker{
		Docker: dc,
	}, nil
}

// cleanup removes all containers and volumes in the docker network, and removes the network itself.
func (td testDocker) cleanup() error {
	ctx := context.Background()
	filter := filters.NewArgs()
	filter.Add("network", td.NetworkName)

	containers, err := td.Cli.ContainerList(ctx, types.ContainerListOptions{Filters: filter})
	if err != nil {
		return errors.Wrap(err, "list containers")
	}

	for _, c := range containers {
		// cleanup its volumes
		for _, mount := range c.Mounts {
			if mount.Type == "volume" {
				if err = td.Cli.VolumeRemove(ctx, mount.Name, true); err != nil {
					log.Println("failed to remove volume: ", err) // no need to return for failed volume deletion
				}
			}
		}

		if err = td.Cli.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{}); err != nil {
			return errors.Wrap(err, "remove container")
		}
	}

	if err = td.Cli.NetworkRemove(ctx, td.NetworkName); err != nil {
		return errors.Wrap(err, "remove network")
	}
	return nil
}

func TestCreateService(t *testing.T) {

}
