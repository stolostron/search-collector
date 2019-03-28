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
	DEFAULT_RUNTIME_MODE   = "production"
	DEFAULT_CLUSTER_NAME   = "localtest"
	DEFAULT_AGGREGATOR_URL = "https://localhost:3010"
)

// Define a config type for gonfig to hold our config properties.
type Config struct {
	RuntimeMode   string `env:"RUNTIME_MODE"`   // Running mode (development or production)
	AggregatorURL string `env:"AGGREGATOR_URL"` // URL of the Aggregator, includes port but not any path
	ClusterName   string `env:"CLUSTER_NAME"`   // The name of this cluster
	KubeConfig    string `env:"KUBECONFIG"`     // Local kubeconfig path
	TillerURL     string `env:"TILLER_URL"`     // URL host path of the tiller server
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
	// These can be overridden in the next step if environment variables are set.
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
	// Simply put, the order of preference is env -> config.json -> default constants (from left to right)
	setDefault(&Cfg.RuntimeMode, "RUNTIME_MODE", DEFAULT_RUNTIME_MODE)
	setDefault(&Cfg.ClusterName, "CLUSTER_NAME", DEFAULT_AGGREGATOR_URL)
	setDefault(&Cfg.AggregatorURL, "AGGREGATOR_URL", DEFAULT_AGGREGATOR_URL)

	defaultKubePath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	if _, err := os.Stat(defaultKubePath); os.IsNotExist(err) {
		// set default to empty string if path does not reslove
		defaultKubePath = ""
	}
	setDefault(&Cfg.KubeConfig, "KUBECONFIG", defaultKubePath)

	// Special case for default url, this is set below
	setDefault(&Cfg.TillerURL, "TILLER_URL", "")
	// TODO: tiller url coming soon to a pr near you!!
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
