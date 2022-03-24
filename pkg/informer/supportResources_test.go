package informer

import (
	// "fmt"

	"context"
	"fmt"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// "k8s.io/cli-runtime/pkg/genericclioptions"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

//generate fakeclient to get configmap
//
func TestGetConfigMapByName(t *testing.T) {
	//Test scenarios:
	// Correct ConfigMap - clientset1
	// No configmap. -
	// Configmap without AllowResources or DenyResources keys.- clientset2
	// Configmap with invalid yaml in AllowResources or DenyResources. clientset3
	// Configmap with invalid Name - clientset
	configmaps := []struct {
		clientset kubernetes.Interface
		n         string
		ns        string
		err       error
	}{

		{ // ConfigMap - Correct format for Allowed/Denied Resources and has correct Name
			clientset: fake.NewSimpleClientset(&v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "search-collector-config",
					Namespace: "open-cluster-management",
				},
				Data: map[string]string{
					"AllowedResources": "- apiGroups: \n    - \"*\"\n  resources: \n    - services\n    - pods\n- apiGroups:\n    - admission.k8s.io\n    - authentication.k8s.io\n  resources:\n    - \"*\"",
					"DeniedResources":  "- apiGroups:\n    - \"*\"\n  resources:\n    - secrets\n- apiGroups:\n    - admission.k8s.io\n  resources:\n    - policies\n    - iampolicies\n    - certificatepolicies",
				},
			}), ns: "open-cluster-management",
			n:   "search-collector-config",
			err: nil},

		{ // ConfigMap AllowedResources is missing key (first apiGroups)
			clientset: fake.NewSimpleClientset(&v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "search-collector-config",
					Namespace: "open-cluster-management",
				},
				Data: map[string]string{
					"AllowedResources": "- resources: \n    - services\n    - pods\n- apiGroups:\n    - admission.k8s.io\n    - authentication.k8s.io\n  resources:\n    - \"*\"",
					"DeniedResources":  "- apiGroups:\n    - \"*\"\n  resources:\n    - secrets\n- apiGroups:\n    - admission.k8s.io\n  resources:\n    - policies\n    - iampolicies\n    - certificatepolicies",
				},
			}),
			ns:  "open-cluster-management",
			n:   "search-collector-config",
			err: fmt.Errorf("This configMap is missing a key")},

		// ConfigMap has incorrect structure for the DeniedResources and AllowedResources (each resources does not start in new line)
		{clientset: fake.NewSimpleClientset(&v1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "search-collector-config",
				Namespace: "open-cluster-management",
			},
			Data: map[string]string{
				"AllowedResources": "apiGroups: [\"*\"]\n  resources: [services, pods]\napiGroups:[admission.k8s.io,authentication.k8s.io]\n  resources:[\"*\"]",
				"DeniedResources":  "apiGroups: [\"*\"]\n  resources: [secrets]\napiGroups:[admission.k8s.io]\n  resources:[policies, iampolicies, certificatepolicies]",
			},
		}),
			ns:  "open-cluster-management",
			n:   "search-collector-config",
			err: fmt.Errorf("This configMap has an invalid AllowedResources/DeniedResources format")},

		// ConfigMap has incorrect name
		{clientset: fake.NewSimpleClientset(&v1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wrong-name",
				Namespace: "open-cluster-management",
			},
			Data: map[string]string{
				"AllowedResources": "- apiGroups: \n    - \"*\"\n  resources: \n    - services\n    - pods\n- apiGroups:\n    - admission.k8s.io\n    - authentication.k8s.io\n  resources:\n    - \"*\"",
				"DeniedResources":  "- apiGroups:\n    - \"*\"\n  resources:\n    - secrets\n- apiGroups:\n    - admission.k8s.io\n  resources:\n    - policies\n    - iampolicies\n    - certificatepolicies",
			},
		}),
			ns:  "open-cluster-management",
			n:   "search-collector-config",
			err: fmt.Errorf("This configMap is missing a key")},
	}

	for _, clientset := range configmaps {

		config, err := clientset.clientset.CoreV1().ConfigMaps(clientset.ns).Get(context.TODO(), clientset.n, metav1.GetOptions{})
		if err != nil {
			if clientset.err == nil {
				t.Fatalf(err.Error())
			}
			if !strings.EqualFold(clientset.err.Error(), err.Error()) {
				t.Fatalf("expected err: %s got err: %s", clientset.err, err)
			}
		} else {
			if _, ok := config.Data["AllowedResources"]; !ok {
				t.Errorf("AllowedResources not there or malformed")
			}
			if _, ok := config.Data["DeniedResources"]; !ok {
				t.Errorf("DeniedResources not there or malformed")
			}
		}

	}

	return

}
