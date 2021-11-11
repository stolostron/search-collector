package lease

import (
	"crypto/tls"
	"net"
	"net/http"

	"github.com/golang/glog"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

// ServeHealthProbes starts a server to check healthz and readyz probes
func ServeHealthProbes(healthProbeBindAddress string, configCheck healthz.Checker) {
	healthzHandler := &healthz.Handler{Checks: map[string]healthz.Checker{
		"healthz-ping": healthz.Ping,
		"configz-ping": configCheck,
	}}
	readyzHandler := &healthz.Handler{Checks: map[string]healthz.Checker{
		"readyz-ping": healthz.Ping,
	}}

	mux := http.NewServeMux()
	mux.Handle("/readyz", http.StripPrefix("/readyz", readyzHandler))
	mux.Handle("/healthz", http.StripPrefix("/healthz", healthzHandler))

	// Configure TLS
	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		},
	}
	server := http.Server{
		Handler:   mux,
		TLSConfig: cfg,
	}

	ln, err := net.Listen("tcp", healthProbeBindAddress)
	if err != nil {
		glog.Errorf("error listening on %s: %v", ":8000", err)
		return
	}

	glog.Infof("health probes server is running...")
	// Run server
	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			glog.Fatal(err, "health probe server not running due to error")
		}
	}()
}
