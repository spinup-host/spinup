package monitor

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/go-connections/nat"
	"github.com/spinup-host/spinup/misc"
	"log"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/spinup-host/spinup/internal/dockerservice"
)

const (
	PrometheusContainerName = "spinup-prometheus"
	PgExporterContainerName = "spinup-postgres-exporter"
)

// Target represents a postgres host (target) for monitoring
type Target struct {
	DockerNetwork string
	ContainerName string
	UserName      string
	Password      string
	Host          string
	Port          int
}

// BootstrapServices starts up prometheus and exporter services in docker containers
// todo: clean up started services on SIGKILL or SIGTERM
func (r *Runtime) BootstrapServices() error {
	var err error
	var promContainer *dockerservice.Container
	var pgExporterContainer *dockerservice.Container
	ctx := context.TODO()

	promContainer, err = r.dockerClient.GetContainer(ctx, PrometheusContainerName)
	if err != nil {
		return err
	}
	if err == nil && promContainer == nil {
		promContainer, err = newPrometheusContainer()
		if err != nil {
			return err
		}
		_, err = promContainer.Start(ctx, r.dockerClient)
		if err != nil {
			log.Println(promContainer.Stop(ctx, r.dockerClient, types.ContainerStartOptions{}))
			return err
		}
	} else {
		log.Println("reusing existing prometheus container")
	}

	pgExporterContainer, err = r.dockerClient.GetContainer(ctx, PgExporterContainerName)
	if err != nil {
		return err
	}
	if err == nil && pgExporterContainer == nil {
		pgExporterContainer, err = newPostgresExporterContainer("")
		if err != nil {
			return err
		}
		_, err = pgExporterContainer.Start(ctx, r.dockerClient)
		if err != nil {
			log.Println(pgExporterContainer.Stop(ctx, r.dockerClient, types.ContainerStartOptions{}))
			return err
		}
	} else {
		log.Println("reusing existing postgres_exporter container")
	}

	r.pgExporterContainer = pgExporterContainer
	r.prometheusContainer = promContainer
	return nil
}

func newPostgresExporterContainer(dsn string) (*dockerservice.Container, error) {
	networkName := "spinup_services"
	endpointConfig := map[string]*network.EndpointSettings{}
	endpointConfig[networkName] = &network.EndpointSettings{}
	nwConfig := network.NetworkingConfig{EndpointsConfig: endpointConfig}
	image := "quay.io/prometheuscommunity/postgres-exporter"
	var env []string
	if dsn != "" {
		env = append(env, misc.StringToDockerEnvVal("DATA_SOURCE_NAME", dsn))
	}

	metricsPort := nat.Port("9187/tcp")
	config := container.Config{
		Image: image,
		Env:   env,
	}
	postgresExporterContainer := dockerservice.NewContainer(
		PgExporterContainerName,
		config,
		container.HostConfig{
			PortBindings: nat.PortMap{
				metricsPort: []nat.PortBinding{{
					HostIP:   "127.0.0.1",
					HostPort: "9187",
				}},
			},
		},
		nwConfig,
	)
	return &postgresExporterContainer, nil
}

func newPrometheusContainer() (*dockerservice.Container, error) {
	networkName := "spinup_services"
	endpointConfig := map[string]*network.EndpointSettings{}
	endpointConfig[networkName] = &network.EndpointSettings{}
	nwConfig := network.NetworkingConfig{EndpointsConfig: endpointConfig}
	image := "bitnami/prometheus"

	config := container.Config{
		Image: image,
	}

	metricsPort := nat.Port("9090/tcp")
	promContainer := dockerservice.NewContainer(
		PrometheusContainerName,
		config,
		container.HostConfig{
			PortBindings: nat.PortMap{
				metricsPort: []nat.PortBinding{{
					HostIP:   "127.0.0.1",
					HostPort: "9090",
				}},
			},
		},
		nwConfig,
	)
	return &promContainer, nil
}
