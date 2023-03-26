package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/spinup-host/spinup/internal/metastore"
	"github.com/spinup-host/spinup/testutils"
)

// cluster tests contain unit tests for cluster-related API endpoints.
func TestListCluster(t *testing.T) {
	svc := &mockClusterService{}

	testClusters := []metastore.ClusterInfo{
		{
			ID:        1,
			Name:      "test_cluster_1",
			ClusterID: "test_cluster_1",
		},
	}

	svc.On("ListClusters", mock.Anything).Return(testClusters, nil)

	loggerConfig := zap.NewProductionConfig()
	loggerConfig.OutputPaths = []string{"stdout"}
	logger, err := loggerConfig.Build()
	assert.NoError(t, err)

	appConfig := testutils.GetConfig()
	ch, err := NewClusterHandler(svc, appConfig, logger)
	server := createServer(ch)

	t.Run("fails for unauthenticated users", func(t *testing.T) {
		listRequest, err := http.NewRequest(http.MethodGet, "/listcluster", nil)
		assert.NoError(t, err)
		response := executeRequest(server, listRequest)
		assert.Equal(t, http.StatusUnauthorized, response.Code)
	})

}
