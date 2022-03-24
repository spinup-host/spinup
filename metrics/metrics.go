package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spinup-host/config"
	"github.com/spinup-host/internal/metastore"
)

func HandleMetrics(w http.ResponseWriter, req *http.Request) {
	if (*req).Method != "GET" {
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}
	/* 	authHeader := req.Header.Get("Authorization")
	   	apiKeyHeader := req.Header.Get("x-api-key")
	   	var err error
	   	config.Cfg.UserID, err = config.ValidateUser(authHeader, apiKeyHeader)
	   	if err != nil {
	   		log.Printf(err.Error())
	   		http.Error(w, "error validating user", http.StatusUnauthorized)
	   		return
	   	} */
	errCh := make(chan error, 1)
	recordMetrics(errCh)
	promhttp.Handler().ServeHTTP(w, req)
}

var (
	containersCreated = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "spinup_containers_created_gauge",
		Help: "The total number of containers created by spinup",
	})
)

var db metastore.Db
var err error

func recordMetrics(errCh chan error) {
	dbPath := config.Cfg.Common.ProjectDir + "/" + config.Cfg.UserID + "/" + config.Cfg.UserID + ".db"
	if db.Client == nil {
		db, err = metastore.NewDb(dbPath)
		if err != nil {
			return
		}
	}
	go func() {
		for {
			time.Sleep(2 * time.Second)
			clusterInfos, err := metastore.ClustersInfo(db)
			if err != nil {
				errCh <- fmt.Errorf("couldn't read cluster info %w", err)
				close(errCh)
			}
			containersCreated.Set(float64(len(clusterInfos)))
		}
	}()
}
