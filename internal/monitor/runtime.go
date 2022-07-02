// Package monitor is responsible for managing monitoring-related services such as
// prometheus and prometheus exporters.
package monitor

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"strconv"

	"github.com/docker/docker/api/types"
	"go.uber.org/zap"

	"github.com/spinup-host/spinup/internal/dockerservice"
)

const DsnKey = "DATA_SOURCE_NAME"

// Runtime wraps runtime configuration and state of the monitoring service
type Runtime struct {
	targets       []*Target

	pgExporterContainer *dockerservice.Container
	prometheusContainer *dockerservice.Container
	dockerClient        dockerservice.Docker
	dockerHostAddr      string
	logger              *zap.Logger
}

func NewRuntime(dockerClient dockerservice.Docker, logger *zap.Logger) *Runtime {
	return &Runtime{
		targets:       make([]*Target, 0),
		dockerClient:  dockerClient,
		logger:        logger,
	}
}

// AddTarget adds a new service to the list of targets being monitored.
func (r *Runtime) AddTarget(ctx context.Context, t *Target) error {
	oldDSN, err := r.pgExporterContainer.GetEnv(ctx, r.dockerClient, DsnKey)
	if err != nil {
		return errors.Wrap(err,"could not get current data sources from postgres_exporter")
	}

	if err := r.pgExporterContainer.Stop(ctx, r.dockerClient, types.ContainerStartOptions{}); err != nil {
		return err
	}
	if err := r.pgExporterContainer.Remove(ctx, r.dockerClient); err != nil {
		return err
	}

	var newDSN string
	if oldDSN == "" {
		newDSN = fmt.Sprintf("postgresql://%s:%s@%s:%s/?sslmode=disable", t.UserName, t.Password, r.dockerHostAddr, strconv.Itoa(t.Port))
	} else {
		newDSN = fmt.Sprintf("%s,postgresql://%s:%s@%s:%s/?sslmode=disable", oldDSN, t.UserName, t.Password, r.dockerHostAddr, strconv.Itoa(t.Port))
	}
	newContainer, err := r.newPostgresExporterContainer(newDSN)
	if err != nil {
		return err
	}

	_, err = newContainer.Start(ctx, r.dockerClient)
	if err != nil {
		r.logger.Error("stopping exporter container", zap.Error(newContainer.Stop(ctx, r.dockerClient, types.ContainerStartOptions{})))
		return err
	}

	r.pgExporterContainer = newContainer
	r.targets = append(r.targets, t)
	return nil
}
