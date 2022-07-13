package dockerservice

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"github.com/spinup-host/spinup/misc"
)

type Docker struct {
	Cli *client.Client
}

var ErrDuplicateNetwork = errors.New("duplicate networks found with given name")

// GetContainer returns a docker container with the provided name (if any exists).
// if no match exists, it returns a nil container and a nil error.
func (d Docker) GetContainer(ctx context.Context, name string) (*Container, error) {
	listFilters := filters.NewArgs()
	listFilters.Add("name", name)
	containers, err := d.Cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: listFilters})
	if err != nil {
		return &Container{}, fmt.Errorf("error listing containers %w", err)
	}

	for _, match := range containers {
		// TODO: name of the container has prefixed with "/"
		// I have hardcoded here; perhaps there is a better way to handle this
		if misc.SliceContainsString(match.Names, "/"+name) {
			data, err := d.Cli.ContainerInspect(ctx, match.ID)
			if err != nil {
				return nil, errors.Wrapf(err, "getting data for container %s", match.ID)
			}

			c := &Container{
				ID:     match.ID,
				Name:   name,
				Config: *data.Config,
				NetworkConfig: network.NetworkingConfig{
					EndpointsConfig: data.NetworkSettings.Networks,
				},
			}
			return c, nil
		}
	}
	return nil, nil
}

// CreateNetwork creates a new Docker network.
func (d Docker) CreateNetwork(ctx context.Context, name string, opt types.NetworkCreate) (types.NetworkCreateResponse, error) {
	networkResponse, err := d.Cli.NetworkCreate(ctx, name, opt)
	if err == nil {
		return networkResponse, nil
	}

	if !strings.Contains(err.Error(), fmt.Sprintf("network with name %s already exists", name)) {
		return networkResponse, err
	} else {
		listFilters := filters.NewArgs()
		listFilters.Add("name", name)
		networks, err := d.Cli.NetworkList(ctx, types.NetworkListOptions{Filters: listFilters})
		if err != nil {
			return networkResponse, err
		}

		if len(networks) > 1 {
			// multiple networks with the same name exists.
			// we return an error and let the user clean them out
			return networkResponse, ErrDuplicateNetwork
		}
		return types.NetworkCreateResponse{
			ID: networks[0].ID,
		}, nil
	}
}

// RemoveNetwork removes an existing docker network.
func (d Docker) RemoveNetwork(ctx context.Context, networkID string) error {
	return d.Cli.NetworkRemove(ctx, networkID)
}

func CreateVolume(ctx context.Context, d Docker, opt volume.VolumeCreateBody) (types.Volume, error) {
	log.Println("INFO: volume created successfully ", opt.Name)
	return d.Cli.VolumeCreate(ctx, opt)
}

func RemoveVolume(ctx context.Context, d Docker, volumeID string) error {
	return d.Cli.VolumeRemove(ctx, volumeID, true)
}
