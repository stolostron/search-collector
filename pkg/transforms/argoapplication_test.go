/*
IBM Confidential
OCO Source Materials
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets,
irrespective of what has been deposited with the U.S. Copyright Office.
Copyright (c) 2020 Red Hat, Inc.
*/
// Copyright Contributors to the Open Cluster Management project

package transforms

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestTransformArgoApplication(t *testing.T) {
	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Application",
			"metadata": map[string]interface{}{
				"creationTimestamp": "2021-02-10T02:15:57Z",
				"name":              "helloworld",
				"namespace":         "argocd",
			},
			"spec": map[string]interface{}{
				"destination": map[string]interface{}{
					"name":      "local-cluster",
					"namespace": "argo-helloworld",
					"server":    "https://kubernetes.default.svc",
				},
				"project": "default",
				"source": map[string]interface{}{
					"path":           "helloworld",
					"chart":          "hello-chart",
					"repoURL":        "https://github.com/fxiang1/app-samples",
					"targetRevision": "HEAD",
				},
				"syncPolicy": map[string]interface{}{
					"automated": map[string]interface{}{
						"selfHeal": true,
					},
				},
			},
		},
	}
	generic := GenericResourceBuilder(u)
	node := ArgoApplicationResourceBuilder(generic.BuildNode(), u.UnstructuredContent()).BuildNode()

	// Test only the fields that exist in argo application - the common test will test the other bits
	AssertEqual("kind", node.Properties["kind"], "Application", t)
	AssertEqual("destinationName", node.Properties["destinationName"], "local-cluster", t)
	AssertEqual("destinationNamespace", node.Properties["destinationNamespace"], "argo-helloworld", t)
	AssertEqual("destinationServer", node.Properties["destinationServer"], "https://kubernetes.default.svc", t)
	AssertEqual("path", node.Properties["path"], "helloworld", t)
	AssertEqual("chart", node.Properties["chart"], "hello-chart", t)
	AssertEqual("repoURL", node.Properties["repoURL"], "https://github.com/fxiang1/app-samples", t)
	AssertEqual("targetRevision", node.Properties["targetRevision"], "HEAD", t)
}
