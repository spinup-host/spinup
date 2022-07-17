package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/spinup-host/spinup/config"
	ds "github.com/spinup-host/spinup/internal/dockerservice"
	"github.com/spinup-host/spinup/internal/metastore"
	"github.com/spinup-host/spinup/internal/monitor"
)

type testDocker struct {
	ds.Docker
}

func newTestDocker(networkName string) (testDocker, error) {
	dc, err := ds.NewDocker(networkName)
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

	containers, err := td.Cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: filter})
	if err != nil {
		return errors.Wrap(err, "list containers")
	}

	var cleanupErr error
	for _, c := range containers {
		stopTimeout := 1*time.Second
		if err = td.Cli.ContainerStop(ctx, c.ID, &stopTimeout); err != nil {
			if strings.Contains(err.Error(), "No such container") {
				continue
			}
			cleanupErr = multierr.Append(cleanupErr, errors.Wrap(err, "stop container"))
		}
		if err = td.Cli.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{}); err != nil {
			if strings.Contains(err.Error(), "No such container") {
				continue
			}
			cleanupErr = multierr.Append(cleanupErr, errors.Wrap(err,"remove container"))
		}

		// cleanup its volumes
		for _, mount := range c.Mounts {
			if mount.Type == "volume" {
				if err = td.Cli.VolumeRemove(ctx, mount.Name, true); err != nil {
					cleanupErr = multierr.Append(cleanupErr, errors.Wrap(err, "remove volume"))
				}
			}
		}
	}

	if err = td.Cli.NetworkRemove(ctx, td.NetworkName); err != nil {
		cleanupErr = multierr.Append(cleanupErr, errors.Wrap(err, "remove network"))
	}
	return nil
}

func TestCreateService(t *testing.T) {
	testID := uuid.New().String()
	dc, err := newTestDocker(testID)
	require.NoError(t, err)

	store, path, err := newTestStore(testID)
	require.NoError(t, err)

	logger, err := newTestLogger()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.Remove(path)
		assert.NoError(t, dc.cleanup())
	})

	cfg := config.Configuration{
		Common: struct {
			Architecture string `yaml:"architecture"`
			ProjectDir   string `yaml:"projectDir"`
			Ports        []int  `yaml:"ports"`
			ClientID     string `yaml:"client_id"`
			ClientSecret string `yaml:"client_secret"`
			ApiKey       string `yaml:"api_key"`
			LogDir       string `yaml:"log_dir"`
			LogFile      string `yaml:"log_file"`
			Monitoring   bool   `yaml:"monitoring"`
		}(struct {
			Architecture string
			ProjectDir   string
			Ports        []int
			ClientID     string
			ClientSecret string
			ApiKey       string
			LogDir       string
			LogFile      string
			Monitoring   bool
		}{ProjectDir: os.TempDir()}),
	}
	rt := monitor.NewRuntime(dc.Docker, monitor.WithLogger(logger), monitor.WithAppConfig(cfg))
	svc := NewService(dc.Docker, store, rt, logger, cfg)

	t.Run("without monitoring", func(t *testing.T) {
		containerName := "test-db-"+uuid.New().String()
		ctx := context.Background()
		info := &metastore.ClusterInfo{
			Architecture: "amd64",
			Type: "postgres",
			Host: "localhost",
			Name: containerName,
			Port: 19990,
			Username: "test",
			Password: "test",
			MajVersion: 13,
			MinVersion: 6,
		}
		err = svc.CreateService(ctx, info)
		assert.NoError(t, err)

		pg, err := svc.dockerClient.GetContainer(ctx, "spinup-postgres-"+containerName)
		assert.NoError(t, err)
		assert.Equal(t, "running", pg.State)
	})
	
	t.Run("with monitoring", func(t *testing.T) {
		containerName := "test-db-"+uuid.New().String()
		ctx := context.Background()

		err = svc.monitorRuntime.BootstrapServices(ctx)
		assert.NoError(t, err)

		info := &metastore.ClusterInfo{
			Architecture: "amd64",
			Type: "postgres",
			Host: "localhost",
			Name: containerName,
			Port: 19991,
			Username: "test",
			Password: "test",
			MajVersion: 13,
			MinVersion: 6,
			Monitoring: "enable",
		}
		exporterName := ds.PgExporterPrefix + "-" +testID
		currentExporter, err := svc.dockerClient.GetContainer(ctx, exporterName)
		assert.NoError(t, err)

		err = svc.CreateService(ctx, info)
		assert.NoError(t, err)

		pgC, err := svc.dockerClient.GetContainer(ctx, "spinup-postgres-"+containerName)
		assert.NoError(t, err)
		assert.Equal(t, "running", pgC.State)

		// monitoring services are started in the background, so we try for a while before giving up
		tries := uint32(0)
		maxTries := uint32(10)
		var newExporter *ds.Container
		for tries < maxTries {
			newExporter, err = svc.dockerClient.GetContainer(ctx, exporterName)
			if err == nil && newExporter.ID != currentExporter.ID && newExporter.State == "running" {
				break
			}
			time.Sleep(1 * time.Second)
			tries++
		}
		assert.NoErrorf(t, err, "could not find new exporter container after %d retries", tries)
		assert.Equal(t, "running", newExporter.State)
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

func newTestLogger() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{"stdout"}
	return cfg.Build()
}
