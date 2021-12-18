package v1alpha1

import (
	"context"
	"fmt"
	"github.com/tenzen-y/imperator/pkg/consts"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var (
	machinelog = logf.Log.WithName("machine-resource")
	kubeReader client.Reader
)

func (r *Machine) SetupWebhookWithManager(mgr ctrl.Manager) error {
	kubeReader = mgr.GetAPIReader()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-imperator-tenzen-y-io-v1alpha1-machine,mutating=true,failurePolicy=fail,sideEffects=None,groups=imperator.tenzen-y.io,resources=machines,verbs=create;update,versions=v1alpha1,name=defaulter.machine.imperator.tenzen-y.io,admissionReviewVersions={v1,v1beta1}

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

}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-imperator-tenzen-y-io-v1alpha1-machine,mutating=false,failurePolicy=fail,sideEffects=None,groups=imperator.tenzen-y.io,resources=machines,verbs=create;update,versions=v1alpha1,name=validator.machine.imperator.tenzen-y.io,admissionReviewVersions={v1,v1beta1}

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
	if err := kubeReader.List(context.Background(), machines, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			consts.MachineGroupKey: machineGroupName,
		}),
	}); err != nil {
		return err
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
	if err := kubeReader.List(context.Background(), nodes, &client.ListOptions{}); err != nil {
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
		if m.Spec.GPU.Family == "" {
			return fmt.Errorf("gpu.generation must be set value")
		}
	}
	return nil
}
