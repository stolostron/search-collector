/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.ibm.com/IBMPrivateCloud/search-collector/pkg/config"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/tlsutil"
)

var helmClient *helm.Client
var helmReleaseRetry chan *Event
var inputChannel chan *Event
var kubeClientConfig *rest.Config
var tillerPresent bool

var timeToSleep, shortSleep, longSleep time.Duration
var helmWarningPrinted bool

func StartHelmClientProvider(i chan *Event, kubeConfig *rest.Config) {
	inputChannel = i
	InitializeRetryChannel()
	longSleep = time.Duration(60) * time.Second // If successful helm connection, wait `longSleep` before checking connection/retry
	shortSleep = time.Duration(5) * time.Second // If unsuccessful helm connection, wait `shortSleep` before trying to reconnect
	timeToSleep = longSleep
	kubeClientConfig = kubeConfig

	startUp := true

	glog.Info("Beginning helmClient Cycle")
	go func() {
		for {
			if RetryNecessary() || startUp {
				startUp = false
				if HealthyConnection() {
					resetVars()
				} else {
					helmTlsConfig, err := tlsutil.ClientConfig(config.Cfg.TillerOpts)
					if err != nil {
						msg := fmt.Sprintf("Error creating helm TLS configuration: %s. Will retry after %s.", err.Error(), shortSleep)
						shortSleep = processConnError(shortSleep, msg)
						timeToSleep = shortSleep // start checking again sooner rather than later
					} else {
						// Create Helm client for transformer
						helmClient = helm.NewClient(
							helm.WithTLS(helmTlsConfig),
							helm.Host(config.Cfg.TillerURL),
						)
						if HealthyConnection() {
							resetVars()
						} else {
							msg := "Error creating helmClient: helmClient not healthy. Will retry after"
							shortSleep = processConnError(shortSleep, msg)
							timeToSleep = shortSleep // start checking again sooner rather than later
						}
					}
				}
			}
			time.Sleep(timeToSleep)
		}
	}()
}

func processConnError(shortSleep time.Duration, msg string) time.Duration {
	if !helmWarningPrinted {
		glog.Warningf("%s %s.", msg, shortSleep)
		helmWarningPrinted = true
	} else {
		if shortSleep.Hours() < 1 { //Maximum time between checks is 1 hour
			shortSleep = shortSleep * 5 // if Tiller Config is not available, gradually increase time between the checks
		}
		glog.V(3).Infof("%s %s.", msg, shortSleep)
	}
	return shortSleep
}

func resetVars() {
	glog.V(3).Info("helmClient has healthy connection")
	timeToSleep = longSleep // helmClient exists, wait a full minute before checking again
	helmWarningPrinted = false
	shortSleep = time.Duration(5) * time.Second
	Retry()
}
func RetryNecessary() bool {
	// Check if Tiller service is present, before trying to connect - disable retrying, if not present
	var options metaV1.GetOptions
	clientset, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		glog.Warning("Cannot construct kubernetes Client From Config: ", err)
	} else {
		tillerSvc, err := clientset.CoreV1().Services("kube-system").Get("tiller-deploy", options)
		if tillerSvc != nil && err == nil {
			tillerPresent = true
		} else {
			tillerPresent = false
			glog.V(3).Info("Cannot retrieve tiller service: ", err)
		}
	}
	return len(helmReleaseRetry) > 0 && tillerPresent
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

func AddToRetryChannel(e *Event) {
	go func() {
		if len(helmReleaseRetry) < 1 {
			glog.Warning("Failed to retrieve release, adding Event to retry channel: ", e)
		}
		helmReleaseRetry <- e
	}()
}

func Retry() {
	retryCount := len(helmReleaseRetry)
	glog.V(3).Info("Triggered HelmClientProvider retry for ", retryCount, " elements")
	elemCount := 0
	for elem := range helmReleaseRetry {
		if elemCount >= retryCount {
			break
		}
		elemCount = elemCount + 1
		inputChannel <- elem
	}
}
