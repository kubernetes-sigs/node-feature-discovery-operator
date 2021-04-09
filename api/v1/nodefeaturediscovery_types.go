/*
Copyright 2021. The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeFeatureDiscoverySpec defines the desired state of NodeFeatureDiscovery
// +k8s:openapi-gen=true
type NodeFeatureDiscoverySpec struct {
	Operand      OperandSpec `json:"operand"`
	WorkerConfig ConfigMap   `json:"workerConfig"`
	CustomConfig ConfigMap   `json:"customConfig"`
}

// OperandSpec describes configuration options for the operand
type OperandSpec struct {
	// +kubebuilder:validation:Pattern=[a-zA-Z0-9\.\-\/]+
	Namespace string `json:"namespace,omitempty"`

	// +kubebuilder:validation:Pattern=[a-zA-Z0-9\-]+
	Image string `json:"image,omitempty"`

	// Image pull policy
	// +kubebuilder:validation:Optional
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`
}

// ConfigMap describes configuration options for the NFD worker
type ConfigMap struct {
	// BinaryData holds the NFD configuration file
	ConfigData string `json:"configData"`
}

// NodeFeatureDiscoveryStatus defines the observed state of NodeFeatureDiscovery
// +k8s:openapi-gen=true
type NodeFeatureDiscoveryStatus struct {
	// Conditions represents the latest available observations of current state.
	// +optional
	Conditions []conditionsv1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=nodefeaturediscoveries,scope=Namespaced

// NodeFeatureDiscovery is the Schema for the nodefeaturediscoveries API
type NodeFeatureDiscovery struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeFeatureDiscoverySpec   `json:"spec,omitempty"`
	Status NodeFeatureDiscoveryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeFeatureDiscoveryList contains a list of NodeFeatureDiscovery
type NodeFeatureDiscoveryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeFeatureDiscovery `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeFeatureDiscovery{}, &NodeFeatureDiscoveryList{})
}

// ImagePath returns a compiled full valid image string
func (o *OperandSpec) ImagePath() string {
	return o.Image
}

// ImagePolicy returns a valid corev1.PullPolicy from the string in the CR
func (o *OperandSpec) ImagePolicy(pullPolicy string) corev1.PullPolicy {
	switch corev1.PullPolicy(pullPolicy) {
	case corev1.PullAlways:
		return corev1.PullAlways
	case corev1.PullNever:
		return corev1.PullNever
	}
	return corev1.PullIfNotPresent
}

// Data returns a valid ConfigMap name
func (c *ConfigMap) Data() string {
	return c.ConfigData
}
