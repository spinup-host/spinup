package metrics

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spinup-host/spinup/config"
	"github.com/spinup-host/spinup/internal/metastore"
)

type MetricsHandler struct {
	Db metastore.Db
}

func NewMetricsHandler() (MetricsHandler, error) {
	path := filepath.Join(config.Cfg.Common.ProjectDir, "metastore.db")
	log.Println("remove path:", path)
	db, err := metastore.NewDb(path)
	if err != nil {
		return MetricsHandler{}, err
	}
	return MetricsHandler{
		Db: db,
	}, nil
}

func (m *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if (*r).Method != "GET" {
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}
	authHeader := r.Header.Get("Authorization")
	apiKeyHeader := r.Header.Get("x-api-key")
	var err error
	config.Cfg.UserID, err = config.ValidateUser(authHeader, apiKeyHeader)
	if err != nil {
		log.Printf(err.Error())
		http.Error(w, "error validating user", http.StatusUnauthorized)
		return
	}
	errCh := make(chan error, 1)
	recordMetrics(m.Db, errCh)
	promhttp.Handler().ServeHTTP(w, r)
}

var (
	containersCreated = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "spinup_containers_created_gauge",
		Help: "The total number of containers created by spinup",
	})
)

func recordMetrics(db metastore.Db, errCh chan error) {
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
