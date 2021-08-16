package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MachineNodePoolSpec defines the desired state of MachineNodePool
type MachineNodePoolSpec struct {

	// MachineGroupName is node pool group
	//+kubebuilder:validation:Required
	MachineGroupName string `json:"machineGroupName"`

	// NodePool is node list that machineGroup is managing.
	// +kubebuilder:validation:Required
	NodePool []NodePool `json:"nodePool"`
}

type NodePool struct {

	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=ready;maintenance
	Mode string `json:"mode"`
}

// MachineNodePoolStatus defines the observed state of MachineNodePool
type MachineNodePoolStatus struct {

	// +optional
	Conditions []metav1.Condition `json:"condition,omitempty"`

	// +optional
	NodePoolCondition []NodePoolCondition `json:"nodePool,omitempty"`
}

type NodePoolCondition struct {

	// +optional
	Name string `json:"name,omitempty"`

	// +optional
	NodeCondition MachineNodeCondition `json:"condition,omitempty"`
}

// MachineNodeCondition is condition of Kubernetes Nodes
// +kubebuilder:validation:Enum=Healthy;Maintenance;Unhealthy
type MachineNodeCondition string

const (
	NodeReady       MachineNodeCondition = "Ready"
	NodeMaintenance MachineNodeCondition = "Maintenance"
	NodeNotReady    MachineNodeCondition = "NotReady"
)

const (
	ConditionReady = "Ready"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Group",type="string",JSONPath=".spec.machineGroup"

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
