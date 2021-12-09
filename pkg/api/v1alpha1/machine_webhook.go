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
	"strconv"
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

//+kubebuilder:webhook:path=/mutate-imperator-tenzen-y-io-v1alpha1-machine,mutating=true,failurePolicy=fail,sideEffects=None,groups=imperator.tenzen-y.io,resources=machines,verbs=create;update,versions=v1alpha1,name=mmachine.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Machine{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Machine) Default() {
	machinelog.Info("default", "name", r.Name)

	for _, pool := range r.Spec.NodePool {
		// input default assignmentType
		if pool.Taint == nil {
			pool.Taint = pointer.Bool(false)
		}
		// input default scheduleChildren
		if pool.MachineType.ScheduleChildren == nil {
			pool.MachineType.ScheduleChildren = pointer.Bool(false)
		}
	}

}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-imperator-tenzen-y-io-v1alpha1-machine,mutating=false,failurePolicy=fail,sideEffects=None,groups=imperator.tenzen-y.io,resources=machines,verbs=create;update,versions=v1alpha1,name=vmachine.kb.io,admissionReviewVersions={v1,v1beta1}

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
	if err := r.ValidateDependence(); err != nil {
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
		machineTypeMap[m.Name] = m
	}

	for _, p := range r.Spec.NodePool {
		if _, exist := machineTypeMap[p.MachineType.Name]; !exist {
			return fmt.Errorf("%s was not found in spec.machineTypes", p.MachineType.Name)
		}

		if *p.MachineType.ScheduleChildren {
			childrenNum := 0
			for _, m := range machineTypeMap {
				if m.Dependence == nil {
					continue
				}
				if m.Dependence.Parent != p.MachineType.Name {
					continue
				}
				childrenNum++
			}
			if childrenNum == 0 {
				return fmt.Errorf("machineType, %s does not have children machineTypes, please set false in machineType.scheduleChildren", p.MachineType.Name)
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
		if m.Spec.GPU.Generation == "" {
			return fmt.Errorf("gpu.generation must be set value")
		}
	}
	return nil
}

func (r *Machine) ValidateDependence() error {
	machineTypeNames := map[string]MachineDetailSpec{}
	for _, mt := range r.Spec.MachineTypes {
		machineTypeNames[mt.Name] = mt.Spec
	}

	for _, mt := range r.Spec.MachineTypes {
		// skip validation if spec.dependence is empty
		if mt.Dependence == nil {
			continue
		}

		parent := mt.Dependence.Parent

		if parent == "" {
			return fmt.Errorf("dependence.parent must be set value")
		}
		if mt.Dependence.AvailableRatio == "" {
			return fmt.Errorf("dependence.availableRatio must be set value")
		}

		// Validate dependency machine name
		if _, exist := machineTypeNames[parent]; !exist {
			return fmt.Errorf("failed to find machine type %s in spec.machineTypes", parent)
		}

		// Validate ratio
		ratio, err := strconv.ParseFloat(mt.Dependence.AvailableRatio, 32)
		if err != nil {
			return fmt.Errorf("name: <%s>, value: <%s>; dependence.availableRatio must be set as type float", mt.Name, mt.Dependence.AvailableRatio)
		}
		if ratio > 1 {
			return fmt.Errorf("name: <%s>, value: <%s>; dependence.availableRatio must not be set greater than 1(availableRatio <= 1)", mt.Name, mt.Dependence.AvailableRatio)
		}

		// Validate CPU
		mustParseParentCPU := machineTypeNames[parent].CPU
		parentCPU := float64(mustParseParentCPU.Value())
		childCPU := float64(mt.Spec.CPU.Value())
		if parentCPU*ratio != childCPU {
			return fmt.Errorf("parent CPU: <%f>, child CPU <%f>; the ratio of number of cpus in child <%s> to the number of one in parent <%s> is wrong", parentCPU, childCPU, mt.Name, parent)
		}

		// Validate Memory
		mustParseParentMemory := machineTypeNames[parent].Memory
		parentMemory := float64(mustParseParentMemory.Value())
		childMemory := float64(mt.Spec.Memory.Value())
		if parentMemory*ratio != childMemory {
			return fmt.Errorf("parent Memory: <%f>, child Memory <%f>; the ratio of memory capacities in child %s to the memory capacities in parent %s is wrong", parentMemory, childMemory, mt.Name, parent)
		}

		// Validate GPU
		parentGPU := machineTypeNames[parent].GPU
		childGPU := mt.Spec.GPU
		if (childGPU != nil) != (parentGPU != nil) {
			return fmt.Errorf("machine name: <%s>; child must be set GPU setting same as parent", mt.Name)
		}
		if mt.Spec.GPU != nil {
			parentGPUNum := float64(parentGPU.Num.Value())
			childGPUNum := float64(childGPU.Num.Value())
			if parentGPUNum*ratio != childGPUNum {
				return fmt.Errorf("the ratio of number of gpus in child; <%s> to the number of one in parent; <%s> is wrong", mt.Name, parent)
			}
			if parentGPU.Type != childGPU.Type {
				return fmt.Errorf("gpu.type must be the same for parent; <%s> and child; <%s>", parent, mt.Name)
			}
			if parentGPU.Generation != childGPU.Generation {
				return fmt.Errorf("gpu.generation must be the same for parent; <%s> and child; <%s>", parent, mt.Name)
			}
		}

	}
	return nil
}
