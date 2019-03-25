package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/tkanos/gonfig"
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

	err = gonfig.GetConf(*FilePath, &Cfg)
	if err != nil { // Check for stinkies, if this fails we can't really recover
		fmt.Println("Error reading config file:", err) // Uses fmt.Println in case something is wrong with glog args
		os.Exit(1)
	}
	glog.Info("Successfully read from config file: ", *FilePath)
}
