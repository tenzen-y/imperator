/*
Copyright 2021 Yuki Iwai (@tenzen-y)

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

	// nvidia.com/gpu.family
	// +optional
	Family string `json:"family,omitempty"`

	// nvidia.com/gpu.product
	// +optional
	Product string `json:"product,omitempty"`

	// nvidia.com/gpu.machine
	// +optional
	Machine string `json:"machine,omitempty"`
}

// MachineStatus defines the observed state of Machine
type MachineStatus struct {

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +kubebuilder:validation:Required
	AvailableMachines []AvailableMachineCondition `json:"availableMachines,omitempty"`
}

type AvailableMachineCondition struct {

	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`

	// +kubebuilder:validation:Required
	Usage UsageCondition `json:"usage,omitempty"`
}

type UsageCondition struct {

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum:=0
	Maximum int32 `json:"maximum"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum:=0
	Reserved int32 `json:"reserved"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum:=0
	Used int32 `json:"used"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum:=0
	Waiting int32 `json:"waiting"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Group",type="string",JSONPath=`.metadata.labels['imperator\.tenzen-y\.io/machine-group']`
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=`.status.conditions[?(@.type=='Ready')].status`

// Machine is the Schema for the machines API
type Machine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineSpec   `json:"spec,omitempty"`
	Status MachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MachineList contains a list of Machine
type MachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Machine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Machine{}, &MachineList{})
}
