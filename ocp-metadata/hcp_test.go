// Copyright 2026 The go-commons Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ocpmetadata

import (
	"encoding/json"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestGetClusterMetadataHCP(t *testing.T) {
	meta := newHCPTestMetadata(t)

	clusterMetadata, err := meta.GetClusterMetadata()
	if err != nil {
		t.Fatalf("GetClusterMetadata returned unexpected error: %v", err)
	}
	if clusterMetadata.ClusterType != "rosa-hcp" {
		t.Fatalf("ClusterType = %q, want %q", clusterMetadata.ClusterType, "rosa-hcp")
	}
	if clusterMetadata.MasterNodesCount != 0 {
		t.Fatalf("MasterNodesCount = %d, want %d", clusterMetadata.MasterNodesCount, 0)
	}
	if clusterMetadata.MasterNodesType != "" {
		t.Fatalf("MasterNodesType = %q, want %q", clusterMetadata.MasterNodesType, "")
	}
	if clusterMetadata.WorkerNodesCount != 2 {
		t.Fatalf("WorkerNodesCount = %d, want %d", clusterMetadata.WorkerNodesCount, 2)
	}
	if clusterMetadata.TotalNodes != 2 {
		t.Fatalf("TotalNodes = %d, want %d", clusterMetadata.TotalNodes, 2)
	}

	payload, err := json.Marshal(clusterMetadata)
	if err != nil {
		t.Fatalf("Marshal returned unexpected error: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatalf("Unmarshal returned unexpected error: %v", err)
	}
	if _, ok := raw["masterNodesCount"]; !ok {
		t.Fatal("masterNodesCount missing from JSON output, want explicit 0")
	}
	if _, ok := raw["masterNodesType"]; !ok {
		t.Fatal("masterNodesType missing from JSON output, want explicit empty string")
	}
}

func newHCPTestMetadata(t *testing.T) Metadata {
	t.Helper()

	clientSet := k8sfake.NewClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "worker-0",
				Labels: map[string]string{
					"node-role.kubernetes.io/worker":   "",
					"node.kubernetes.io/instance-type": "m6i.xlarge",
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "worker-1",
				Labels: map[string]string{
					"node-role.kubernetes.io/worker":   "",
					"node.kubernetes.io/instance-type": "m6i.xlarge",
				},
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kube-system",
				Name:      "cluster-config-v1",
			},
			Data: map[string]string{
				"install-config": `
publish: External
fips: false
compute:
- name: worker
  architecture: amd64
controlPlane:
  architecture: amd64
`,
			},
		},
	)
	discovery, ok := clientSet.Discovery().(*fake.FakeDiscovery)
	if !ok {
		t.Fatal("unexpected discovery type")
	}
	discovery.FakedServerVersion = &version.Info{GitVersion: "v1.31.6"}
	for _, group := range []string{APIGroupOpenShiftConfig, APIGroupOpenShiftRoute} {
		discovery.Resources = append(discovery.Resources, &metav1.APIResourceList{
			GroupVersion: group + "/v1",
		})
	}

	return Metadata{
		connector: fakeConnector{
			clientSet: clientSet,
			dynamicClient: dynamicfake.NewSimpleDynamicClient(
				runtime.NewScheme(),
				newClusterVersion("4.18.9"),
				newHCPInfrastructure(),
				newConfigNetwork(),
				newOperatorNetwork(),
			),
			restConfig: &rest.Config{},
		},
	}
}

func newHCPInfrastructure() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "config.openshift.io/v1",
			"kind":       "Infrastructure",
			"metadata": map[string]interface{}{
				"name": "cluster",
			},
			"status": map[string]interface{}{
				"infrastructureName":   "hcp-test-abcde",
				"platform":             "AWS",
				"controlPlaneTopology": "External",
				"platformStatus": map[string]interface{}{
					"aws": map[string]interface{}{
						"region": "us-east-1",
						"resourceTags": []interface{}{
							map[string]interface{}{
								"key":   "red-hat-clustertype",
								"value": "rosa",
							},
						},
					},
					"type": "AWS",
				},
			},
		},
	}
}
