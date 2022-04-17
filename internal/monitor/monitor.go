package monitor

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/spinup-host/spinup/config"
	"github.com/spinup-host/spinup/internal/dockerservice"
	"github.com/spinup-host/spinup/misc"
)

const (
	PrometheusContainerName = "spinup-prometheus"
	PgExporterContainerName = "spinup-postgres-exporter"
	NetworkName             = "spinup_services"
)

// Target represents a postgres service for monitoring.
// The assumption is that all services are running on the same host, hence only ports should differ
type Target struct {
	DockerNetwork string
	ContainerName string
	UserName      string
	Password      string
	Port          int
}

// BootstrapServices starts up prometheus and exporter services in docker containers
// todo: clean up started services on SIGKILL or SIGTERM
func (r *Runtime) BootstrapServices() error {
	var err error
	var promContainer *dockerservice.Container
	var pgExporterContainer *dockerservice.Container
	ctx := context.TODO()

	defer func() {
		if err != nil {
			if promContainer != nil {
				stopErr := promContainer.Stop(ctx, r.dockerClient, types.ContainerStartOptions{})
				r.logger.Error("stopping prometheus container", zap.Error(stopErr))
			}

			if pgExporterContainer != nil {
				stopErr := pgExporterContainer.Stop(ctx, r.dockerClient, types.ContainerStartOptions{})
				r.logger.Error("stopping exporter container", zap.Error(stopErr))
			}
		}
	}()

	promContainer, err = r.dockerClient.GetContainer(ctx, PrometheusContainerName)
	if err != nil {
		return err
	}

	if err == nil && promContainer == nil {
		promCfgPath, err := r.getPromConfigPath()
		if err != nil {
			return errors.Wrap(err, "failed to mount prometheus config")
		}
		promContainer, err = r.newPrometheusContainer(promCfgPath)
		if err != nil {
			return err
		}
		_, err = promContainer.Start(ctx, r.dockerClient)
		if err != nil {
			return errors.Wrap(err, "failed to start prometheus container")
		}

		// we expect all containers to have the same gateway IP, but we assign it here
		// so that we can update the prometheus config with the right IP of targets
		r.dockerHostAddr = promContainer.NetworkConfig.EndpointsConfig[NetworkName].Gateway
		if err = r.writePromConfig(promCfgPath); err != nil {
			return errors.Wrap(err, "failed to update prometheus config")
		}
	} else {
		// if the container exists, we only update the host address without over-writing the existing prometheus config
		r.dockerHostAddr = promContainer.NetworkConfig.EndpointsConfig[NetworkName].Gateway
		r.logger.Info("reusing existing prometheus container")
	}

	pgExporterContainer, err = r.dockerClient.GetContainer(ctx, PgExporterContainerName)
	if err != nil {
		return err
	}
	if err == nil && pgExporterContainer == nil {
		pgExporterContainer, err = r.newPostgresExporterContainer("")
		if err != nil {
			return err
		}
		_, err = pgExporterContainer.Start(ctx, r.dockerClient)
		if err != nil {
			return errors.Wrap(err, "failed to start pg_exporter container")
		}
	} else {
		r.logger.Info("reusing existing postgres_exporter container")
	}

	r.logger.Info(fmt.Sprintf("using docker host address :%s", r.dockerHostAddr))
	r.pgExporterContainer = pgExporterContainer
	r.prometheusContainer = promContainer
	return nil
}

func (r *Runtime) newPostgresExporterContainer(dsn string) (*dockerservice.Container, error) {
	endpointConfig := map[string]*network.EndpointSettings{}
	endpointConfig[NetworkName] = &network.EndpointSettings{}
	nwConfig := network.NetworkingConfig{EndpointsConfig: endpointConfig}
	image := "quay.io/prometheuscommunity/postgres-exporter"
	var env []string
	if dsn != "" {
		env = append(env, misc.StringToDockerEnvVal("DATA_SOURCE_NAME", dsn))
	}

	metricsPort := nat.Port("9187/tcp")
	postgresExporterContainer := dockerservice.NewContainer(
		PgExporterContainerName,
		container.Config{
			Image: image,
			Env:   env,
		},
		container.HostConfig{
			PortBindings: nat.PortMap{
				metricsPort: []nat.PortBinding{{
					HostIP:   "0.0.0.0",
					HostPort: "9187",
				}},
			},
		},
		nwConfig,
	)
	return &postgresExporterContainer, nil
}

func (r *Runtime) newPrometheusContainer(promCfgPath string) (*dockerservice.Container, error) {
	endpointConfig := map[string]*network.EndpointSettings{}
	endpointConfig[NetworkName] = &network.EndpointSettings{}
	nwConfig := network.NetworkingConfig{EndpointsConfig: endpointConfig}
	image := "bitnami/prometheus"

	promDataDir := filepath.Join(config.Cfg.Common.ProjectDir, "prom_data")
	err := os.Mkdir(promDataDir, 0644)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return &dockerservice.Container{}, errors.Wrap(err, "could not create prometheus data dir")
	}

	// Mount points for prometheus config and prometheus persistence
	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: promCfgPath,
			Target: "/opt/bitnami/prometheus/conf/prometheus.yml",
		},
		{
			Type:   mount.TypeBind,
			Source: promDataDir,
			Target: "/opt/bitnami/prometheus/data",
		},
	}

	metricsPort := nat.Port("9090/tcp")
	promContainer := dockerservice.NewContainer(
		PrometheusContainerName,
		container.Config{
			Image: image,
			User:  "root",
		},
		container.HostConfig{
			PortBindings: nat.PortMap{
				metricsPort: []nat.PortBinding{{
					HostIP:   "0.0.0.0",
					HostPort: "9090",
				}},
			},
			Mounts: mounts,
		},
		nwConfig,
	)
	return &promContainer, nil
}

func (r *Runtime) getPromConfigPath() (string, error) {
	cfgPath := filepath.Join(config.Cfg.Common.ProjectDir, "prometheus.yml")
	_, err := os.Open(cfgPath)
	if errors.Is(err, os.ErrNotExist) {
		_, err = os.Create(cfgPath)
	}

	return cfgPath, err
}

func (r *Runtime) writePromConfig(cfgPath string) error {
	cfg := fmt.Sprintf(`scrape_configs:
  - job_name: prometheus
    scrape_interval: 5s
    static_configs:
    - targets:
      - "%s"
  - job_name: pg_exporter
    scrape_interval: 5s
    static_configs:
    - targets:
      - "%s"
`, net.JoinHostPort(r.dockerHostAddr, "9090"), net.JoinHostPort(r.dockerHostAddr, "9187"))
	if err := os.WriteFile(cfgPath, []byte(cfg), 0644); err != nil {
		return err
	}
	return nil
}
