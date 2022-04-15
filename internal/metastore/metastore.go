package metastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/spinup-host/spinup/config"
)

type Db struct {
	Client *sql.DB
}

// clustersInfo type has methods which provide us to filter them by name etc.
type clustersInfo []config.ClusterInfo

// FilterByName filters cluster by name. Since cluster names are unique as soon as we find a match we return.
func (c clustersInfo) FilterByName(name string) (config.ClusterInfo, error) {
	for _, clusterInfo := range c {
		if clusterInfo.Name == name {
			return clusterInfo, nil
		}
	}
	return config.ClusterInfo{}, errors.New("cluster not found")
}

func NewDb(path string) (Db, error) {
	db, err := open(path)
	if err != nil {
		return Db{}, fmt.Errorf("unable to create a new db sqlite db client %w", err)
	}
	return Db{Client: db}, nil
}

func open(path string) (*sql.DB, error) {
	return sql.Open("sqlite", path)
}

// migration creates table.
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

// TODO: How to write generic functions with varying fields and types? Maybe generics.
func InsertServiceIntoMeta(db Db, sql, clusterId, name, username, password string, port, majVersion, minVersion int) error {
	tx, err := db.Client.Begin()
	if err != nil {
		return fmt.Errorf("unable to begin a transaction %w", err)
	}
	if err = migration(context.Background(), db); err != nil {
		return fmt.Errorf("error running a migration %w", err)
	}
	res, err := tx.ExecContext(context.Background(), sql, clusterId, name, username, password, port, majVersion, minVersion)
	if err != nil {
		return fmt.Errorf("unable to execute %s %v", sql, err)
	}
	rows, _ := res.RowsAffected()
	log.Println("INFO: rows inserted into clusterInfo table:", rows)
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func InsertBackupIntoMeta(db Db, sql, clusterId, destination, bucket string, second, minute, hour, dom, month, dow int) error {
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

// ClustersInfo returns all clusters from clusterinfo table.
func ClustersInfo(db Db) (clustersInfo, error) {
	if err := db.Client.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging sqlite database %w", err)
	}
	if err := migration(context.Background(), db); err != nil {
		return nil, fmt.Errorf("error running a migration %w", err)
	}
	rows, err := db.Client.Query("select id, clusterId, name, username, port, majversion, minversion from clusterInfo")
	if err != nil {
		return nil, fmt.Errorf("unable to query clusterinfo")
	}
	defer rows.Close()
	var csi clustersInfo
	var cluster config.ClusterInfo
	for rows.Next() {
		err = rows.Scan(&cluster.ID, &cluster.ClusterID, &cluster.Name, &cluster.Username, &cluster.Port, &cluster.MajVersion, &cluster.MinVersion)
		if err != nil {
			log.Fatal(err)
		}
		csi = append(csi, cluster)
	}
	return csi, nil
}
