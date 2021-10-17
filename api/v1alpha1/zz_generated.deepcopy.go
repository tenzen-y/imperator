//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2021.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AvailableMachineCondition) DeepCopyInto(out *AvailableMachineCondition) {
	*out = *in
	out.Usage = in.Usage
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AvailableMachineCondition.
func (in *AvailableMachineCondition) DeepCopy() *AvailableMachineCondition {
	if in == nil {
		return nil
	}
	out := new(AvailableMachineCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Dependence) DeepCopyInto(out *Dependence) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Dependence.
func (in *Dependence) DeepCopy() *Dependence {
	if in == nil {
		return nil
	}
	out := new(Dependence)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GPUSpec) DeepCopyInto(out *GPUSpec) {
	*out = *in
	out.Num = in.Num.DeepCopy()
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GPUSpec.
func (in *GPUSpec) DeepCopy() *GPUSpec {
	if in == nil {
		return nil
	}
	out := new(GPUSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Machine) DeepCopyInto(out *Machine) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Machine.
func (in *Machine) DeepCopy() *Machine {
	if in == nil {
		return nil
	}
	out := new(Machine)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Machine) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineDetailSpec) DeepCopyInto(out *MachineDetailSpec) {
	*out = *in
	out.CPU = in.CPU.DeepCopy()
	out.Memory = in.Memory.DeepCopy()
	if in.GPU != nil {
		in, out := &in.GPU, &out.GPU
		*out = new(GPUSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Dependence != nil {
		in, out := &in.Dependence, &out.Dependence
		*out = new(Dependence)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineDetailSpec.
func (in *MachineDetailSpec) DeepCopy() *MachineDetailSpec {
	if in == nil {
		return nil
	}
	out := new(MachineDetailSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineList) DeepCopyInto(out *MachineList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Machine, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineList.
func (in *MachineList) DeepCopy() *MachineList {
	if in == nil {
		return nil
	}
	out := new(MachineList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MachineList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineNodePool) DeepCopyInto(out *MachineNodePool) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineNodePool.
func (in *MachineNodePool) DeepCopy() *MachineNodePool {
	if in == nil {
		return nil
	}
	out := new(MachineNodePool)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MachineNodePool) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineNodePoolList) DeepCopyInto(out *MachineNodePoolList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]MachineNodePool, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineNodePoolList.
func (in *MachineNodePoolList) DeepCopy() *MachineNodePoolList {
	if in == nil {
		return nil
	}
	out := new(MachineNodePoolList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MachineNodePoolList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineNodePoolSpec) DeepCopyInto(out *MachineNodePoolSpec) {
	*out = *in
	if in.NodePool != nil {
		in, out := &in.NodePool, &out.NodePool
		*out = make([]NodePool, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineNodePoolSpec.
func (in *MachineNodePoolSpec) DeepCopy() *MachineNodePoolSpec {
	if in == nil {
		return nil
	}
	out := new(MachineNodePoolSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineNodePoolStatus) DeepCopyInto(out *MachineNodePoolStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.NodePoolCondition != nil {
		in, out := &in.NodePoolCondition, &out.NodePoolCondition
		*out = make([]NodePoolCondition, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineNodePoolStatus.
func (in *MachineNodePoolStatus) DeepCopy() *MachineNodePoolStatus {
	if in == nil {
		return nil
	}
	out := new(MachineNodePoolStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineSpec) DeepCopyInto(out *MachineSpec) {
	*out = *in
	if in.NodePool != nil {
		in, out := &in.NodePool, &out.NodePool
		*out = make([]NodePool, len(*in))
		copy(*out, *in)
	}
	if in.MachineTypes != nil {
		in, out := &in.MachineTypes, &out.MachineTypes
		*out = make([]MachineType, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineSpec.
func (in *MachineSpec) DeepCopy() *MachineSpec {
	if in == nil {
		return nil
	}
	out := new(MachineSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineStatus) DeepCopyInto(out *MachineStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.AvailableMachines = in.AvailableMachines
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineStatus.
func (in *MachineStatus) DeepCopy() *MachineStatus {
	if in == nil {
		return nil
	}
	out := new(MachineStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineType) DeepCopyInto(out *MachineType) {
	*out = *in
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineType.
func (in *MachineType) DeepCopy() *MachineType {
	if in == nil {
		return nil
	}
	out := new(MachineType)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodePool) DeepCopyInto(out *NodePool) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodePool.
func (in *NodePool) DeepCopy() *NodePool {
	if in == nil {
		return nil
	}
	out := new(NodePool)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodePoolCondition) DeepCopyInto(out *NodePoolCondition) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodePoolCondition.
func (in *NodePoolCondition) DeepCopy() *NodePoolCondition {
	if in == nil {
		return nil
	}
	out := new(NodePoolCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UsageCondition) DeepCopyInto(out *UsageCondition) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UsageCondition.
func (in *UsageCondition) DeepCopy() *UsageCondition {
	if in == nil {
		return nil
	}
	out := new(UsageCondition)
	in.DeepCopyInto(out)
	return out
}
