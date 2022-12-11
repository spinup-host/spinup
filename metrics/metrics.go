package metrics

import (
	"fmt"
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
	Db        metastore.Db
	appConfig config.Configuration
}

func NewMetricsHandler(cfg config.Configuration) (MetricsHandler, error) {
	path := filepath.Join(cfg.Common.ProjectDir, "metastore.db")
	db, err := metastore.NewDb(path)
	if err != nil {
		return MetricsHandler{}, err
	}
	return MetricsHandler{
		Db:        db,
		appConfig: cfg,
	}, nil
}

func (m *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if (*r).Method != "GET" {
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}
	errCh := make(chan error, 1)
	recordMetrics(m.Db, errCh)
	promhttp.Handler().ServeHTTP(w, r)
}

var containersCreated = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "spinup_containers_created_gauge",
	Help: "The total number of containers created by spinup",
})

func recordMetrics(db metastore.Db, errCh chan error) {
	go func() {
		for {
			time.Sleep(2 * time.Second)
			clusterInfos, err := metastore.AllClusters(db)
			if err != nil {
				errCh <- fmt.Errorf("couldn't read cluster info %w", err)
				close(errCh)
			}
			containersCreated.Set(float64(len(clusterInfos)))
		}
	}()
}
