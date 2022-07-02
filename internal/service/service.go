package service

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spinup-host/spinup/internal/dockerservice"
	"github.com/spinup-host/spinup/internal/metastore"
	"github.com/spinup-host/spinup/internal/monitor"
	"github.com/spinup-host/spinup/internal/postgres"
	"github.com/spinup-host/spinup/utils"
	"go.uber.org/zap"
)

type Service struct {
	store metastore.Db
	dockerClient dockerservice.Docker
	monitorRuntime *monitor.Runtime
}

func NewService(client dockerservice.Docker, store metastore.Db, mr *monitor.Runtime ) Service {
	return Service{
		store: store,
		dockerClient: client,
		monitorRuntime: mr,
	}
}

type ServiceInfo struct {
	// one of arm64v8 or arm32v7 or amd64
	Architecture string
	//Port         uint
	Db            DbCluster
	DockerNetwork string
	Version       Version
	BackupEnabled bool
	Backup        backupConfig
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

	Memory     int64
	CPU        int64
	Monitoring string
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

func (svc Service) CreateService(ctx context.Context, info *ServiceInfo) error {
	image := fmt.Sprintf("%s/%s:%d.%d", info.Architecture, info.Db.Type, info.Version.Maj, info.Version.Min)

	postgresContainerProp := postgres.ContainerProps{
		Name:      info.Db.Name,
		Username:  info.Db.Username,
		Password:  info.Db.Password,
		Port:      info.Db.Port,
		Memory:    info.Db.Memory,
		CPUShares: info.Db.CPU,
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
		utils.Logger.Warn("container may be unhealthy", zap.Strings("warnings", body.Warnings))
	}
	pgContainer.ID = body.ID
	pgContainer.Warning = body.Warnings
	utils.Logger.Info("created postgres container", zap.String("container ID", pgContainer.ID))

	insertSql := "insert into clusterInfo(clusterId, name, username, password, port, majVersion, minVersion) values(?, ?, ?, ?, ?, ?, ?)"
	if err := metastore.InsertService(svc.store, insertSql, pgContainer.ID, info.Db.Name, info.Db.Username, info.Db.Password, info.Db.Port, int(info.Version.Maj), int(info.Version.Min)); err != nil {
		return errors.Wrap(err, "saving cluster info to store")
	}

	if info.Db.Monitoring == "enable" {
		target := &monitor.Target{
			ContainerName: pgContainer.Name,
			UserName:      info.Db.Username,
			Password:      info.Db.Password,
			Port:          info.Db.Port,
		}
		go func(target *monitor.Target) {
			// we use a background context since this is a goroutine and the orignal request
			// might have been terminated.
			if err := svc.addMonitorTarget(context.Background(), target); err != nil {
				utils.Logger.Error("could not monitor target", zap.Error(err))
			}
			return
		}(target)
	}

	return nil
}

func (svc *Service) addMonitorTarget(ctx context.Context, target *monitor.Target) error {
	var err error
	if svc.monitorRuntime == nil {
		svc.monitorRuntime = monitor.NewRuntime(svc.dockerClient, utils.Logger)
		if err := svc.monitorRuntime.BootstrapServices(ctx); err != nil {
			return errors.Wrap(err, "failed to start monitoring services")
		}
	}
	if err = svc.monitorRuntime.AddTarget(ctx, target); err != nil {
		return errors.Wrap(err, "failed to add target")
	}
	return nil
}