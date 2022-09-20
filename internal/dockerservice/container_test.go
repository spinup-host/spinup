package dockerservice

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStart(t *testing.T) {
	testID := uuid.New().String()
	ctx := context.Background()
	dc, err := newDockerTest(ctx, testID)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = dc.cleanup(t)
		if err != nil {
			t.Logf("failed to clean up test containers" + err.Error())
		}
	})

	t.Run("duplicate container name", func(t *testing.T) {
		c1 := NewContainer(
			"test_container",
			container.Config{Image: testImage},
			container.HostConfig{},
			network.NetworkingConfig{
				EndpointsConfig: map[string]*network.EndpointSettings{
					dc.NetworkName: {},
				},
			},
		)
		_, err = c1.Start(ctx, dc.Docker)
		assert.NoError(t, err)

		c2 := NewContainer(
			"test_container",
			container.Config{Image: testImage},
			container.HostConfig{}, network.NetworkingConfig{
				EndpointsConfig: map[string]*network.EndpointSettings{
					dc.NetworkName: {},
				},
			})
		_, err = c2.Start(ctx, dc.Docker)
		assert.ErrorIs(t, err, ErrDuplicateContainerName)
	})
}
