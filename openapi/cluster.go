package openapi

import (
	"github.com/spinup-host/spinup/api"
	"github.com/spinup-host/spinup/internal/metastore"
)

// swagger:route GET /listcluster cluster listCluster
// List all created clusters.
//
//	Responses:
//		200: listClusterResponse
//		401: unauthorizedResponse

// listClusterResponseWrapper wraps a successful response when listing clusters.
// swagger:model listClusterResponse
type listClusterResponseWrapper struct {
	// in:body
	Data []metastore.ClusterInfo `json:"data"`
}

// unauthorizedResponseWrapper wraps an unauthorized response.
// swagger:model unauthorizedResponse
type unauthorizedResponseWrapper struct {
	// in:body
	Message string `json:"message"`
}

// swagger:route Post /create cluster createCluster
// Create a new cluster.
//
//	Responses:
//		200: createClusterResponse
//		401: unauthorizedResponse

// swagger:parameters createCluster
type createClusterParamsWrapper struct {
	// Parameters for create the new cluster
	// in:body
	Body api.Cluster
}

// createClusterResponseWrapper wraps a successful response after creating a cluster endpoint.
// swagger:model createClusterResponse
type createClusterResponseWrapper struct {
	// in:body
	Data metastore.ClusterInfo `json:"data"`
}
