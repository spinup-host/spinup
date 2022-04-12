// Package monitor is responsible for managing monitoring-related services such as
// prometheus and prometheus exporters.
package monitor

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/spinup-host/spinup/internal/dockerservice"
	"log"
	"strconv"
)

// Runtime wraps runtime configuration and state of the monitoring service
type Runtime struct {
	targets       []*Target
	pgExporterDSN string

	pgExporterContainer *dockerservice.Container
	prometheusContainer *dockerservice.Container
	dockerClient        dockerservice.Docker
}

func NewRuntime(dockerClient dockerservice.Docker) *Runtime {
	return &Runtime{
		targets:       make([]*Target, 0),
		pgExporterDSN: "",
		dockerClient:  dockerClient,
	}
}

// AddTarget adds a new service to the list of targets being monitored.
func (r *Runtime) AddTarget(ctx context.Context, t *Target) error {
	if err := r.pgExporterContainer.Stop(ctx, r.dockerClient, types.ContainerStartOptions{}); err != nil {
		return err
	}
	if err := r.pgExporterContainer.Remove(ctx, r.dockerClient); err != nil {
		return err
	}

	newDSN := fmt.Sprintf("%s,postgresql://%s:%s@%s:%s/?sslmode=disable", r.pgExporterDSN, t.UserName, t.Password, t.Host, strconv.Itoa(t.Port))
	newContainer, err := newPostgresExporterContainer(newDSN)
	if err != nil {
		return err
	}

	_, err = newContainer.Start(ctx, r.dockerClient)
	if err != nil {
		log.Println(newContainer.Stop(ctx, r.dockerClient, types.ContainerStartOptions{}))
		return err
	}

	r.pgExporterContainer = newContainer
	r.targets = append(r.targets, t)
	return nil
}
