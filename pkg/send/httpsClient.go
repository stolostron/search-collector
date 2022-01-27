/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
*/
// Copyright (c) 2020 Red Hat, Inc.

package send

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"

	"github.com/stolostron/search-collector/pkg/config"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured/unstructuredscheme"
	"k8s.io/client-go/rest"

	"github.com/golang/glog"
)

func getHTTPSClient() (client http.Client) {

	// Klusterlet deployment: Get httpClient using the mounted kubeconfig.
	if !config.Cfg.DeployedInHub {
		config.Cfg.AggregatorConfig.NegotiatedSerializer = unstructuredscheme.NewUnstructuredNegotiatedSerializer()
		aggregatorRESTClient, err := rest.UnversionedRESTClientFor(config.Cfg.AggregatorConfig)
		if err != nil {
			// Exit because this is an unrecoverable configuration problem.
			glog.Fatal("Error getting httpClient from kubeconfig. Original error: ", err)
		}
		client = *(aggregatorRESTClient.Client)
		return client
	} else {
		// Hub deployment: Generate TLS config using the mounted certificates.
		caCert, err := ioutil.ReadFile("./sslcert/tls.crt")
		if err != nil {
			// Exit because this is an unrecoverable configuration problem.
			glog.Fatal("Error loading TLS certificate from mounted secret. Certificate must be mounted at ./sslcert/tls.crt  Original error: ", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		cert, err := tls.LoadX509KeyPair("./sslcert/tls.crt", "./sslcert/tls.key")
		if err != nil {
			// Exit because this is an unrecoverable configuration problem.
			glog.Fatal("Error loading TLS certificate from mounted secret. Certificate must be mounted at ./sslcert/tls.crt and ./sslcert/tls.key  Original error: ", err)
		}

		// Configure TLS
		tlsCfg := &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			},
			RootCAs:      caCertPool,
			Certificates: []tls.Certificate{cert},
		}

		tr := &http.Transport{
			TLSClientConfig: tlsCfg,
		}

		client = http.Client{Transport: tr}

		return client
	}
}
