// Copyright Contributors to the Open Cluster Management project

package server

import (
	"k8s.io/klog/v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stolostron/search-collector/pkg/config"
	"github.com/stolostron/search-collector/pkg/metrics"
)

func StartAndListen() {
	router := mux.NewRouter()
	router.HandleFunc("/liveness", LivenessProbe).Methods("GET")
	router.HandleFunc("/readiness", ReadinessProbe).Methods("GET")
	router.Handle("/metrics", promhttp.HandlerFor(metrics.PromRegistry, promhttp.HandlerOpts{})).Methods("GET")

	srv := &http.Server{
		Addr:              config.Cfg.ServerAddress,
		Handler:           router,
		ReadHeaderTimeout: time.Duration(config.Cfg.HTTPTimeout) * time.Millisecond,
	}

	go func() {
		klog.Info("Listening on: ", srv.Addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			klog.Fatal(err, ". Encountered while starting the server.")
		}
	}()

	// Listen and wait for termination signal.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigs // Waits for termination signal.
	klog.Warningf("Received termination signal %s. Exiting server. ", sig)
}
