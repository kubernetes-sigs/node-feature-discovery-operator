package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NodeFeatureDiscoverySpec defines the desired state of NodeFeatureDiscovery
// +k8s:openapi-gen=true
type NodeFeatureDiscoverySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// NodeFeatureDiscoveryStatus defines the observed state of NodeFeatureDiscovery
// +k8s:openapi-gen=true
type NodeFeatureDiscoveryStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeFeatureDiscovery is the Schema for the nodefeaturediscoveries API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type NodeFeatureDiscovery struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeFeatureDiscoverySpec   `json:"spec,omitempty"`
	Status NodeFeatureDiscoveryStatus `json:"status,omitempty"`
}

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
