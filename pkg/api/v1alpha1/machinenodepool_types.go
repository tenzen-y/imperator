package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MachineNodePoolSpec defines the desired state of MachineNodePool
type MachineNodePoolSpec struct {

	// MachineGroupName is node pool group
	// +kubebuilder:validation:Required
	MachineGroupName string `json:"machineGroupName"`

	// NodePool is node list that machineGroup is managing.
	// +kubebuilder:validation:Required
	NodePool []NodePool `json:"nodePool"`

	// MachineTypeStock is available machineType list.
	// +kubebuilder:validation:Required
	MachineTypeStock []NodePoolMachineTypeStock `json:"machineTypeStock"`
}

type NodePool struct {

	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=ready;maintenance
	Mode NodePoolMode `json:"mode"`

	// +optional
	// default=false
	Taint *bool `json:"taint,omitempty"`

	// +kubebuilder:validation:Required
	MachineType []NodePoolMachineType `json:"machineType"`
}

type NodePoolMachineType struct {

	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

type NodePoolMode string

const (
	NodeModeReady       NodePoolMode = "ready"
	NodeModeNotReady    NodePoolMode = "not-ready"
	NodeModeMaintenance NodePoolMode = "maintenance"
)

func (mode NodePoolMode) Value() string {
	return string(mode)
}

type NodePoolMachineTypeStock struct {

	// +kubebuilder:validation:Required
	Name string `json:"name"`
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
	NodeHealthy     MachineNodeCondition = "Healthy"
	NodeMaintenance MachineNodeCondition = "Maintenance"
	NodeUnhealthy   MachineNodeCondition = "Unhealthy"
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

// +kubebuilder:object:root=true

// MachineNodePoolList contains a list of MachineNodePool
type MachineNodePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MachineNodePool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MachineNodePool{}, &MachineNodePoolList{})
}
