package service

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/spinup-host/spinup/config"
	"github.com/spinup-host/spinup/internal/dockerservice"
	"github.com/spinup-host/spinup/internal/metastore"
	"github.com/spinup-host/spinup/internal/monitor"
	"github.com/spinup-host/spinup/internal/postgres"
)

type Service struct {
	store          metastore.Db
	dockerClient   dockerservice.Docker
	monitorRuntime *monitor.Runtime

	logger    *zap.Logger
	svcConfig config.Configuration
}

type ErrNoMatch struct {
	id string
}

func (e ErrNoMatch) Error() string {
	return fmt.Sprintf("no resource found with ID: '%s'", e.id)
}

func NewService(client dockerservice.Docker, store metastore.Db, mr *monitor.Runtime, logger *zap.Logger, cfg config.Configuration) Service {
	return Service{
		store:          store,
		dockerClient:   client,
		monitorRuntime: mr,

		logger:    logger,
		svcConfig: cfg,
	}
}

type Version struct {
	Maj uint
	Min uint
}
type DbCluster struct {
	Name     string
	ID       string
	Type     string
	Port     int
	Username string
	Password string
}

type backupConfig struct {
	// https://man7.org/linux/man-pages/man5/crontab.5.html
	Schedule map[string]interface{}
	Dest     Destination `json:"Dest"`
}

type Destination struct {
	Name         string
	BucketName   string
	ApiKeyID     string
	ApiKeySecret string
}

// CreateService creates a new database service alongside the needed containers.
func (svc Service) CreateService(ctx context.Context, info *metastore.ClusterInfo) error {
	image := fmt.Sprintf("%s/%s:%d.%d", "amd64", "postgres", info.MajVersion, info.MinVersion)

	postgresContainerProp := postgres.ContainerProps{
		Name:      info.Name,
		Username:  info.Username,
		Password:  info.Password,
		Port:      info.Port,
		Memory:    info.Memory,
		CPUShares: info.CPU,
		Image:     image,
	}

	pgContainer, err := postgres.NewPostgresContainer(svc.dockerClient, postgresContainerProp)
	if err != nil {
		return errors.Wrap(err, "creating new postgres container")
	}

	body, err := pgContainer.Start(ctx, svc.dockerClient)
	if err != nil {
		return errors.Wrap(err, "starting postgres container")
	}
	if len(body.Warnings) != 0 {
		svc.logger.Warn("container may be unhealthy", zap.Strings("warnings", body.Warnings))
	}
	info.ClusterID = body.ID

	if err := metastore.InsertService(svc.store, *info); err != nil {
		return errors.Wrap(err, "saving cluster info to store")
	}

	if info.Monitoring == "enable" {
		if svc.monitorRuntime == nil {
			svc.monitorRuntime = monitor.NewRuntime(svc.dockerClient, monitor.WithLogger(svc.logger), monitor.WithAppConfig(svc.svcConfig))
			if err := svc.monitorRuntime.BootstrapServices(ctx); err != nil {
				return errors.Wrap(err, "failed to start monitoring services")
			}
		}

		target := &monitor.Target{
			ContainerName: pgContainer.Name,
			UserName:      info.Username,
			Password:      info.Password,
			Port:          info.Port,
		}
		go func(target *monitor.Target) {
			// we use a background context since this is a goroutine and the orignal request
			// might have been terminated.
			if err := svc.addMonitorTarget(context.Background(), target); err != nil {
				svc.logger.Error("could not monitor target", zap.Error(err))
			}
			return
		}(target)
	}

	return nil
}

func (svc *Service) addMonitorTarget(ctx context.Context, target *monitor.Target) error {
	var err error
	if err = svc.monitorRuntime.AddTarget(ctx, target); err != nil {
		return errors.Wrap(err, "failed to add target")
	}
	return nil
}

// ListClusters list all clusters currently available
func (svc Service) ListClusters(ctx context.Context) ([]metastore.ClusterInfo, error) {
	clusters, err := metastore.AllClusters(svc.store)
	if err != nil {
		return nil, err
	}
	if len(clusters) < 1 {
		clusters = []metastore.ClusterInfo{}
	}
	return clusters, nil
}

// GetClusterByID returns the specific cluster with the given ID, returns ErrNoMatch if no cluster was found.
func (svc Service) GetClusterByID(ctx context.Context, clusterID string) (metastore.ClusterInfo, error) {
	ci, err := metastore.GetClusterByID(svc.store, clusterID)
	if err != nil {
		return ci, err
	}
	if ci.ClusterID == "" && ci.Name == "" {
		return ci, ErrNoMatch{
			id: clusterID,
		}
	}

	return ci, nil
}
