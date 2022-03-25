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
func Test_GetConfigMapByName(t *testing.T) {
	//Test scenarios:
	// Correct ConfigMap - clientset1
	// No configmap. -
	// Configmap without AllowResources or DenyResources keys.- clientset2
	// Configmap with invalid yaml in AllowResources or DenyResources. clientset3
	// Configmap with invalid Name - clientset
	configmaps := []struct {
		clientset kubernetes.Interface
		name      string
		namespace string
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
			}), namespace: "open-cluster-management",
			name: "search-collector-config",
			err:  nil},

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
			namespace: "open-cluster-management",
			name:      "search-collector-config",
			err:       fmt.Errorf("yaml: line 1: did not find expected key")},

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
				"DeniedResources":  "- apiGroups:\n    - \"*\"\n  resources:\n    - secrets\n- apiGroups:\n    - admission.k8s.io\n  resources:\n    - policies\n    - iampolicies\n    - certificatepolicies",
			},
		}),
			namespace: "open-cluster-management",
			name:      "search-collector-config",
			err:       fmt.Errorf("yaml: line 1: did not find expected key")},

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
			namespace: "open-cluster-management",
			name:      "wrong-name",
			err:       fmt.Errorf("configmaps \"search-collector-config\" not found")},
	}
	// errs := make([]error, 2)

	for _, clientset := range configmaps {

		config, err := clientset.clientset.CoreV1().ConfigMaps(clientset.namespace).Get(context.TODO(), clientset.name, metav1.GetOptions{})
		if err != nil {
			if clientset.err == nil { //if config has err but the clientset does not
				t.Fatalf(err.Error())
			}
			if !strings.EqualFold(clientset.err.Error(), err.Error()) { //if the errors don't match
				t.Fatalf("expected err was %s but got err like %s", clientset.err, err)
			}
			// } else {
			// 	glog.Info("Didn't find ConfigMap with name search-collector-config. Will collect all resources. ", err) //there is an error with config

		}

		_, _, allowerr, denyerr := GetAllowDenyData(config)

		if allowerr != nil {
			if clientset.err == nil {
				t.Fatalf(allowerr.Error())
			}
			if !strings.EqualFold(clientset.err.Error(), allowerr.Error()) {
				t.Fatalf("expected err was %s but got err like %s", clientset.err, allowerr)
			}
		}
		if denyerr != nil {
			if clientset.err == nil {
				t.Fatalf(denyerr.Error())
			}
			if !strings.EqualFold(clientset.err.Error(), denyerr.Error()) {
				t.Fatalf("expected err was %s but got err like %s", clientset.err, denyerr)
			}
		}
		// errs = append(errs, allowerr, denyerr)

		// for _, err := range errs {
		// if err != nil {
		// 	if clientset.err == nil {
		// 		t.Fatalf(err.Error())
		// 	}
		// 	if !strings.EqualFold(clientset.err.Error(), err.Error()) {
		// 		t.Fatalf("expected err was %s but got err like %s", clientset.err, err)
		// 	}
		// }
		// }

	}
}
