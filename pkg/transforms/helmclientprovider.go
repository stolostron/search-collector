/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"k8s.io/helm/pkg/helm"
	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/config"
	"time"
	"github.com/golang/glog"
	"k8s.io/helm/pkg/tlsutil"
)

var helmClient *helm.Client
var helmReleaseRetry chan *Event
var inputChannel chan *Event

func StartHelmClientProvider(i chan *Event) {
	inputChannel = i
	InitializeRetryChannel();
	longSleep := time.Duration(60) * time.Second  // If successful helm connection, wait `longSleep` before checking connection/retry
	shortSleep := time.Duration(5) * time.Second  // If unsuccessful helm connection, wait `shortSleep` before trying to reconnect
	timeToSleep := longSleep

	startUp := true

	glog.Info("Beginning helmClient Cycle")
	go func() {
		for {
			if RetryNecessary() || startUp {
				startUp = false
				if HealthyConnection() {
					glog.V(3).Info("helmClient has healthy connection")
					timeToSleep = longSleep // helmClient exists, wait a full minute before checking again
					Retry();
				} else {
					helmTlsConfig, err := tlsutil.ClientConfig(config.Cfg.TillerOpts)
					if err != nil {
						glog.Warning("Error creating helm TLS configuration: ", err)
						timeToSleep = shortSleep // start checking again sooner rather than later
					} else {
						// Create Helm client for transformer
						helmClient = helm.NewClient(
							helm.WithTLS(helmTlsConfig),
							helm.Host(config.Cfg.TillerURL),
						)
						if HealthyConnection(){
							glog.V(3).Info("Created new healthy helm client")
							timeToSleep = longSleep // helmClient exists, wait a full minute before checking again
							Retry();
						} else {
							glog.Warning("Error creating helmClient: helmClient not healthy")
							timeToSleep = shortSleep // start checking again sooner rather than later
						}
					}
				}
			}
			time.Sleep(timeToSleep)
		}
	}()
}

func RetryNecessary() bool {
	return len(helmReleaseRetry) > 0
}

func HealthyConnection() bool {
	if helmClient != nil {
		if ping := helmClient.PingTiller(); ping == nil { // nil ping means good tiller connection
			return true
		}
	}
	return false
}

func GetHelmClient() *helm.Client {
	return helmClient
}

func InitializeRetryChannel() {
	maxInt16 := 1<<15 - 1 // need to buffer channel, make its capacity as large as possible
	glog.V(3).Info("Initializing retry channel")
	helmReleaseRetry = make(chan *Event, maxInt16)
}

func AddToRetryChannel( e *Event) {
	go func() {
		glog.Warning("Failed to retrieve release, adding Event to retry channel: ", e)
		helmReleaseRetry <- e
	}()
}

func Retry() {
	retryCount := len(helmReleaseRetry)
	glog.V(3).Info("Triggered HelmClientProvider retry for ", retryCount, " elements")
	elemCount := 0
	for elem := range helmReleaseRetry {
		if elemCount >= retryCount {
			break;
		}
		elemCount = elemCount +1
		inputChannel <- elem
	}
}