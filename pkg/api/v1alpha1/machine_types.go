package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MachineSpec defines the desired state of Machine
type MachineSpec struct {

	// NodePool is node list that machineGroup is managing.
	// +kubebuilder:validation:Required
	NodePool []NodePool `json:"nodePool"`

	// +kubebuilder:validation:Required
	MachineTypes []MachineType `json:"machineTypes"`
}

type MachineType struct {

	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// +kubebuilder:validation:Required
	Spec MachineDetailSpec `json:"spec"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum:=0
	Available int32 `json:"available"`

	// +optional
	Dependence *Dependence `json:"dependence,omitempty"`
}

type Dependence struct {

	// +optional
	Parent string `json:"parent,omitempty"`

	// +optional
	AvailableRatio string `json:"availableRatio,omitempty"`
}

type MachineDetailSpec struct {

	// +kubebuilder:validation:Required
	CPU resource.Quantity `json:"cpu"`

	// +kubebuilder:validation:Required
	Memory resource.Quantity `json:"memory"`

	// +optional
	GPU *GPUSpec `json:"gpu,omitempty"`
}

type GPUSpec struct {

	// +optional
	Type corev1.ResourceName `json:"type,omitempty"`

	// +optional
	Num resource.Quantity `json:"num,omitempty"`

	// +optional
	Generation string `json:"generation,omitempty"`
}

// MachineStatus defines the observed state of Machine
type MachineStatus struct {

	// +optional
	Conditions []metav1.Condition `json:"condition,omitempty"`

	// +optional
	AvailableMachines []AvailableMachineCondition `json:"availableMachines,omitempty"`
}

type AvailableMachineCondition struct {

	// +optional
	Name string `json:"name,omitempty"`

	// +optional
	Usage UsageCondition `json:"usage,omitempty"`
}

type UsageCondition struct {

	// +optional
	// +kubebuilder:validation:Minimum:=0
	Maximum int32 `json:"maximum,omitempty"`

	// +optional
	// +kubebuilder:validation:Minimum:=0
	Ready int32 `json:"ready,omitempty"`

	// +optional
	// +kubebuilder:validation:Minimum:=0
	Used int32 `json:"used,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// Machine is the Schema for the machines API
type Machine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineSpec   `json:"spec,omitempty"`
	Status MachineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MachineList contains a list of Machine
type MachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Machine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Machine{}, &MachineList{})
}