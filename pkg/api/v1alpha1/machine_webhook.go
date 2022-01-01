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
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/tenzen-y/imperator/pkg/consts"
)

// log is for logging in this package.
var (
	machinelog = logf.Log.WithName("machine-resource")
	kubeReader client.Reader
	ctx        context.Context
)

func (r *Machine) SetupWebhookWithManager(signalHandler context.Context, mgr ctrl.Manager) error {
	kubeReader = mgr.GetAPIReader()
	ctx = signalHandler
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-imperator-tenzen-y-io-v1alpha1-machine,mutating=true,failurePolicy=fail,sideEffects=None,groups=imperator.tenzen-y.io,resources=machines,verbs=create;update,versions=v1alpha1,name=defaulter.machine.imperator.tenzen-y.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Machine{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Machine) Default() {
	machinelog.Info("default", "name", r.Name)

	for _, pool := range r.Spec.NodePool {
		// Taint is false by default
		if pool.Taint == nil {
			pool.Taint = pointer.Bool(false)
		}
	}

	// initialize machineAvailable
	for _, mt := range r.Spec.MachineTypes {
		r.Status.AvailableMachines = append(r.Status.AvailableMachines, AvailableMachineCondition{
			Name: mt.Name,
			Usage: UsageCondition{
				Maximum:  mt.Available,
				Reserved: 0,
				Used:     0,
				Waiting:  0,
			},
		})
	}

}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:path=/validate-imperator-tenzen-y-io-v1alpha1-machine,mutating=false,failurePolicy=fail,sideEffects=None,groups=imperator.tenzen-y.io,resources=machines,verbs=create;update,versions=v1alpha1,name=validator.machine.imperator.tenzen-y.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Machine{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Machine) ValidateCreate() error {
	machinelog.Info("validate create", "name", r.Name)
	if err := r.ValidateAllOperation(); err != nil {
		return err
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Machine) ValidateUpdate(old runtime.Object) error {
	machinelog.Info("validate update", "name", r.Name)
	if err := r.ValidateAllOperation(); err != nil {
		return err
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Machine) ValidateDelete() error {
	machinelog.Info("validate delete", "name", r.Name)
	return nil
}

func (r *Machine) ValidateAllOperation() error {
	if err := r.ValidateLabel(); err != nil {
		return err
	}
	if err := r.ValidateNodeName(); err != nil {
		return err
	}
	if err := r.ValidateNodePoolMachineTypeName(); err != nil {
		return err
	}
	if err := r.ValidateGPUSpec(); err != nil {
		return err
	}
	return nil
}

func (r *Machine) ValidateLabel() error {
	machineGroupName, exist := r.Labels[consts.MachineGroupKey]
	if !exist {
		return fmt.Errorf("%s is must be set in .metadata.labels", consts.MachineGroupKey)
	}

	machines := &MachineList{}
	if err := kubeReader.List(ctx, machines, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			consts.MachineGroupKey: machineGroupName,
		}),
	}); err != nil {
		return err
	}

	if len(machines.Items) == 0 {
		return nil
	}

	for _, m := range machines.Items {
		if m.Name != r.Name {
			return fmt.Errorf("machineGroup label is duplicate, machineGroup must be set as unique")
		}
	}

	return nil
}

func (r *Machine) ValidateNodeName() error {
	nodes := &corev1.NodeList{}
	if err := kubeReader.List(ctx, nodes, &client.ListOptions{}); err != nil {
		return err
	}

	existNodes := map[string]bool{}
	for _, n := range nodes.Items {
		existNodes[n.Name] = true
	}
	for _, np := range r.Spec.NodePool {
		if !existNodes[np.Name] {
			return fmt.Errorf("failed to find node %s in Kubernetes Cluster", np.Name)
		}
	}

	return nil
}

func (r *Machine) ValidateNodePoolMachineTypeName() error {

	machineTypeMap := map[string]MachineType{}
	for _, m := range r.Spec.MachineTypes {
		if _, exist := machineTypeMap[m.Name]; exist {
			return fmt.Errorf("machineType name <%s> is duplicated", m.Name)
		}
		machineTypeMap[m.Name] = m
	}

	for _, p := range r.Spec.NodePool {
		if len(p.MachineType) != 1 {
			// Support only one machineType in first release
			return fmt.Errorf("<%s>; can not set multiple machineType in nodePool.machineType", p.Name)
		}
		for _, mt := range p.MachineType {
			if _, exist := machineTypeMap[mt.Name]; !exist {
				return fmt.Errorf("%s was not found in spec.machineTypes", mt.Name)
			}
		}
	}

	return nil
}

func (r *Machine) ValidateGPUSpec() error {
	for _, m := range r.Spec.MachineTypes {
		if m.Spec.GPU == nil {
			continue
		}
		if m.Spec.GPU.Type == "" {
			return fmt.Errorf("gpu.type must be set value")
		}
		if m.Spec.GPU.Num.Value() < 0 {
			return fmt.Errorf("gpu.num must be set 0 or more value")
		}
		var gpuSelectorTypes []string
		for _, s := range []string{m.Spec.GPU.Family, m.Spec.GPU.Product, m.Spec.GPU.Machine} {
			if s == "" {
				continue
			}
			gpuSelectorTypes = append(gpuSelectorTypes, s)
		}

		if gpuSelectorTypes == nil {
			return fmt.Errorf("you must set a value for either gpu.family, gpu.product or gpu.machine")
		} else if len(gpuSelectorTypes) > 1 {
			return fmt.Errorf("only one GPU family, product or machine cane be set")
		}
	}
	return nil
}
