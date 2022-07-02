package metastore

import (
	"context"
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServices(t *testing.T) {
	t.Parallel()
	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)

	path := filepath.Join(tmpDir, "test.db")
	defer func(name string) {
		_ = os.Remove(name)
	}(path)

	db, err := NewDb(path)
	require.NoError(t, err)

	clusters := []ClusterInfo{
		{
			Host: "localhost",
			Name: "db1",
			ClusterID: generateID("db1"),
			Username: "user1",
			Password: "password1",
			Port: 9001,
			MajVersion: 13,
			MinVersion: 0,
		},{
			Host: "localhost",
			Name: "db2",
			ClusterID: generateID("db2"),
			Username: "user2",
			Password: "password2",
			Port: 9001,
			MajVersion: 13,
			MinVersion: 0,
		},{
			Host: "localhost",
			Name: "db3",
			ClusterID: generateID("db3"),
			Username: "user3",
			Password: "password3",
			Port: 9002,
			MajVersion: 13,
			MinVersion: 0,
		},{
			Host: "localhost",
			Name: "db4",
			ClusterID: generateID("db4"),
			Username: "user4",
			Password: "password4",
			Port: 9003,
			MajVersion: 13,
			MinVersion: 0,
		},
	}

	t.Run("migration fails for bad client", func(t *testing.T) {
		t.Parallel()
		badDB, err := NewDb(os.TempDir()) // attempt to use a directory as sqlite path should fail
		assert.NoError(t, err)
		require.Error(t, migration(context.TODO(), badDB))
	})

	t.Run("filter by name", func(t *testing.T) {
		ci := clustersInfo(clusters)
		match, err := ci.FilterByName("db1")
		assert.NoError(t, err)
		assert.Equal(t, clusters[0], match)

		noMatch, err := ci.FilterByName("random_db")
		assert.Error(t, err)
		assert.Equal(t, "", noMatch.Name)
	})

	t.Run("insert service", func(t *testing.T) {
		for _, cluster := range clusters {
			assert.NoError(t, InsertService(db, cluster))
		}
	})

	t.Run("get all clusters", func(t *testing.T) {
		result, err := AllClusters(db)

		assert.NoError(t, err)
		assert.Equal(t, len(clusters), len(result))
	})

	t.Run("get cluster by ID", func(t *testing.T) {
		result, err := GetClusterByID(db, generateID("db1"))
		assert.NoError(t, err)

		// we don't compare the struct directly since it may have been modified (e.g., auto-incremented ID field is added)
		// and thus, won't be equal.
		assert.Equal(t, clusters[0].ClusterID, result.ClusterID)
	})
}

func generateID(name string) string {
	sha := sha1.New()
	sha.Write([]byte(name))
	return fmt.Sprintf("%x", sha.Sum(nil))
}
