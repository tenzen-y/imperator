package controllers

import (
	"context"
	"fmt"
	imperatorv1alpha1 "github.com/tenzen-y/imperator/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"os"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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
			if err := r.cleanupNode(ctx, pool); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(pool, consts.MachineNodePoolFinalizer)
			if err := r.Update(ctx, pool); err != nil {
				return ctrl.Result{}, err
			}
			r.Recorder.Eventf(pool, corev1.EventTypeNormal, "Updated", "%s, %s: remove finalizer", consts.KindMachineNodePool, pool.Name)
		}
		return ctrl.Result{}, nil
	}

	if err := r.reconcileNode(ctx, pool); err != nil {
		logger.Error(err, "unable to reconcile", "name", pool.Name)
		r.Recorder.Eventf(pool, corev1.EventTypeWarning, "Failed", "MachineNodePool, %s: failed to reconcile: %s", pool.Name, err.Error())
		meta.SetStatusCondition(&pool.Status.Conditions, metav1.Condition{
			Type:               imperatorv1alpha1.ConditionReady,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             metav1.StatusFailure,
			Message:            err.Error(),
		})
		if updateErr := r.Status().Update(ctx, pool, &client.UpdateOptions{}); updateErr != nil {
			logger.Error(updateErr, "failed to update MachineNodePool status", "name", pool.Name)
		}
		return ctrl.Result{Requeue: true}, err
	}

	return r.updateStatus(ctx, pool)
}

func (r *MachineNodePoolReconciler) cleanupNode(ctx context.Context, pool *imperatorv1alpha1.MachineNodePool) error {
	for _, p := range pool.Spec.NodePool {

		node := &corev1.Node{}
		if err := r.Get(ctx, client.ObjectKey{Name: p.Name}, node); err != nil && !errors.IsNotFound(err) {
			return err
		}
		if err := r.removeNodeLabel(ctx, pool, node); err != nil {
			return err
		}
		if err := r.removeNodeTaint(ctx, pool, node); err != nil {
			return err
		}

		// initialize node
		node = &corev1.Node{}
		if err := r.Get(ctx, client.ObjectKey{Name: p.Name}, node); err != nil && !errors.IsNotFound(err) {
			return err
		}
		if err := r.removeNodeAnnotation(ctx, pool, node); err != nil {
			return err
		}

	}
	return nil
}

func (r *MachineNodePoolReconciler) removeNodeAnnotation(ctx context.Context, pool *imperatorv1alpha1.MachineNodePool, node *corev1.Node) error {
	logger := log.FromContext(ctx)
	delete(node.Annotations, consts.MachineGroupKey)
	if err := r.Update(ctx, node, &client.UpdateOptions{}); err != nil {
		logger.Error(err, fmt.Sprintf("unable to remove annotation from %s", node.Name), "name", pool.Name)
		return err
	}
	r.Recorder.Eventf(pool, corev1.EventTypeNormal, "Updated", "%s, %s: remove node annotation from %s", consts.KindMachineNodePool, pool.Name, node.Name)
	return nil
}

func (r *MachineNodePoolReconciler) removeNodeLabel(ctx context.Context, pool *imperatorv1alpha1.MachineNodePool, node *corev1.Node) error {
	logger := log.FromContext(ctx)
	newNode := node.DeepCopy()

	delete(newNode.Labels, consts.MachineStatusKey)

	if reflect.DeepEqual(node.Labels, newNode.Labels) {
		return nil
	}

	if err := r.Update(ctx, newNode, &client.UpdateOptions{}); err != nil {
		logger.Error(err, fmt.Sprintf("unable to remove label from %s", node.Name), "name", pool.Name)
		return err
	}
	r.Recorder.Eventf(pool, corev1.EventTypeNormal, "Updated", "%s, %s: remove node label from %s", consts.KindMachineNodePool, pool.Name, node.Name)

	return nil
}

func (r *MachineNodePoolReconciler) removeNodeTaint(ctx context.Context, pool *imperatorv1alpha1.MachineNodePool, node *corev1.Node) error {
	logger := log.FromContext(ctx)

	newNode := node.DeepCopy()
	taints := map[string]corev1.Taint{}
	for _, t := range newNode.Spec.Taints {
		taints[t.Key] = t
	}

	// if taint has machine-status, remove it.
	if _, exist := taints[consts.MachineStatusKey]; exist {
		for index, t := range newNode.Spec.Taints {
			if t.Key != consts.MachineStatusKey {
				continue
			}
			newNode.Spec.Taints = append(newNode.Spec.Taints[:index], newNode.Spec.Taints[index+1:]...)
		}
	}

	if reflect.DeepEqual(node.Spec.Taints, newNode.Spec.Taints) {
		return nil
	}

	if err := r.Update(ctx, newNode, &client.UpdateOptions{}); err != nil {
		logger.Error(err, fmt.Sprintf("unable to remove taint from %s", newNode.Name), "name", pool.Name)
		return err
	}
	r.Recorder.Eventf(pool, corev1.EventTypeNormal, "Updated", "%s, %s: remove node taint from %s", consts.KindMachineNodePool, pool.Name, newNode.Name)

	return nil
}

func (r *MachineNodePoolReconciler) reconcileNode(ctx context.Context, pool *imperatorv1alpha1.MachineNodePool) error {
	logger := log.FromContext(ctx)

	for _, p := range pool.Spec.NodePool {

		// cleanup old env
		node := &corev1.Node{}
		if err := r.Get(ctx, client.ObjectKey{Name: p.Name}, node); err != nil {
			return err
		}
		switch p.AssignmentType {
		case consts.AssignLabel:
			if err := r.removeNodeTaint(ctx, pool, node); err != nil {
				return err
			}
		case consts.AssignTaint:
			if err := r.removeNodeLabel(ctx, pool, node); err != nil {
				return err
			}
		}

		// initialize Node
		node = &corev1.Node{}
		if err := r.Get(ctx, client.ObjectKey{Name: p.Name}, node); err != nil {
			return err
		}

		newNode := node.DeepCopy()
		// set annotation
		if newNode.Annotations == nil {
			newNode.Annotations = map[string]string{}
		}
		newNode.Annotations[consts.MachineGroupKey] = pool.Spec.MachineGroupName

		taints := map[string]corev1.Taint{}
		for _, t := range newNode.Spec.Taints {
			taints[t.Key] = t
		}

		newPoolMachineStatusValue := consts.MachineStatusReady
		// looking for down Node.
		for _, t := range consts.CannotUseNodeTaints {
			if _, exist := taints[t]; !exist {
				continue
			}
			if os.Getenv("ENVTEST") == "true" && t == "node.kubernetes.io/not-ready" {
				continue
			}
			newPoolMachineStatusValue = consts.MachineStatusNotReady
			break
		}
		if p.Mode == consts.MachineStatusMaintenance {
			newPoolMachineStatusValue = consts.MachineStatusMaintenance
		}

		switch p.AssignmentType {
		case consts.AssignLabel:
			if newNode.Labels == nil {
				newNode.Labels = map[string]string{}
			}
			newNode.Labels[consts.MachineStatusKey] = newPoolMachineStatusValue
			if reflect.DeepEqual(node.Labels, newNode.Labels) {
				continue
			}
		case consts.AssignTaint:
			if newNode.Spec.Taints == nil {
				newNode.Spec.Taints = []corev1.Taint{}
			}

			// set machine status to taint
			now := metav1.Now()
			if _, exist := taints[consts.MachineStatusKey]; !exist {
				newNode.Spec.Taints = append(newNode.Spec.Taints, corev1.Taint{
					Key:       consts.MachineStatusKey,
					Value:     newPoolMachineStatusValue,
					Effect:    corev1.TaintEffectNoSchedule,
					TimeAdded: &now,
				})
			} else {
				for i, t := range newNode.Spec.Taints {
					if t.Key != consts.MachineStatusKey {
						continue
					}
					newNode.Spec.Taints[i] = corev1.Taint{
						Key:       consts.MachineStatusKey,
						Value:     newPoolMachineStatusValue,
						Effect:    corev1.TaintEffectNoSchedule,
						TimeAdded: &now,
					}
				}
			}

			if reflect.DeepEqual(node.Spec.Taints, newNode.Spec.Taints) {
				continue
			}
		}

		if err := r.Update(ctx, newNode, &client.UpdateOptions{}); err != nil {
			logger.Error(err, fmt.Sprintf("unable to set %s to %s", p.AssignmentType, p.Name), "name", pool.Name)
			return err
		}
		r.Recorder.Eventf(pool, corev1.EventTypeNormal, "Updated", "%s, %s: update node; add %s to %s", consts.KindMachineNodePool, pool.Name, p.AssignmentType, p.Name)
	}
	logger.Info("reconcile Node successfully", "name", pool.Name)
	return nil
}

func (r *MachineNodePoolReconciler) updateStatus(ctx context.Context, pool *imperatorv1alpha1.MachineNodePool) (ctrl.Result, error) {
	var nodeConditions []imperatorv1alpha1.NodePoolCondition
	for _, p := range pool.Spec.NodePool {
		node := &corev1.Node{}
		if err := r.Get(ctx, client.ObjectKey{Name: p.Name}, node); err != nil {
			return ctrl.Result{}, err
		}

		nodeLabelCondition := node.Labels[consts.MachineStatusKey]
		if p.AssignmentType == consts.AssignTaint {
			for _, t := range node.Spec.Taints {
				if t.Key != consts.MachineStatusKey {
					continue
				}
				nodeLabelCondition = t.Value
			}
		}

		nc := imperatorv1alpha1.MachineNodeCondition("")
		if p.Mode == consts.MachineStatusMaintenance {
			nc = imperatorv1alpha1.NodeMaintenance
		} else if nodeLabelCondition == consts.MachineStatusReady {
			nc = imperatorv1alpha1.NodeHealthy
		} else if nodeLabelCondition == consts.MachineStatusNotReady {
			nc = imperatorv1alpha1.NodeUnhealthy
		}

		nodeConditions = append(nodeConditions, imperatorv1alpha1.NodePoolCondition{
			Name:          p.Name,
			NodeCondition: nc,
		})
	}

	if !reflect.DeepEqual(pool.Status.NodePoolCondition, nodeConditions) {
		r.Recorder.Eventf(pool, corev1.EventTypeNormal, "Updated", "%s, %s: updated MachineNodePool condition", consts.KindMachineNodePool, pool.Name)
		pool.Status.NodePoolCondition = nodeConditions
		meta.SetStatusCondition(&pool.Status.Conditions, metav1.Condition{
			Type:               imperatorv1alpha1.ConditionReady,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             metav1.StatusSuccess,
			Message:            "update status conditions",
		})
		if err := r.Status().Update(ctx, pool, &client.UpdateOptions{}); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MachineNodePoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	nodeHandler := handler.EnqueueRequestsFromMapFunc(r.nodeReconcileTrigger)

	nodePredicates := predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(event event.UpdateEvent) bool {
			newLabels := event.ObjectNew.(*corev1.Node).Labels
			oldLabels := event.ObjectOld.(*corev1.Node).Labels
			if !reflect.DeepEqual(newLabels, oldLabels) {
				return true
			}

			newAnnotations := event.ObjectNew.(*corev1.Node).Annotations
			oldAnnotations := event.ObjectOld.(*corev1.Node).Annotations
			if !reflect.DeepEqual(newAnnotations, oldAnnotations) {
				return true
			}

			newTaints := event.ObjectNew.(*corev1.Node).Spec.Taints
			oldTaints := event.ObjectOld.(*corev1.Node).Spec.Taints
			return !reflect.DeepEqual(newTaints, oldTaints)
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&imperatorv1alpha1.MachineNodePool{}).
		Watches(&source.Kind{Type: &corev1.Node{}}, nodeHandler, builder.WithPredicates(nodePredicates)).
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
	for _, pool := range pools.Items {
		if pool.Spec.MachineGroupName == o.(*corev1.Node).Annotations[consts.MachineGroupKey] {
			req = append(req, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&pool)})
		}
	}
	return req
}
