package service

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"path/filepath"
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
	t.Parallel()
	testID := uuid.New().String()
	dc, err := newTestDocker(testID)
	require.NoError(t, err)

	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)

	path := filepath.Join(tmpDir, testID+".db")
	defer func(name string) {
		_ = os.Remove(name)
	}(path)

	require.NoError(t, err)


	t.Run("without monitoring", func(t *testing.T) {
		info := ServiceInfo{
			Architecture: "amd64",
			Db: DbCluster{
				Name: "test-svc",
				Type: "postgres",
				Port: 15555,
				Username: "test",
				Password: "test",
			},
			DockerNetwork: "test-network",
		}
	})
}