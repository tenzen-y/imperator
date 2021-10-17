package v1alpha1

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

//+kubebuilder:webhook:path=/mutate-imperator-imprator-io-v1alpha1-machine,mutating=true,failurePolicy=fail,sideEffects=None,groups=imperator.tenzen-y.io,resources=machines,verbs=create;update,versions=v1alpha1,name=mmachine.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Machine{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Machine) Default() {
	machinelog.Info("default", "name", r.Name)

	for _, p := range r.Spec.NodePool {
		if p.AssignmentType == "" {
			p.AssignmentType = AssignmentTypeLabel
		}
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-imperator-imprator-io-v1alpha1-machine,mutating=false,failurePolicy=fail,sideEffects=None,groups=imperator.tenzen-y.io,resources=machines,verbs=create;update,versions=v1alpha1,name=vmachine.kb.io,admissionReviewVersions={v1,v1beta1}

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
	if err := r.ValidateAllOperation(); err != nil {
		return err
	}
	return nil
}

func (r *Machine) ValidateAllOperation() error {
	if err := r.ValidateNodeName(); err != nil {
		return err
	}
	if err := r.ValidateDependence(); err != nil {
		return err
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

	// Validate machineTypes.name
	nodePoolNodes := map[string]bool{}
	for _, n := range r.Spec.NodePool {
		nodePoolNodes[n.Name] = true
	}
	for _, n := range r.Spec.MachineTypes {
		if !nodePoolNodes[n.Name] {
			return fmt.Errorf("failed to find machineType name; %s in nodePool", n.Name)
		}

	}

	return nil
}

func (r *Machine) ValidateDependence() error {
	machineTypeNames := map[string]*MachineDetailSpec{}
	for _, mt := range r.Spec.MachineTypes {
		machineTypeNames[mt.Name] = &mt.Spec
	}

	for _, mt := range r.Spec.MachineTypes {
		if mt.Spec.Dependence != nil {
			parent := mt.Spec.Dependence.Parent

			// Validate dependency machine name
			if _, exist := machineTypeNames[parent]; !exist {
				return fmt.Errorf("failed to find machine type %s in spec.machineTypes", parent)
			}

			// Validate ratio
			ratio, err := strconv.ParseFloat(mt.Spec.Dependence.AvailableRatio, 32)
			if err != nil {
				return fmt.Errorf("name: %s, value: %s; dependence.availableRatio muste be set as type float", mt.Name, mt.Spec.Dependence.AvailableRatio)
			}
			if ratio >= 0 {
				return fmt.Errorf("dependence.availableRatio must be set less than zero(availableRatio < 0)")
			}

			// Validate CPU
			parentCPU := machineTypeNames[parent].CPU.Value()
			childCPU := mt.Spec.CPU.Value()
			if float64(parentCPU)*ratio != float64(childCPU) {
				return fmt.Errorf("the ratio of number of cpus in child; %s to the number of one in parent; %s is wrong", mt.Name, parent)
			}

			// Validate Memory
			parentMemory := machineTypeNames[parent].Memory.Value()
			childMemory := mt.Spec.Memory.Value()
			if float64(parentMemory)*ratio != float64(childMemory) {
				return fmt.Errorf("the ratio of memory capacities in child; %s to the memory capacities in parent; %s is wrong", mt.Name, parent)
			}

			// Validate GPU
			parentGPU := machineTypeNames[parent].GPU
			childGPU := mt.Spec.GPU
			if (childGPU != nil) == (parentGPU != nil) {
				return fmt.Errorf("child must be set GPU setting same as parent; %v", mt.Name)
			}
			if mt.Spec.GPU != nil {
				if float64(parentGPU.Num.Value())*ratio != float64(childGPU.Num.Value()) {
					return fmt.Errorf("the ratio of number of gpus in child; %s to the number of one in parent; %s is wrong", mt.Name, parent)
				}
				if parentGPU.Type != childGPU.Type {
					return fmt.Errorf("gpu.type must be the same for parent; %s and child; %s", parent, mt.Name)
				}
				if parentGPU.Generation != childGPU.Generation {
					return fmt.Errorf("gpu.generation must be the same for parent; %s and child; %s", parent, mt.Name)
				}
			}

		}
	}
	return nil
}
