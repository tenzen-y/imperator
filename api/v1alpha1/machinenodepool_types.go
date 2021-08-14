package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MachineNodePoolSpec defines the desired state of MachineNodePool
type MachineNodePoolSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of MachineNodePool. Edit machinenodepool_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// MachineNodePoolStatus defines the observed state of MachineNodePool
type MachineNodePoolStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// MachineNodePool is the Schema for the machinenodepools API
type MachineNodePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineNodePoolSpec   `json:"spec,omitempty"`
	Status MachineNodePoolStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MachineNodePoolList contains a list of MachineNodePool
type MachineNodePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MachineNodePool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MachineNodePool{}, &MachineNodePoolList{})
}
