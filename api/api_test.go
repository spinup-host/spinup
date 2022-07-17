package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

const TestAppPort = 23455

func createServer(ch ClusterHandler) *http.Server {
	addr := fmt.Sprintf(":%d", TestAppPort)

	router := http.NewServeMux()
	router.HandleFunc("/hello", Hello)
	router.HandleFunc("/createservice", ch.CreateService)
	router.HandleFunc("/listcluster", ch.ListCluster)
	router.HandleFunc("/cluster", ch.GetCluster)

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}
	return srv
}

func executeRequest(srv *http.Server, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rr, req)

	return rr
}
