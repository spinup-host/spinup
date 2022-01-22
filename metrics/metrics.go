package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spinup-host/api"
	"github.com/spinup-host/config"
)

func HandleMetrics(w http.ResponseWriter, req *http.Request) {
	if (*req).Method != "GET" {
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}
	authHeader := req.Header.Get("Authorization")
	apiKeyHeader := req.Header.Get("x-api-key")
	var err error
	config.Cfg.UserID, err = config.ValidateUser(authHeader, apiKeyHeader)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
	}

	recordMetrics()
	promhttp.Handler().ServeHTTP(w, req)
}

var (
	containersCreated = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "spinup_containers_created_gauge",
		Help: "The total number of containers created by spinup",
	})
)

func recordMetrics() {
	go func() {
		for {
			time.Sleep(2 * time.Second)
			dbPath := config.Cfg.Common.ProjectDir + "/" + config.Cfg.UserID
			clusterInfos := api.ReadClusterInfo(dbPath, "viggy28")
			containersCreated.Set(float64(len(clusterInfos)))
		}
	}()
}
