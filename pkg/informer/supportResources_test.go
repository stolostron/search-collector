package informer

import (
	"context"
	"fmt"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

//generate fakeclient to get configmap
func Test_GetConfigMapByName(t *testing.T) {

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

	for _, clientset := range configmaps {

		config, err := clientset.clientset.CoreV1().ConfigMaps(clientset.namespace).Get(context.TODO(), clientset.name, metav1.GetOptions{})
		if err != nil {
			if clientset.err == nil { //if config has err but the clientset does not
				t.Fatalf("Received unexpected error from client. Error: %s", err.Error())
			}
			if !strings.EqualFold(clientset.err.Error(), err.Error()) { //if the errors don't match
				t.Fatalf("expected err was %s but got err like %s", clientset.err, err)
			}

		}

		_, _, allowerr, denyerr := GetAllowDenyData(config)

		if allowerr != nil {
			if clientset.err == nil {
				t.Fatalf("Received unexpected error from client. Error: %s", err.Error())
			}
			if !strings.EqualFold(clientset.err.Error(), allowerr.Error()) {
				t.Fatalf("expected err was %s but got err like %s", clientset.err, allowerr)
			}
		}
		if denyerr != nil {
			if clientset.err == nil {
				t.Fatalf("Received unexpected error from client. Error: %s", err.Error())
			}
			if !strings.EqualFold(clientset.err.Error(), denyerr.Error()) {
				t.Fatalf("expected err was %s but got err like %s", clientset.err, denyerr)
			}
		}

	}

}

func Test_supportedResources(t *testing.T) {

	clientset := fake.NewSimpleClientset(&v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "search-collector-config",
			Namespace: "open-cluster-management",
		},
		Data: map[string]string{
			"AllowedResources": "- apiGroups: \n    - \"*\"\n  resources: \n    - services\n    - pods\n- apiGroups:\n    - authentication.k8s.io\n  resources:\n    - \"*\"",
			"DeniedResources":  "- apiGroups:\n    - \"*\"\n  resources:\n    - secrets\n- apiGroups:\n    - admission.k8s.io\n  resources:\n    - policies\n    - iampolicies\n    - certificatepolicies\n- apiGroups:\n    - admission.k8s.io\n  resources:\n    - \"*\"",
		},
	})

	config, _ := clientset.CoreV1().ConfigMaps("open-cluster-management").Get(context.TODO(), "search-collector-config", metav1.GetOptions{})
	allow, deny, _, _ := GetAllowDenyData(config)

	allowData := [][]string{{"authentication.k8s.io", "deployments"}, {"services.anygroup.io", "services"}} // Test all resources allowed for group // Test resource matching any group
	denyData := [][]string{{"admission.k8s.io", "iampolicies"}, {"secrets.anygroup.io", "secrets"}}         // Test all resources allowed for group // Test resource matching any group

	for _, data := range allowData {
		allowed := isResourceAllowed(data[0], data[1], allow, deny)
		if !allowed {
			t.Errorf("Expected group: %s resource: %s to be allowed, but got denied.", data[0], data[1])
		}
	}
	for _, data := range denyData {
		denied := isResourceAllowed(data[0], data[1], allow, deny)
		if denied {
			t.Errorf("Expected group: %s resource: %s to be denied, but got allowed.", data[0], data[1])
		}

	}

}
