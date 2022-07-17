package api

import (
	"context"

	"github.com/spinup-host/spinup/internal/metastore"
)


// clusterService provides an interface for API handlers to manage clusters
//go:generate mockery --name=clusterService --case=snake --inpackage --testonly
type clusterService interface {
	CreateService(ctx context.Context, info *metastore.ClusterInfo) error
	ListClusters(ctx context.Context) ([]metastore.ClusterInfo, error)
	GetClusterByID(ctx context.Context, clusterID string) (metastore.ClusterInfo, error)
}
