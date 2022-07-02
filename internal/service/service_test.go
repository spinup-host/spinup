package service

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spinup-host/spinup/internal/dockerservice"
	"github.com/spinup-host/spinup/internal/metastore"
	"github.com/spinup-host/spinup/internal/monitor"
	"github.com/spinup-host/spinup/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type testDocker struct {
	dockerservice.Docker
}

func newTestDocker(networkName string) (testDocker, error) {
	dc, err := dockerservice.NewDocker(networkName)
	if err != nil {
		return testDocker{}, fmt.Errorf("could not create docker client: %s", err.Error())
	}

	_, err = dc.CreateNetwork(context.Background(), networkName)
	if err != nil {
		return testDocker{}, errors.Wrap(err, "create network")
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
		stopTimeout := 1*time.Second
		if err = td.Cli.ContainerStop(ctx, c.ID, &stopTimeout); err != nil {
			return errors.Wrap(err, "stop container")
		}
		if err = td.Cli.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{}); err != nil {
			return errors.Wrap(err, "remove container")
		}

		// cleanup its volumes
		for _, mount := range c.Mounts {
			if mount.Type == "volume" {
				if err = td.Cli.VolumeRemove(ctx, mount.Name, true); err != nil {
					log.Println("failed to remove volume: ", err) // no need to return for failed volume deletion
				}
			}
		}
	}

	if err = td.Cli.NetworkRemove(ctx, td.NetworkName); err != nil {
		return errors.Wrap(err, "remove network")
	}
	return nil
}

func TestCreateService(t *testing.T) {
	testID := uuid.New().String()
	dc, err := newTestDocker(testID)
	require.NoError(t, err)

	store, path, err := newTestStore(testID)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.Remove(path)
		assert.NoError(t, dc.cleanup())
	})

	rt := monitor.NewRuntime(dc.Docker, utils.Logger)
	svc := NewService(dc.Docker, store, rt)

	t.Run("without monitoring", func(t *testing.T) {
		info := &metastore.ClusterInfo{
			Architecture: "amd64",
			Type: "postgres",
			Host: "localhost",
			Name: "test-db-"+uuid.New().String(),
			Port: 19990,
			Username: "test",
			Password: "test",
			MajVersion: 13,
			MinVersion: 6,
		}
		err = svc.CreateService(context.Background(), info)
		assert.NoError(t, err)
	})
}

func newTestStore(name string) (metastore.Db, string, error) {
	db := metastore.Db{}
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return db, "", errors.Wrap(err, "create dir")
	}

	path := filepath.Join(tmpDir, name+".db")
	db, err = metastore.NewDb(path)
	if err != nil {
		return db, "", errors.Wrap(err, "open connection")
	}
	return db, path, nil
}
