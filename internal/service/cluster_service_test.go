package service

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/spinup-host/spinup/config"
	ds "github.com/spinup-host/spinup/internal/dockerservice"
	"github.com/spinup-host/spinup/internal/metastore"
	"github.com/spinup-host/spinup/internal/monitor"
	"github.com/spinup-host/spinup/tests"
)

var (
	maxPort = 60000
	minPort = 10000
)

func TestCreateService(t *testing.T) {
	testID := uuid.New().String()
	ctx := context.Background()
	dc, err := tests.NewDockerTest(ctx, testID)
	require.NoError(t, err)

	store, path, err := newTestStore(testID)
	require.NoError(t, err)

	logger, err := newTestLogger()
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = os.Remove(path)
		assert.NoError(t, dc.Cleanup())
	})

	rand.Seed(time.Now().UnixNano())
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
		containerName := "test-db-" + uuid.New().String()
		ctx := context.Background()
		info := &metastore.ClusterInfo{
			Architecture: "amd64",
			Type:         "postgres",
			Host:         "localhost",
			Name:         containerName,
			Port:         rand.Intn(maxPort-minPort) + minPort,
			Username:     "test",
			Password:     "test",
			MajVersion:   13,
			MinVersion:   6,
		}
		err = svc.CreateService(ctx, info)
		assert.NoError(t, err)

		pg, err := svc.dockerClient.GetContainer(ctx, "spinup-postgres-"+containerName)
		assert.NoError(t, err)
		assert.Equal(t, "running", pg.State)
	})

	t.Run("with monitoring", func(t *testing.T) {
		containerName := "test-db-" + uuid.New().String()
		ctx := context.Background()

		err = svc.monitorRuntime.BootstrapServices(ctx)
		assert.NoError(t, err)

		info := &metastore.ClusterInfo{
			Architecture: "amd64",
			Type:         "postgres",
			Host:         "localhost",
			Name:         containerName,
			Port:         rand.Intn(maxPort-minPort) + minPort,
			Username:     "test",
			Password:     "test",
			MajVersion:   13,
			MinVersion:   6,
			Monitoring:   "enable",
		}
		exporterName := ds.PgExporterPrefix + "-" + testID
		currentExporter, err := svc.dockerClient.GetContainer(ctx, exporterName)
		assert.NoError(t, err)

		err = svc.CreateService(ctx, info)
		assert.NoError(t, err)

		pgC, err := svc.dockerClient.GetContainer(ctx, "spinup-postgres-"+containerName)
		assert.NoError(t, err)
		assert.Equal(t, "running", pgC.State)

		grafanaC, err := svc.dockerClient.GetContainer(ctx, ds.GrafanaPrefix+"-"+testID)
		assert.NoError(t, err)
		assert.Equal(t, "running", grafanaC.State)

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
