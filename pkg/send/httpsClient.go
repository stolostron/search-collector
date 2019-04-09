/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package send

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"

	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/config"

	"github.com/golang/glog"
)

func getHTTPSClient() (client http.Client) {
	caCert, err := ioutil.ReadFile("./sslcert/tls.crt")
	if err != nil {
		glog.Error("Error loading TLS certificate. Certificate must be mounted at ./sslcert/tls.crt")
		glog.Error(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair("./sslcert/tls.crt", "./sslcert/tls.key")
	if err != nil {
		glog.Error("Error loading TLS certificate. Certificate must be mounted at ./sslcert/tls.crt and ./sslcert/tls.key")
		glog.Error(err)
	}

	// Configure TLS
	tlsCfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			// TODO: Update list with acceptable FIPS ciphers.
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	// TLS is different when deployed in hub vs klusterlet.
	if !config.Cfg.DeployedInHub {
		glog.Warning("Using insecure HTTPS client.")
		tlsCfg.InsecureSkipVerify = true
	} else {
		tlsCfg.RootCAs = caCertPool
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	tr := &http.Transport{
		TLSClientConfig: tlsCfg,
	}

	client = http.Client{Transport: tr}

	return client
}
