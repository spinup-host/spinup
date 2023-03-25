package dockerservice

import (
	"context"
	"log"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
)

var testImage = "tianon/true"

// dockerTest is a package-specific wrapper around Docker to be used for tests without creating an import cycle
type dockerTest struct {
	Docker
}

func newDockerTest(ctx context.Context, networkName string) (dockerTest, error) {
	dc, err := NewDocker(networkName)
	if err != nil {
		return dockerTest{}, err
	}

	_, err = dc.CreateNetwork(ctx)
	if err != nil {
		return dockerTest{}, errors.Wrap(err, "create network")
	}
	return dockerTest{
		Docker: dc,
	}, nil
}

// cleanup removes all containers and volumes in the docker network, and removes the network itself.
func (dt dockerTest) cleanup(t *testing.T) error {
	t.Helper()
	ctx := context.Background()
	filter := filters.NewArgs()
	filter.Add("network", dt.NetworkName)

	containers, err := dt.Cli.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: filter})
	if err != nil {
		return errors.Wrap(err, "list containers")
	}

	var cleanupErr error
	stopTimeout := 1 // timeout in seconds
	for _, c := range containers {
		t.Log(c.Names)
		stopOpts := container.StopOptions{
			Timeout: &stopTimeout,
		}
		if err = dt.Cli.ContainerStop(ctx, c.ID, stopOpts); err != nil {
			if strings.Contains(err.Error(), "No such container") {
				continue
			}
			cleanupErr = multierr.Append(cleanupErr, errors.Wrap(err, "stop container"))
		}
		if err = dt.Cli.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{}); err != nil {
			if strings.Contains(err.Error(), "No such container") {
				continue
			}
			cleanupErr = multierr.Append(cleanupErr, errors.Wrap(err, "remove container"))
		}

		// cleanup its volumes
		for _, mount := range c.Mounts {
			if mount.Type == "volume" {
				if err = dt.Cli.VolumeRemove(ctx, mount.Name, true); err != nil {
					cleanupErr = multierr.Append(cleanupErr, errors.Wrap(err, "remove volume"))
				}
			}
		}
	}

	if err = dt.Cli.NetworkRemove(ctx, dt.NetworkName); err != nil {
		cleanupErr = multierr.Append(cleanupErr, errors.Wrap(err, "remove network"))
	}
	return nil
}

func Test_imageExistsLocally(t *testing.T) {
	ctx := context.Background()
	testID := uuid.New().String()
	dc, err := newDockerTest(ctx, testID)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = dc.cleanup(t)
		if err != nil {
			t.Log("failed to clean test containers", err.Error())
		}
	})

	data := []struct {
		name                        string
		image                       string
		pullImageFromDockerRegistry bool
		expected                    bool
	}{
		{"image exist", "tianon/true", true, true},
		{"image doesnot exist", "imageDoesnotExist:notag", false, false},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			if d.pullImageFromDockerRegistry {
				log.Println("INFO: pulling docker image from docker registry:", d.image)
				// INFO: not sure what's the best way to make sure an image exists locally. Hence pulling it before testing imageExistsLocally.
				// Perhaps we could move this to TestMain() which means we need to define a type for struct - not sure its that the right way to do
				// postgres:9.6-alpine image will be pulled since its fairly small. It could be any image.
				if err := pullImageFromDockerRegistry(dc.Docker, d.image); err != nil {
					t.Errorf("error setting up imageExistsLocally() for test data %+v", d)
				}
			}
			actual, err := imageExistsLocally(context.Background(), dc.Docker, d.image)
			if err != nil {
				t.Errorf("error testing imageExistsLocally() for test data %+v", d)

			}
			if actual != d.expected {
				t.Errorf("incorrect result: actual %t , expected %t", actual, d.expected)
			}
		})
	}
}

func Test_pullImageFromDockerRegistry(t *testing.T) {
	ctx := context.Background()
	testID := uuid.New().String()
	dc, err := newDockerTest(ctx, testID)
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, dc.cleanup(t))
	})

	data := []struct {
		name     string
		image    string
		expected error
	}{
		{"image exist", "tianon/true", nil},
		{"image doesnot exist", "imageDoesnotExistInRegistry:notag", errors.New("unable to pull docker image")},
	}
	for _, d := range data {
		t.Run(d.name, func(t *testing.T) {
			actual := pullImageFromDockerRegistry(dc.Docker, d.image)
			if actual != d.expected {
				if !strings.Contains(actual.Error(), d.expected.Error()) {
					t.Errorf("incorrect result: actual %t , expected %t", actual, d.expected)
				}
			}
		})
	}
}
