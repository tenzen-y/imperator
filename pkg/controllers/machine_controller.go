package controllers

import (
	"context"
	cmp "github.com/google/go-cmp/cmp"
	imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

// MachineReconciler reconciles a Machine object
type MachineReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=imperator.tenzen-y.io,resources=machines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=imperator.tenzen-y.io,resources=machines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=imperator.tenzen-y.io,resources=machines/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Machine object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *MachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	machine := &imperatorv1alpha1.Machine{}
	logger.Info("fetching Machine Resource")

	err := r.Get(ctx, req.NamespacedName, machine)
	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}

	return r.reconcile(ctx, machine)
}

func (r *MachineReconciler) reconcile(ctx context.Context, machine *imperatorv1alpha1.Machine) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if err := r.reconcileMachineNodePool(ctx, machine); err != nil {
		logger.Error(err, "unable to reconcile MachineNodePool", "name", machine.Name)
		r.Recorder.Eventf(machine, corev1.EventTypeWarning, "Failed", "Machine, %s: failed to reconcile MachineNodePool: %s", machine.Name, err.Error())
		meta.SetStatusCondition(&machine.Status.Conditions, metav1.Condition{
			Type: imperatorv1alpha1.ConditionReady,
			Status: metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason: metav1.StatusFailure,
			Message: err.Error(),
		})
		return ctrl.Result{Requeue: true}, err
	}
	if err := r.reconcileStatefulSet(ctx, machine); err != nil {
		logger.Error(err, "unable to reconcile StateFulSet", "name", machine.Name)
		r.Recorder.Eventf(machine, corev1.EventTypeWarning, "Failed", "Machine, %s: failed to reconcile StatefulSet: %s", machine.Name, err.Error())
		meta.SetStatusCondition(&machine.Status.Conditions, metav1.Condition{
			Type: imperatorv1alpha1.ConditionReady,
			Status: metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason: metav1.StatusFailure,
			Message: err.Error(),
		})
		return ctrl.Result{Requeue: true}, err
	}

	return r.updateStatus(ctx, machine)
}

func (r *MachineReconciler) reconcileMachineNodePool(ctx context.Context, machine *imperatorv1alpha1.Machine) error {
	logger := log.FromContext(ctx)

	pool := &imperatorv1alpha1.MachineNodePool{
		TypeMeta: metav1.TypeMeta{
			Kind: consts.KindMachine,
			APIVersion: strings.Join([]string{
				imperatorv1alpha1.GroupVersion.Group,
				imperatorv1alpha1.GroupVersion.Version,
			}, "/"),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: strings.Join([]string{
				machine.Labels[consts.MachineGroupKey],
				"node-pool",
			}, "-"),
		},
	}

	origin := &imperatorv1alpha1.MachineNodePoolSpec{}
	opeResult, err := ctrl.CreateOrUpdate(ctx, r.Client, pool, func() error {
		origin = pool.Spec.DeepCopy()

		if pool.Labels == nil {
			pool.Labels = map[string]string{}
		}
		if _, exist := pool.Labels[consts.MachineGroupKey]; !exist {
			pool.Labels[consts.MachineGroupKey] = machine.Labels[consts.MachineGroupKey]
		}

		nodePoolSpec := machine.Spec.DeepCopy().NodePool
		pool.Spec.NodePool = nodePoolSpec
		return ctrl.SetControllerReference(machine, pool, r.Scheme)
	})
	if err != nil {
		logger.Error(err, "failed to reconcile MachineNodePool")
	}
	if opeResult != controllerutil.OperationResultNone {
		logger.Info("reconciled MachineNodePool", string(opeResult))
	}
	if opeResult == controllerutil.OperationResultUpdated {
		logger.Info(cmp.Diff(origin, pool))
	}

	return nil
}

func (r *MachineReconciler) reconcileStatefulSet(ctx context.Context, machine *imperatorv1alpha1.Machine) error {
	//logger := log.FromContext(ctx)
	return nil
}

func (r *MachineReconciler) updateStatus(ctx context.Context, machine *imperatorv1alpha1.Machine) (ctrl.Result ,error) {
	//logger := log.FromContext(ctx)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&imperatorv1alpha1.Machine{}).
		Owns(&imperatorv1alpha1.MachineNodePool{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}
