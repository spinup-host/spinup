package dockerservice

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/pkg/errors"

	"github.com/spinup-host/spinup/misc"
)

type Docker struct {
	Cli         *client.Client
	NetworkName string
}

// NewDocker returns a Docker struct
func NewDocker(networkName string, opts ...client.Opt) (Docker, error) {
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		fmt.Printf("error creating client %v", err)
	}
	return Docker{NetworkName: networkName, Cli: cli}, nil
}

var ErrDuplicateNetwork = errors.New("duplicate networks found with given name")
var ErrDuplicateContainerName = errors.New("a container already exists with the given name")

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
				ID:         match.ID,
				Name:       name,
				State:      match.State,
				Config:     *data.Config,
				HostConfig: *data.HostConfig,
				// note that if the container is stopped, network info will be empty and won't be populated
				// until you call one of Start(), Restart(), or StartExisting().
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
func (d Docker) CreateNetwork(ctx context.Context) (types.NetworkCreateResponse, error) {
	networkResponse, err := d.Cli.NetworkCreate(ctx, d.NetworkName, types.NetworkCreate{CheckDuplicate: true})
	if err == nil {
		return networkResponse, nil
	}

	if !strings.Contains(err.Error(), fmt.Sprintf("network with name %s already exists", d.NetworkName)) {
		return networkResponse, err
	} else {
		listFilters := filters.NewArgs()
		listFilters.Add("name", d.NetworkName)
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

func CreateVolume(ctx context.Context, d Docker, opt volume.CreateOptions) (volume.Volume, error) {
	return d.Cli.VolumeCreate(ctx, opt)
}

func RemoveVolume(ctx context.Context, d Docker, volumeID string) error {
	return d.Cli.VolumeRemove(ctx, volumeID, true)
}

func resolveLocalPath(localPath string) (absPath string, err error) {
	if absPath, err = filepath.Abs(localPath); err != nil {
		return
	}
	return archive.PreserveTrailingDotOrSeparator(absPath, localPath, '/'), nil
}

// CopyFromContainer copies the source file to the given container.
// It is a slimmed-down version of the original `docker cp` implementation.
// e.g (no support for STDIN streaming, progress output, symlinks, or path validation)
func (d Docker) CopyFromContainer(ctx context.Context, containerID, srcPath, dstPath string) (err error) {
	if dstPath != "-" {
		// Get an absolute destination path.
		dstPath, err = resolveLocalPath(dstPath)
		if err != nil {
			return err
		}
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	content, stat, err := d.Cli.CopyFromContainer(ctx, containerID, srcPath)
	if err != nil {
		return err
	}
	defer content.Close()

	srcInfo := archive.CopyInfo{
		Path:   srcPath,
		Exists: true,
		IsDir:  stat.Mode.IsDir(),
	}

	preArchive := content
	if len(srcInfo.RebaseName) != 0 {
		_, srcBase := archive.SplitPathDirEntry(srcInfo.Path)
		preArchive = archive.RebaseArchiveEntries(content, srcBase, srcInfo.RebaseName)
	}
	return archive.CopyTo(preArchive, srcInfo, dstPath)
}

// CopyToContainer copies the source file to the given container.
// It is a slimmed-down version of the original `docker cp` implementation.
// Care must be taken to ensure that the dstPath is an existing directory in the target container.
func (d Docker) CopyToContainer(ctx context.Context, containerID, srcPath, dstPath string) (err error) {
	if srcPath != "-" {
		// Get an absolute source path.
		srcPath, err = resolveLocalPath(srcPath)
		if err != nil {
			return err
		}
	}

	// Prepare destination copy info by stat-ing the container path.
	dstInfo := archive.CopyInfo{Path: dstPath}
	dstStat, err := d.Cli.ContainerStatPath(ctx, containerID, dstPath)

	// Ignore any error and assume that the parent directory of the destination
	// path exists, in which case the copy may still succeed. If there is any
	// type of conflict (e.g., non-directory overwriting an existing directory
	// or vice versa) the extraction will fail. If the destination simply did
	// not exist, but the parent directory does, the extraction will still
	// succeed.
	if err == nil {
		dstInfo.Exists, dstInfo.IsDir = true, dstStat.Mode.IsDir()
	}

	var (
		content         io.ReadCloser
		resolvedDstPath string
	)

	if srcPath == "-" {
		content = os.Stdin
		resolvedDstPath = dstInfo.Path
		if !dstInfo.IsDir {
			return errors.Errorf("destination \"%s:%s\" must be a directory", containerID, dstPath)
		}
	} else {
		srcInfo, err := archive.CopyInfoSourcePath(srcPath, false)
		if err != nil {
			return errors.Wrap(err, "failed to prepare source file(s)")
		}

		srcArchive, err := archive.TarResource(srcInfo)
		if err != nil {
			return errors.Wrap(err, "failed to archive source file(s)")
		}
		defer srcArchive.Close()

		// With the stat info about the local source as well as the
		// destination, we have enough information to know whether we need to
		// alter the archive that we upload so that when the server extracts
		// it to the specified directory in the container we get the desired
		// copy behavior.

		// See comments in the implementation of `archive.PrepareArchiveCopy`
		// for exactly what goes into deciding how and whether the source
		// archive needs to be altered for the correct copy behavior when it is
		// extracted. This function also infers from the source and destination
		// info which directory to extract to, which may be the parent of the
		// destination that the user specified.
		dstDir, preparedArchive, err := archive.PrepareArchiveCopy(srcArchive, srcInfo, dstInfo)
		if err != nil {
			return errors.Wrap(err, "failed to prepare archive copy")
		}
		defer preparedArchive.Close()

		resolvedDstPath = dstDir
		content = preparedArchive
	}

	options := types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: false,
		CopyUIDGID:                false,
	}
	return d.Cli.CopyToContainer(ctx, containerID, resolvedDstPath, content, options)
}
