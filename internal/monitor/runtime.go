// Package monitor is responsible for managing monitoring-related services such as
// prometheus and prometheus exporters.
package monitor

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/spinup-host/spinup/config"
	ds "github.com/spinup-host/spinup/internal/dockerservice"
)

const (
	DsnKey = "DATA_SOURCE_NAME"
)

// Runtime wraps runtime configuration and state of the monitoring service
type Runtime struct {
	targets              []*Target
	pgExporterName       string
	prometheusName       string
	grafanaContainerName string
	pgExporterContainer  *ds.Container
	prometheusContainer  *ds.Container
	dockerClient         ds.Docker
	dockerHostAddr       string

	appConfig config.Configuration
	logger    *zap.Logger
}

type RuntimeOptions func(runtime *Runtime)

func WithLogger(logger *zap.Logger) RuntimeOptions {
	return func(runtime *Runtime) {
		runtime.logger = logger
	}
}

func WithAppConfig(cfg config.Configuration) RuntimeOptions {
	return func(runtime *Runtime) {
		runtime.appConfig = cfg
	}
}

func NewRuntime(dockerClient ds.Docker, opts ...RuntimeOptions) *Runtime {
	rt := &Runtime{
		targets:              make([]*Target, 0),
		dockerClient:         dockerClient,
		pgExporterName:       ds.PgExporterPrefix,
		prometheusName:       ds.PrometheusPrefix,
		grafanaContainerName: ds.GrafanaPrefix,
	}
	if dockerClient.NetworkName != "" {
		rt.pgExporterName = ds.PgExporterPrefix + "-" + dockerClient.NetworkName
		rt.prometheusName = ds.PrometheusPrefix + "-" + dockerClient.NetworkName
		rt.grafanaContainerName = ds.GrafanaPrefix + "-" + dockerClient.NetworkName
	}

	for _, opt := range opts {
		opt(rt)
	}
	return rt
}

// AddTarget adds a new service to the list of targets being monitored.
func (r *Runtime) AddTarget(ctx context.Context, t *Target) error {
	oldDSN, err := r.pgExporterContainer.GetEnv(ctx, r.dockerClient, DsnKey)
	if err != nil {
		return errors.Wrap(err, "could not get current data sources from postgres_exporter")
	}

	if err := r.pgExporterContainer.Stop(ctx, r.dockerClient); err != nil {
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
		r.logger.Error("stopping exporter container", zap.Error(newContainer.Stop(ctx, r.dockerClient)))
		return err
	}

	r.pgExporterContainer = newContainer
	r.targets = append(r.targets, t)
	return nil
}
