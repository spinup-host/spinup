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

	"github.com/spinup-host/spinup/internal/dockerservice"
	"github.com/spinup-host/spinup/misc"
)

// Target represents a postgres service for monitoring.
// it contains only fields that differ between different services
type Target struct {
	ContainerName string
	UserName      string
	Password      string
	Port          int
}

// BootstrapServices starts up prometheus and exporter services in docker containers
// todo: clean up started services on SIGKILL or SIGTERM
func (r *Runtime) BootstrapServices(ctx context.Context) error {
	var err error
	var promContainer *dockerservice.Container
	var pgExporterContainer *dockerservice.Container
	var gfContainer *dockerservice.Container

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

			if gfContainer != nil {
				stopErr := gfContainer.Stop(ctx, r.dockerClient, types.ContainerStartOptions{})
				r.logger.Error("stopping grafana container", zap.Error(stopErr))
			}
		}
	}()

	promContainer, err = r.dockerClient.GetContainer(ctx, r.prometheusName)
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
		r.dockerHostAddr = promContainer.NetworkConfig.EndpointsConfig[r.dockerClient.NetworkName].Gateway
		if err = r.writePromConfig(promCfgPath); err != nil {
			return errors.Wrap(err, "failed to update prometheus config")
		}
	} else {
		// if the container exists, we only update the host address without over-writing the existing prometheus config
		r.dockerHostAddr = promContainer.NetworkConfig.EndpointsConfig[r.dockerClient.NetworkName].Gateway
		r.logger.Info("reusing existing prometheus container")
		err = promContainer.StartExisting(ctx, r.dockerClient)
		if err != nil {
			return errors.Wrap(err, "failed to start existing prometheus container")
		}
	}

	pgExporterContainer, err = r.dockerClient.GetContainer(ctx, r.pgExporterName)
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
		err = pgExporterContainer.StartExisting(ctx, r.dockerClient)
		if err != nil {
			return errors.Wrap(err, "failed to start existing pg_exporter container")
		}
	}

	gfContainer, err = r.dockerClient.GetContainer(ctx, r.grafanaContainerName)
	if err != nil {
		return err
	}
	gfConfigPath, err := r.getGrafanaConfigPath()
	if err != nil {
		return errors.Wrap(err, "failed to read grafana config")
	}

	if err == nil && gfContainer == nil {
		gfContainer, err = r.newGrafanaContainer(gfConfigPath)
		if err != nil {
			return err
		}
		_, err = gfContainer.Start(ctx, r.dockerClient)
		if err != nil {
			return errors.Wrap(err, "failed to start grafana container")
		}
	} else {
		r.logger.Info("reusing existing grafana container")
		err = gfContainer.StartExisting(ctx, r.dockerClient)
		if err != nil {
			return errors.Wrap(err, "failed to start existing grafana container")
		}
	}
	r.dockerHostAddr = gfContainer.NetworkConfig.EndpointsConfig[r.dockerClient.NetworkName].Gateway
	if err = r.writeGrafanaConfig(gfConfigPath); err != nil {
		return errors.Wrap(err, "failed to update prometheus config")
	}
	if err = gfContainer.Restart(ctx, r.dockerClient); err != nil {
		return err
	}

	r.logger.Info(fmt.Sprintf("using docker host address :%s", r.dockerHostAddr))
	r.pgExporterContainer = pgExporterContainer
	r.prometheusContainer = promContainer
	return nil
}

func (r *Runtime) newPostgresExporterContainer(dsn string) (*dockerservice.Container, error) {
	endpointConfig := map[string]*network.EndpointSettings{}
	endpointConfig[r.dockerClient.NetworkName] = &network.EndpointSettings{}
	nwConfig := network.NetworkingConfig{EndpointsConfig: endpointConfig}
	image := "quay.io/prometheuscommunity/postgres-exporter"
	env := []string{
		misc.StringToDockerEnvVal("DATA_SOURCE_NAME", dsn),
	}

	metricsPort := nat.Port("9187/tcp")
	postgresExporterContainer := dockerservice.NewContainer(
		r.pgExporterName,
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
	endpointConfig[r.dockerClient.NetworkName] = &network.EndpointSettings{}
	nwConfig := network.NetworkingConfig{EndpointsConfig: endpointConfig}
	image := "bitnami/prometheus"

	promDataDir := filepath.Join(r.appConfig.Common.ProjectDir, "prom_data")
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
		r.prometheusName,
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

func (r *Runtime) newGrafanaContainer(gfConfigPath string) (*dockerservice.Container, error) {
	endpointConfig := map[string]*network.EndpointSettings{}
	endpointConfig[r.dockerClient.NetworkName] = &network.EndpointSettings{}
	nwConfig := network.NetworkingConfig{EndpointsConfig: endpointConfig}

	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: gfConfigPath,
			Target: "/etc/grafana/provisioning/datasources/prometheus-grafana.yml",
		},
	}

	image := "grafana/grafana-oss:9.0.5"
	gfContainer := dockerservice.NewContainer(
		r.grafanaContainerName, container.Config{
			Image: image,
		},
		container.HostConfig{
			PortBindings: nat.PortMap{
				nat.Port("3000/tcp"): []nat.PortBinding{{
					HostIP:   "0.0.0.0",
					HostPort: "9091",
				}},
			},
			Mounts: mounts,
		},
		nwConfig,
	)
	return &gfContainer, nil
}

func (r *Runtime) getPromConfigPath() (string, error) {
	cfgPath := filepath.Join(r.appConfig.Common.ProjectDir, "prometheus.yml")
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

func (r *Runtime) getGrafanaConfigPath() (string, error) {
	cfgPath := filepath.Join(r.appConfig.Common.ProjectDir, "prometheus-grafana.yml")
	_, err := os.Open(cfgPath)
	if errors.Is(err, os.ErrNotExist) {
		_, err = os.Create(cfgPath)
	}

	return cfgPath, err
}

func (r *Runtime) writeGrafanaConfig(cfgPath string) error {
	cfg := fmt.Sprintf(`# prometheus provisioning file for grafana
apiVersion: 1

datasources:
- name: Prometheus'
  type: prometheus
  access: 'proxy'
  orgId: 1
  url: http://%s
  isDefault: true
  version: 1
  editable: true
`, net.JoinHostPort(r.dockerHostAddr, "9090"))

	if err := os.WriteFile(cfgPath, []byte(cfg), 0644); err != nil {
		return errors.Wrap(err, "creating grafana config")
	}
	return nil
}
