package controllers

import (
	"context"
	"fmt"
	imperatorv1alpha1 "github.com/tenzen-y/imperator/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// MachineNodePoolReconciler reconciles a MachineNodePool object
type MachineNodePoolReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=imperator.tenzen-y.io,resources=machinenodepools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=imperator.tenzen-y.io,resources=machinenodepools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=imperator.tenzen-y.io,resources=machinenodepools/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=core,resources=nodes/status,verbs=get;list;watch

// Reconcile is main function for reconciliation loop
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

	return r.reconcile(ctx, pool)
}

func (r *MachineNodePoolReconciler) reconcile(ctx context.Context, pool *imperatorv1alpha1.MachineNodePool) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if pool.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(pool, consts.MachineNodePoolFinalizer) {
			controllerutil.AddFinalizer(pool, consts.MachineNodePoolFinalizer)
			if err := r.Update(ctx, pool); err != nil {
				return ctrl.Result{}, err
			}
			r.Recorder.Eventf(pool, corev1.EventTypeNormal, "Updated", "%s, %s: add finalizer", consts.KindMachineNodePool, pool.Name)
			return ctrl.Result{}, nil
		}
	} else {
		if controllerutil.ContainsFinalizer(pool, consts.MachineNodePoolFinalizer) {
			if err := r.removeNodePoolLabel(ctx, pool); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(pool, consts.MachineNodePoolFinalizer)
			if err := r.Update(ctx, pool); err != nil {
				return ctrl.Result{}, err
			}
			r.Recorder.Eventf(pool, corev1.EventTypeNormal, "Updated", "$s, %s: remove finalizer", consts.KindMachineNodePool, pool.Name)
		}
		return ctrl.Result{}, nil
	}

	if err := r.reconcileNode(ctx, pool); err != nil {
		logger.Error(err, "unable to reconcile", "name", pool.Name)
		r.Recorder.Eventf(pool, corev1.EventTypeWarning, "Failed", "MachineNodePool, %s: failed to reconcile: %s", pool.Name, err.Error())
		meta.SetStatusCondition(&pool.Status.Conditions, metav1.Condition{
			Type:    imperatorv1alpha1.ConditionReady,
			Status:  metav1.ConditionFalse,
			Reason:  metav1.StatusFailure,
			Message: err.Error(),
		})
		if updateErr := r.Status().Update(ctx, pool); err != nil {
			logger.Error(updateErr, "failed to update MachineNodePool status", "name", pool.Name)
		}
		return ctrl.Result{}, err
	}

	return r.updateStatus(ctx, pool)
}

func (r *MachineNodePoolReconciler) removeNodePoolLabel(ctx context.Context, pool *imperatorv1alpha1.MachineNodePool) error {
	logger := log.FromContext(ctx)

	for _, p := range pool.Spec.NodePool {
		currentNode := &corev1.Node{}
		if err := r.Get(ctx, client.ObjectKey{Name: p.Name}, currentNode); err != nil && !errors.IsNotFound(err) {
			return err
		}

		nodeLabel := currentNode.GetLabels()
		delete(nodeLabel, consts.MachineGroupKey)
		delete(nodeLabel, consts.MachineStatusKey)

		nodePatch := corev1apply.Node(p.Name).WithLabels(nodeLabel)
		obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(nodePatch)
		if err != nil {
			return err
		}
		patch := &unstructured.Unstructured{
			Object: obj,
		}

		currentApplyConfig, err := corev1apply.ExtractNode(currentNode, consts.ControllerName)
		if err != nil {
			return err
		}

		if equality.Semantic.DeepEqual(nodePatch, currentApplyConfig) {
			return nil
		}

		if err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
			FieldManager: consts.ControllerName,
			Force:        pointer.Bool(true),
		}); err != nil {
			logger.Error(err, fmt.Sprintf("unable to remove label from %s", p.Name), "name", pool.Name)
			return err
		}
		r.Recorder.Eventf(pool, corev1.EventTypeNormal, "Updated", "%s, %s: remove node, %s's label", consts.KindMachineNodePool, pool.Name, p.Name)
	}
	return nil
}

func (r *MachineNodePoolReconciler) reconcileNode(ctx context.Context, pool *imperatorv1alpha1.MachineNodePool) error {
	logger := log.FromContext(ctx)

	for _, p := range pool.Spec.NodePool {

		currentNode := &corev1.Node{}
		if err := r.Get(ctx, client.ObjectKey{Name: p.Name}, currentNode); err != nil && !errors.IsNotFound(err) {
			return err
		}

		nodeLabels := map[string]string{
			consts.MachineGroupKey:  pool.Spec.MachineGroupName,
			consts.MachineStatusKey: consts.MachineStatusReady,
		}
		for _, c := range currentNode.Status.Conditions {
			switch c.Type {
			case corev1.NodeReady:
				if c.Status == corev1.ConditionFalse {
					nodeLabels[consts.MachineStatusKey] = consts.MachineStatusNotReady
				}
			case corev1.NodeNetworkUnavailable:
				if c.Status == corev1.ConditionTrue {
					nodeLabels[consts.MachineStatusKey] = consts.MachineStatusNotReady
				}
			default:
				continue
			}
		}

		nodePatch := corev1apply.Node(p.Name).WithLabels(nodeLabels)
		obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(nodePatch)
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
		patch := &unstructured.Unstructured{
			Object: obj,
		}

		currentApplyConfig, err := corev1apply.ExtractNode(currentNode, consts.ControllerName)
		if err != nil {
			return err
		}

		if equality.Semantic.DeepEqual(nodePatch, currentApplyConfig) {
			return nil
		}

		if err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
			FieldManager: consts.ControllerName,
			Force:        pointer.Bool(true),
		}); err != nil {
			logger.Error(err, "unable to set label to "+p.Name, "name", pool.Name)
			return err
		}
		r.Recorder.Eventf(pool, corev1.EventTypeNormal, "Updated", "%s, %s: update node, %s's label", consts.KindMachineNodePool, pool.Name, p.Name)
	}
	logger.Info("reconcile Node successfully", "name", pool.Name)
	return nil
}

func (r *MachineNodePoolReconciler) updateStatus(ctx context.Context, pool *imperatorv1alpha1.MachineNodePool) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	nodes := &corev1.NodeList{}
	if err := r.List(ctx, nodes, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{consts.MachineGroupKey: pool.Spec.MachineGroupName}),
	}); err != nil && errors.IsNotFound(err) {
		logger.Error(err, "unable to list node managed by imperator", "name", pool.Name)
		return ctrl.Result{}, err
	}

	nodeLabelCondition := map[string]string{}
	for _, n := range nodes.Items {
		nodeLabelCondition[n.Name] = n.GetLabels()[consts.MachineStatusKey]
	}

	var nodeConditions []imperatorv1alpha1.NodePoolCondition
	for _, p := range pool.Spec.NodePool {
		var nc imperatorv1alpha1.MachineNodeCondition
		switch nodeLabelCondition[p.Name] {
		case consts.MachineStatusReady:
			nc = imperatorv1alpha1.NodeReady
		case consts.MachineStatusNotReady:
			nc = imperatorv1alpha1.NodeNotReady
		}
		if p.Mode == "maintenance" {
			nc = imperatorv1alpha1.NodeMaintenance
		}

		nodeConditions = append(nodeConditions, imperatorv1alpha1.NodePoolCondition{
			Name:          p.Name,
			NodeCondition: nc,
		})
	}

	newStatus := &imperatorv1alpha1.MachineNodePoolStatus{}
	if reflect.DeepEqual(pool.Status.NodePoolCondition, nodeConditions) {
		newStatus.NodePoolCondition = pool.Status.NodePoolCondition
	} else {
		newStatus.NodePoolCondition = nodeConditions
	}

	condition := meta.FindStatusCondition(pool.Status.Conditions, imperatorv1alpha1.ConditionReady)
	if meta.IsStatusConditionTrue(pool.Status.Conditions, imperatorv1alpha1.ConditionReady) {
		newStatus.Conditions[0] = *condition
	} else {
		newStatus.Conditions[0] = metav1.Condition{
			Type:   imperatorv1alpha1.ConditionReady,
			Status: metav1.ConditionTrue,
			Reason: metav1.StatusSuccess,
		}
	}

	if !reflect.DeepEqual(pool.Status, newStatus) {
		pool.Status = *newStatus
		r.Recorder.Eventf(pool, corev1.EventTypeNormal, "Updated", "%s, %s: updated condition", consts.KindMachineNodePool, pool.Name)
		if err := r.Status().Update(ctx, pool); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MachineNodePoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	nodeHandler := handler.EnqueueRequestsFromMapFunc(r.nodeReconcileTrigger)

	return ctrl.NewControllerManagedBy(mgr).
		For(&imperatorv1alpha1.MachineNodePool{}).
		Watches(&source.Kind{Type: &corev1.Node{}}, nodeHandler).
		Complete(r)
}

func (r *MachineNodePoolReconciler) nodeReconcileTrigger(o client.Object) []reconcile.Request {
	return r.nodeReconcileRequest(o)
}

func (r *MachineNodePoolReconciler) nodeReconcileRequest(o client.Object) []reconcile.Request {
	pools := &imperatorv1alpha1.MachineNodePoolList{}
	if err := r.List(context.Background(), pools, &client.ListOptions{}); err != nil {
		return nil
	}

	var req []reconcile.Request
	for _, p := range pools.Items {
		if _, ok := p.ObjectMeta.Labels[consts.MachineGroupKey]; !ok {
			continue
		}
		if p.Spec.MachineGroupName == o.GetLabels()[consts.MachineGroupKey] {
			req = append(req, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&p)})
		}
	}
	return req
}
