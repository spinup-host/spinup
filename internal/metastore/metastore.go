package metastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

type Db struct {
	Client *sql.DB
}

type ClusterInfo struct {
	// one of arm64v8 or arm32v7 or amd64
	Architecture string `json:"architecture,omitempty"`

	Type       string `json:"type"` // only "postgres" is supported at the moment
	Host       string `json:"host"`
	ID         int    `json:"id,omitempty"`
	ClusterID  string `json:"cluster_id"`
	Name       string `json:"name"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	MajVersion int    `json:"majversion"`
	MinVersion int    `json:"minversion"`
	Monitoring string `json:"monitoring,omitempty"`
	CPU        int64  `json:"cpu,omitempty"`
	Memory     int64  `json:"memory,omitempty"`

	BackupEnabled bool         `json:"backup_enabled,omitempty"`
	Backup        BackupConfig `json:"backup,omitempty"`
}

type BackupConfig struct {
	// https://man7.org/linux/man-pages/man5/crontab.5.html
	Schedule map[string]interface{}
	Dest     Destination `json:"Dest"`
}

type Destination struct {
	Name         string
	BucketName   string
	ApiKeyID     string
	ApiKeySecret string
}

// clustersInfo type has methods which provide us to filter them by name etc.
type clustersInfo []ClusterInfo

// FilterByName filters cluster by name. Since cluster names are unique as soon as we find a match we return
func (c clustersInfo) FilterByName(name string) (ClusterInfo, error) {
	for _, clusterInfo := range c {
		if clusterInfo.Name == name {
			return clusterInfo, nil
		}
	}
	return ClusterInfo{}, errors.New("cluster not found")
}

func NewDb(path string) (Db, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return Db{}, fmt.Errorf("unable to create a new db sqlite db client %w", err)
	}
	return Db{Client: db}, nil
}

// migration creates table
func migration(ctx context.Context, db Db) error {
	sqlStatements := []string{
		"create table if not exists clusterInfo (id integer not null primary key autoincrement, clusterId text, name text, username text, password text, port integer, majVersion integer, minVersion integer);",
		"create table if not exists backup (id integer not null primary key autoincrement, clusterid text, destination text, bucket text, second integer, minute integer, hour integer, dom integer, month integer, dow integer, foreign key(clusterid) references clusterinfo(clusterid));",
	}
	tx, err := db.Client.Begin()
	if err != nil {
		return fmt.Errorf("couldn't begin a transaction %w", err)

	}
	for _, sqlStatement := range sqlStatements {
		_, err = tx.ExecContext(ctx, sqlStatement)
		if err != nil {
			return fmt.Errorf("couldn't execute a transaction for %s %w", sqlStatement, err)
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("couldn't commit a transaction %w", err)
	}
	return nil
}

// InsertService adds a new row containing the cluster/service info to the database.
// TODO: How to write generic functions with varying fields and types? Maybe generics
func InsertService(db Db, cluster ClusterInfo) error {
	query := "insert into clusterInfo(clusterId, name, username, password, port, majVersion, minVersion) values(?, ?, ?, ?, ?, ?, ?)"
	tx, err := db.Client.Begin()
	if err != nil {
		return fmt.Errorf("unable to begin a transaction %w", err)
	}
	if err = migration(context.Background(), db); err != nil {
		return fmt.Errorf("error running a migration %w", err)
	}
	_, err = tx.ExecContext(context.Background(), query, cluster.ClusterID, cluster.Name, cluster.Username, cluster.Password, cluster.Port, cluster.MajVersion, cluster.MinVersion)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Println("ERROR: failed to rollback transaction: ", rollbackErr)
		}
		return fmt.Errorf("unable to execute %s %v", query, err)
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func InsertBackup(db Db, sql, clusterId, destination, bucket string, second, minute, hour, dom, month, dow int) error {
	tx, err := db.Client.Begin()
	if err != nil {
		return fmt.Errorf("unable to begin a transaction %w", err)
	}
	res, err := tx.ExecContext(context.Background(), sql, clusterId, destination, bucket, second, minute, hour, dom, month, dow)
	if err != nil {
		return fmt.Errorf("unable to execute %s %v", sql, err)
	}
	rows, _ := res.RowsAffected()
	log.Println("INFO: rows inserted into backup table:", rows)
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

// AllClusters returns all clusters from clusterinfo table
func AllClusters(db Db) (clustersInfo, error) {
	if err := db.Client.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging sqlite database %w", err)
	}
	if err := migration(context.Background(), db); err != nil {
		return nil, fmt.Errorf("error running a migration %w", err)
	}
	rows, err := db.Client.Query("select id, clusterId, name, username, password, port, majversion, minversion from clusterInfo")
	if err != nil {
		return nil, fmt.Errorf("unable to query clusterinfo")
	}
	defer rows.Close()
	var csi clustersInfo
	var cluster ClusterInfo
	for rows.Next() {
		err = rows.Scan(&cluster.ID, &cluster.ClusterID, &cluster.Name, &cluster.Username, &cluster.Password, &cluster.Port, &cluster.MajVersion, &cluster.MinVersion)
		if err != nil {
			log.Fatal(err)
		}
		cluster.Host = "localhost"
		csi = append(csi, cluster)
	}
	return csi, nil
}

// GetClusterByID returns info about the cluster whose ID is provided.
func GetClusterByID(db Db, clusterId string) (ClusterInfo, error) {
	var ci ClusterInfo
	query := `SELECT id, clusterId, name, username, password, port, majVersion, minVersion FROM clusterInfo WHERE clusterId = ? LIMIT 1`
	err := db.Client.QueryRow(query, clusterId).Scan(
		&ci.ID,
		&ci.ClusterID,
		&ci.Name,
		&ci.Username,
		&ci.Password,
		&ci.Port,
		&ci.MajVersion,
		&ci.MinVersion,
	)
	ci.Host = "localhost" // filled since we don't save the host yet.
	if errors.Is(err, sql.ErrNoRows) {
		return ci, errors.New(fmt.Sprintf("no cluster with ID: '%s' was found", clusterId))
	}
	return ci, err
}

// GetClusterByName returns info about the cluster whose name is provided.
func GetClusterByName(db Db, clusterName string) (ClusterInfo, error) {
	var ci ClusterInfo
	query := `SELECT id, clusterId, name, username, password, port, majVersion, minVersion FROM clusterInfo WHERE name = ? LIMIT 1`
	err := db.Client.QueryRow(query, clusterName).Scan(
		&ci.ID,
		&ci.ClusterID,
		&ci.Name,
		&ci.Username,
		&ci.Password,
		&ci.Port,
		&ci.MajVersion,
		&ci.MinVersion,
	)
	ci.Host = "localhost" // filled since we don't save the host yet.
	return ci, err
}
