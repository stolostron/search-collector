/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020, 2021 Red Hat, Inc.
*/

package config

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"github.com/golang/glog"
	"github.com/tkanos/gonfig"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Out of box defaults
const (
	COLLECTOR_API_VERSION      = "2.2.0"
	DEFAULT_AGGREGATOR_URL     = "https://localhost:3010" // this will be deprecated in the future
	DEFAULT_AGGREGATOR_HOST    = "https://localhost"
	DEFAULT_AGGREGATOR_PORT    = "3010"
	DEFAULT_CLUSTER_NAME       = "local-cluster"
	DEFAULT_HEARTBEAT_MS       = 300000 // 5 min
	DEFAULT_MAX_BACKOFF_MS     = 600000 // 10 min
	DEFAULT_REDISCOVER_RATE_MS = 120000 // 2 min
	DEFAULT_REPORT_RATE_MS     = 5000   // 5 seconds
	DEFAULT_RUNTIME_MODE       = "production"
)

// Configuration options for the search-collector.
type Config struct {
	AggregatorConfig *rest.Config // Config object for hub. Used to get TLS credentials.

	AggregatorConfigFile string `env:"HUB_CONFIG"`         // Config file for hub. Will be mounted in a secret.
	AggregatorURL        string `env:"AGGREGATOR_URL"`     // URL of the Aggregator, includes port but not any path
	AggregatorHost       string `env:"AGGREGATOR_HOST"`    // Host of the Aggregator
	AggregatorPort       string `env:"AGGREGATOR_PORT"`    // Port of the Aggregator
	ClusterName          string `env:"CLUSTER_NAME"`       // The name of this cluster
	ClusterNamespace     string `env:"CLUSTER_NAMESPACE"`  // The namespace of this cluster
	DeployedInHub        bool   `env:"DEPLOYED_IN_HUB"`    // Tracks if deployed in the Hub or Managed cluster
	HeartbeatMS          int    `env:"HEARTBEAT_MS"`       // Interval(ms) to send empty payload to ensure connection
	KubeConfig           string `env:"KUBECONFIG"`         // Local kubeconfig path
	MaxBackoffMS         int    `env:"MAX_BACKOFF_MS"`     // Maximum backoff in ms to wait after error
	RediscoverRateMS     int    `env:"REDISCOVER_RATE_MS"` // Interval(ms) to poll for changes to CRDs
	ReportRateMS         int    `env:"REPORT_RATE_MS"`     // Interval(ms) to send changes to the aggregator
	RuntimeMode          string `env:"RUNTIME_MODE"`       // Running mode (development or production)
}

var Cfg = Config{}
var FilePath = flag.String("c", "./config.json", "Collector configuration file") // ./config.json is the default

func init() {
	// Load default config from ./config.json.
	// These can be overridden in the next step if environment variables are set.
	if _, err := os.Stat(filepath.Join(".", "config.json")); !os.IsNotExist(err) {
		err = gonfig.GetConf(*FilePath, &Cfg)
		if err != nil {
			fmt.Println("Error reading config file:", err) // Uses fmt.Println in case something is wrong with glog
		}
		glog.Info("Successfully read from config file: ", *FilePath)
	} else {
		glog.Warning("Missing config file: ./config.json.")
	}

	// If environment variables are set, use those values instead of ./config.json
	// Simply put, the order of preference is env -> config.json -> default constants (from left to right)
	setDefault(&Cfg.RuntimeMode, "RUNTIME_MODE", DEFAULT_RUNTIME_MODE)
	setDefault(&Cfg.ClusterName, "CLUSTER_NAME", DEFAULT_CLUSTER_NAME)
	setDefault(&Cfg.ClusterNamespace, "CLUSTER_NAMESPACE", "")

	setDefault(&Cfg.AggregatorHost, "AGGREGATOR_HOST", DEFAULT_AGGREGATOR_HOST)
	setDefault(&Cfg.AggregatorPort, "AGGREGATOR_PORT", DEFAULT_AGGREGATOR_PORT)
	aggHost, aggHostPresent := os.LookupEnv("AGGREGATOR_HOST")
	aggPort, aggPortPresent := os.LookupEnv("AGGREGATOR_PORT")

	//If environment variables are set for aggregator host and port, use those to set the AggregatorURL
	if aggHostPresent && aggPortPresent && aggHost != "" && aggPort != "" {
		Cfg.AggregatorURL = net.JoinHostPort(aggHost, aggPort)
		setDefault(&Cfg.AggregatorURL, "", net.JoinHostPort(DEFAULT_AGGREGATOR_HOST, DEFAULT_AGGREGATOR_PORT))
	} else { // Else use the default AggregatorURL
		setDefault(&Cfg.AggregatorURL, "AGGREGATOR_URL", DEFAULT_AGGREGATOR_URL)
	}

	setDefaultInt(&Cfg.HeartbeatMS, "HEARTBEAT_MS", DEFAULT_HEARTBEAT_MS)
	setDefaultInt(&Cfg.MaxBackoffMS, "MAX_BACKOFF_MS", DEFAULT_MAX_BACKOFF_MS)
	setDefaultInt(&Cfg.RediscoverRateMS, "REDISCOVER_RATE_MS", DEFAULT_REDISCOVER_RATE_MS)
	setDefaultInt(&Cfg.ReportRateMS, "REPORT_RATE_MS", DEFAULT_REPORT_RATE_MS)

	defaultKubePath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	if _, err := os.Stat(defaultKubePath); os.IsNotExist(err) {
		// set default to empty string if path does not reslove
		defaultKubePath = ""
	}
	setDefault(&Cfg.KubeConfig, "KUBECONFIG", defaultKubePath)

	// Special logic for setting DEPLOYED_IN_HUB with default to false
	if val := os.Getenv("DEPLOYED_IN_HUB"); val != "" {
		glog.Infof("Using DEPLOYED_IN_HUB from environment: %s", val)
		var err error
		Cfg.DeployedInHub, err = strconv.ParseBool(val)
		if err != nil {
			glog.Error("Error parsing env DEPLOYED_IN_HUB.  Expected a bool.  Original error: ", err)
			glog.Info("Leaving flag unchanged, assuming it is a Klusterlet")
		}
	} else if !Cfg.DeployedInHub {
		glog.Info("No DEPLOY_IN_HUB from file or environment, assuming it is a Klusterlet")
	}
	setDefault(&Cfg.AggregatorConfigFile, "HUB_CONFIG", "")

	if Cfg.DeployedInHub && Cfg.AggregatorConfigFile != "" {
		glog.Fatal("Config mismatch: DEPLOYED_IN_HUB is true, but HUB_CONFIG is set to connect to another hub")
	} else if !Cfg.DeployedInHub && Cfg.AggregatorConfigFile == "" {
		glog.Fatal("Config mismatch: DEPLOYED_IN_HUB is false, but no HUB_CONFIG is set to connect to another hub")
	}

	if Cfg.AggregatorConfigFile != "" {
		hubConfig, err := clientcmd.BuildConfigFromFlags("", Cfg.AggregatorConfigFile)
		if err != nil {
			glog.Error("Error building K8s client from config file [", Cfg.AggregatorConfigFile, "]. Original error: ",
				err)
		}

		Cfg.AggregatorURL = hubConfig.Host + "/apis/proxy.open-cluster-management.io/v1beta1/namespaces/" +
			Cfg.ClusterNamespace + "/clusterstatuses"
		Cfg.AggregatorConfig = hubConfig

		glog.Info("Running inside klusterlet. Aggregator URL: ", Cfg.AggregatorURL)
	}
}

// Sets config field to perfer the env over config file
// If no config or env set to the default value
func setDefault(field *string, env, defaultVal string) {
	if val := os.Getenv(env); val != "" {
		glog.Infof("Using %s from environment: %s", env, val)
		*field = val
	} else if *field == "" && defaultVal != "" {
		glog.Infof("No %s from file or environment, using default value: %s", env, defaultVal)
		*field = defaultVal
	}
}

// TODO: Combine with function above.
func setDefaultInt(field *int, env string, defaultVal int) {
	if val := os.Getenv(env); val != "" {
		glog.Infof("Using %s from environment: %s", env, val)
		var err error
		*field, err = strconv.Atoi(val)
		if err != nil {
			glog.Error("Error parsing env [", env, "].  Expected an integer.  Original error: ", err)
		}
	} else if *field == 0 && defaultVal != 0 {
		glog.Infof("No %s from file or environment, using default value: %d", env, defaultVal)
		*field = defaultVal
	}
}
