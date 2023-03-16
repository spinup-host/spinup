package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/spinup-host/spinup/config"
	"github.com/spinup-host/spinup/internal/metastore"
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

	svc.On("ListClusters", mock.Anything, ).Return(testClusters, nil)

	loggerCfg := zap.NewProductionConfig()
	loggerCfg.OutputPaths = []string{"stdout"}
	logger, err := loggerCfg.Build()
	assert.NoError(t, err)

	appCfg := config.Configuration{}

	ch, err := NewClusterHandler(svc, appCfg, logger)
	server := createServer(ch)

	t.Run("fails for unauthenticated users", func(t *testing.T) {
		listRequest, err := http.NewRequest(http.MethodGet, "/listcluster", nil)
		assert.NoError(t, err)
		response := executeRequest(server, listRequest)
		assert.Equal(t, http.StatusUnauthorized, response.Code)
	})

}
