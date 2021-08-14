package controllers

import (
	"context"
	imperatorv1alpha1 "github.com/tenzen-y/imperator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func findMachineNodePoolCondition(conditions []imperatorv1alpha1.MachineNodePoolCondition, conditionType imperatorv1alpha1.MachineNodePoolConditionType) *imperatorv1alpha1.MachineNodePoolCondition{
	for _, c := range conditions {
		if c.Type == conditionType {
			return &c
		}
	}
	return nil
}

// MachineNodePoolReconciler reconciles a MachineNodePool object
type MachineNodePoolReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=imperator.imprator.io,resources=machinenodepools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=imperator.imprator.io,resources=machinenodepools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=imperator.imprator.io,resources=machinenodepools/finalizers,verbs=update

func (r *MachineNodePoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	pool := &imperatorv1alpha1.MachineNodePool{}
	logger.Info("fetching MachineNodePool Resource")

	err := r.Get(ctx, req.NamespacedName, pool)
	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		logger.Error(err, "unable to get MachineNodePool", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	if !pool.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	err = r.reconcile(ctx, req, pool)
	if err != nil {
		logger.Error(err, "unable to reconcile", "name", req.NamespacedName)
		r.Recorder.Eventf(pool, corev1.EventTypeWarning, "Failed", "failed to reconcile: %s", err.Error())

		newCondition :=	&imperatorv1alpha1.MachineNodePoolCondition{
			Type: imperatorv1alpha1.ConditionReady,
			Status: corev1.ConditionFalse,
			Reason: err.Error(),
		}

		current := findMachineNodePoolCondition(pool.Status.Conditions, newCondition.Type)
		if current == nil {
			newCondition.LastTransitionTime = metav1.Now()
			pool.Status.Conditions = append(pool.Status.Conditions, *newCondition)
			return ctrl.Result{}, err
		}
		if current.Status != newCondition.Status {
			current.Status = newCondition.Status
			current.LastTransitionTime = metav1.Now()
		}
		current.Reason = newCondition.Reason
		if err = r.Status().Update(ctx, pool); err != nil {
		logger.Error(err, "failed to update status.condition", "name", req.NamespacedName)
		}
		return ctrl.Result{}, err
	}

	return r.updateStatus(ctx, pool)
}

func (r *MachineNodePoolReconciler) updateStatus(ctx context.Context, pool *imperatorv1alpha1.MachineNodePool) (ctrl.Result, error){

	return ctrl.Result{}, nil
}

func (r *MachineNodePoolReconciler) reconcile(ctx context.Context, req ctrl.Request, pool *imperatorv1alpha1.MachineNodePool) error {
	return nil
}

func (r *MachineNodePoolReconciler) reconcileStatefulSet(ctx context.Context, req ctrl.Request, pool *imperatorv1alpha1.MachineNodePoolSpec) error {
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MachineNodePoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&imperatorv1alpha1.MachineNodePool{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}
