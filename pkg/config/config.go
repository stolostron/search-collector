package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/tkanos/gonfig"
)

const (
	DEFAULT_CLUSTER_NAME   = "localtest"
	DEFAULT_AGGREGATOR_URL = "http://localhost:3010"
)

// Define a config type for gonfig to hold our config properties.
type Config struct {
	AggregatorURL string `env:"AGGREGATOR_URL"` // URL of the Aggregator, includes port but not any path
	ClusterName   string `env:"CLUSTER_NAME"`   // The name of this cluster
}

var Cfg = Config{}
var FilePath = flag.String("c", "./config.json", "Collector configuration file") // -c example.json, config.json is the default

func init() {
	// parse flags
	flag.Parse()
	err := flag.Lookup("logtostderr").Value.Set("true") // Glog is weird in that by default it logs to a file. Change it so that by default it all goes to stderr. (no option for stdout).
	if err != nil {
		fmt.Println("Error setting default flag:", err) // Uses fmt.Println in case something is wrong with glog args
		os.Exit(1)
		glog.Fatal("Error setting default flag: ", err)
	}
	defer glog.Flush() // This should ensure that everything makes it out on to the console if the program crashes.

	// Load default config from ./config.json.
	// These can be overriden in the next step if environment variables are set.
	if _, err := os.Stat(filepath.Join(".", "config.json")); !os.IsNotExist(err) {
		err = gonfig.GetConf(*FilePath, &Cfg)
		if err != nil {
			fmt.Println("Error reading config file:", err) // Uses fmt.Println in case something is wrong with glog args
		}
		glog.Info("Successfully read from config file: ", *FilePath)
	} else {
		glog.Warning("Missing config file: ./config.json.")
	}

	// If environment variables are set, use those values instead of ./config.json
	if clusterName := os.Getenv("CLUSTER_NAME"); clusterName != "" {
		glog.Info("Using CLUSTER_NAME from environment: ", clusterName)
		Cfg.ClusterName = clusterName
	} else if Cfg.ClusterName == "" {
		glog.Warning("No ClusterName from file or environment, using default ClusterName: ", DEFAULT_CLUSTER_NAME)
		Cfg.ClusterName = DEFAULT_CLUSTER_NAME
	}
	if aggregatorURL := os.Getenv("AGGREGATOR_URL"); aggregatorURL != "" {
		glog.Info("Using AGGREGATOR_URL from environment: ", aggregatorURL)
		Cfg.AggregatorURL = aggregatorURL
	} else if Cfg.AggregatorURL == "" {
		glog.Warning("No AggregatorURL from file or environment, using default AggregatorURL: ", DEFAULT_AGGREGATOR_URL)
		Cfg.AggregatorURL = DEFAULT_AGGREGATOR_URL
	}
}
