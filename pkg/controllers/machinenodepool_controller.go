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

package controllers

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
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

	imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
	"github.com/tenzen-y/imperator/pkg/controllers/util"
)

// MachineNodePoolReconciler reconciles a MachineNodePool object
type MachineNodePoolReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=imperator.tenzen-y.io,resources=machinenodepools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=imperator.tenzen-y.io,resources=machinenodepools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=imperator.tenzen-y.io,resources=machinenodepools/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=core,resources=nodes/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;update;patch

// Reconcile is main function for reconciliation loop
func (r *MachineNodePoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	pool := &imperatorv1alpha1.MachineNodePool{}
	logger.Info("fetching MachineNodePool Resource")

	err := r.Get(ctx, req.NamespacedName, pool)
	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	} else if err != nil {
		logger.Error(err, "unable to get MachineNodePool", "name", req.NamespacedName)
		return ctrl.Result{Requeue: true}, err
	}

	return r.reconcile(ctx, pool)
}

func (r *MachineNodePoolReconciler) reconcile(ctx context.Context, pool *imperatorv1alpha1.MachineNodePool) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if pool.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(pool, consts.MachineNodePoolFinalizer) {
			controllerutil.AddFinalizer(pool, consts.MachineNodePoolFinalizer)
			if err := r.Update(ctx, pool); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			r.Recorder.Eventf(pool, corev1.EventTypeNormal, "Updated", "add finalizer")
			return ctrl.Result{}, nil
		}
	} else {
		if controllerutil.ContainsFinalizer(pool, consts.MachineNodePoolFinalizer) {
			if err := r.cleanupNode(ctx, pool); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			controllerutil.RemoveFinalizer(pool, consts.MachineNodePoolFinalizer)
			if err := r.Update(ctx, pool); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			r.Recorder.Eventf(pool, corev1.EventTypeNormal, "Updated", "remove finalizer")
		}
		return ctrl.Result{}, nil
	}

	if err := r.reconcileNode(ctx, pool); err != nil {
		logger.Error(err, "failed to reconcile", "name", pool.Name)
		meta.SetStatusCondition(&pool.Status.Conditions, metav1.Condition{
			Type:               imperatorv1alpha1.ConditionReady,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             metav1.StatusFailure,
			Message:            err.Error(),
		})
		if updateErr := r.Status().Update(ctx, pool, &client.UpdateOptions{}); updateErr != nil {
			logger.Error(updateErr, "failed to update MachineNodePool status")
		}
		return ctrl.Result{Requeue: true}, err
	}

	return r.updateStatus(ctx, pool)
}

func (r *MachineNodePoolReconciler) cleanupNode(ctx context.Context, pool *imperatorv1alpha1.MachineNodePool) error {
	logger := log.FromContext(ctx)
	for _, p := range pool.Spec.NodePool {

		node := &corev1.Node{}
		if err := r.Get(ctx, client.ObjectKey{Name: p.Name}, node); err != nil && !errors.IsNotFound(err) {
			return err
		}
		originNode := node.DeepCopy()

		r.removeNodeAnnotation(node)
		annotationDiff := cmp.Diff(originNode.Annotations, node.Annotations)
		if annotationDiff != "" {
			logger.Info(annotationDiff, "nodeName", node.Name)
		}

		r.removeNodeLabel(pool, node)
		labelDiff := cmp.Diff(originNode.Labels, node.Labels)
		if labelDiff != "" {
			logger.Info(labelDiff, "nodeName", node.Name)
		}

		r.removeNodeTaint(pool, node)
		taintDiff := cmp.Diff(originNode.Spec.Taints, node.Spec.Taints, consts.CmpSliceOpts...)
		if taintDiff != "" {
			logger.Info(taintDiff, "nodeName", node.Name)
		}

		if annotationDiff == "" && labelDiff == "" && taintDiff == "" {
			return nil
		}

		if err := r.Update(ctx, node, &client.UpdateOptions{}); err != nil {
			logger.Error(err, fmt.Sprintf("unable to remove annotation, label or taint from %s", node.Name), "nodeName", node.Name)
			return err
		}

		r.Recorder.Eventf(pool, corev1.EventTypeNormal, "Updated", fmt.Sprintf("cleanup annotation, label and taint from %s", node.Name))

	}
	return nil
}

func (r *MachineNodePoolReconciler) removeNodeAnnotation(node *corev1.Node) {
	delete(node.Annotations, consts.MachineGroupKey)
}

func (r *MachineNodePoolReconciler) removeNodeLabel(pool *imperatorv1alpha1.MachineNodePool, node *corev1.Node) {
	// remove machine status from label
	delete(node.Labels, consts.MachineStatusKey)

	// remove machineType from label
	scheduleMachineTypeKeys := make(map[string][]string, len(pool.Spec.NodePool))
	for _, p := range pool.Spec.NodePool {
		scheduleMachineTypeKeys[p.Name] = util.GetScheduleMachineTypeKeys(p.MachineType)
	}
	if keys, exist := scheduleMachineTypeKeys[node.Name]; exist {
		for _, mtKey := range keys {
			if node.Labels[mtKey] != pool.Spec.MachineGroupName {
				continue
			}
			delete(node.Labels, mtKey)
		}
	}
}

func (r *MachineNodePoolReconciler) removeNodeTaint(pool *imperatorv1alpha1.MachineNodePool, node *corev1.Node) {
	taints := util.ExtractKeyValueFromTaint(node.Spec.Taints)

	// if taint has machine-status, remove it.
	// remove machine status from taint
	if _, exist := taints[consts.MachineStatusKey]; exist {
		for index, t := range node.Spec.Taints {
			if t.Key != consts.MachineStatusKey {
				continue
			}
			node.Spec.Taints = append(node.Spec.Taints[:index], node.Spec.Taints[index+1:]...)
		}
	}

	// remove machineType from taint
	scheduleMachineTypeKeys := make(map[string][]string, len(pool.Spec.NodePool))
	for _, p := range pool.Spec.NodePool {
		scheduleMachineTypeKeys[p.Name] = util.GetScheduleMachineTypeKeys(p.MachineType)
	}
	if keys, exist := scheduleMachineTypeKeys[node.Name]; exist {
		for _, mtKey := range keys {
			idx := util.GetTaintKeyIndex(node.Spec.Taints, mtKey)
			if idx == nil {
				continue
			}
			node.Spec.Taints = append(node.Spec.Taints[:*idx], node.Spec.Taints[*idx+1:]...)
		}
	}
}

func (r *MachineNodePoolReconciler) reconcileNode(ctx context.Context, pool *imperatorv1alpha1.MachineNodePool) error {
	logger := log.FromContext(ctx)

	for _, p := range pool.Spec.NodePool {

		// cleanup old env
		node := &corev1.Node{}
		if err := r.Get(ctx, client.ObjectKey{Name: p.Name}, node); err != nil {
			return err
		}
		originNode := node.DeepCopy()
		r.removeNodeLabel(pool, node)
		if p.Taint {
			r.removeNodeTaint(pool, node)
		}

		// set annotation
		if node.Annotations == nil {
			node.Annotations = make(map[string]string)
		}
		node.Annotations[consts.MachineGroupKey] = pool.Spec.MachineGroupName

		taints := util.ExtractKeyValueFromTaint(node.Spec.Taints)
		newPoolMachineStatusValue := imperatorv1alpha1.NodeModeReady
		// looking for down Node.
		for _, t := range consts.CannotUseNodeTaints {
			if _, exist := taints[t]; !exist {
				continue
			}
			if os.Getenv("ENVTEST") == "true" && t == consts.NodeNotReadyTaint {
				continue
			}
			newPoolMachineStatusValue = imperatorv1alpha1.NodeModeNotReady
			break
		}

		scheduleMachineTypeKey := util.GetScheduleMachineTypeKeys(p.MachineType)

		if p.Mode == imperatorv1alpha1.NodeModeMaintenance {
			newPoolMachineStatusValue = imperatorv1alpha1.NodeModeMaintenance
		}

		// Set Label to Node
		if node.Labels == nil {
			node.Labels = make(map[string]string)
		}
		// set machine status to label
		node.Labels[consts.MachineStatusKey] = newPoolMachineStatusValue.Value()
		// set machineType to label
		for _, mtKey := range scheduleMachineTypeKey {
			node.Labels[mtKey] = pool.Spec.MachineGroupName
		}

		if p.Taint {
			if node.Spec.Taints == nil {
				node.Spec.Taints = make([]corev1.Taint, 0)
			}

			// set machine status to taint
			now := metav1.Now()
			if _, exist := taints[consts.MachineStatusKey]; !exist {
				node.Spec.Taints = append(node.Spec.Taints, corev1.Taint{
					Key:       consts.MachineStatusKey,
					Value:     newPoolMachineStatusValue.Value(),
					Effect:    corev1.TaintEffectNoSchedule,
					TimeAdded: &now,
				})
			} else {
				if idx := util.GetTaintKeyIndex(node.Spec.Taints, consts.MachineStatusKey); idx != nil {
					node.Spec.Taints[*idx] = corev1.Taint{
						Key:       consts.MachineStatusKey,
						Value:     newPoolMachineStatusValue.Value(),
						Effect:    corev1.TaintEffectNoSchedule,
						TimeAdded: &now,
					}
				}
			}

			// set machineType to taint
			for _, mtKey := range scheduleMachineTypeKey {
				if _, exist := taints[mtKey]; !exist {
					node.Spec.Taints = append(node.Spec.Taints, corev1.Taint{
						Key:       mtKey,
						Value:     pool.Spec.MachineGroupName,
						Effect:    corev1.TaintEffectNoSchedule,
						TimeAdded: &now,
					})
				} else {
					if idx := util.GetTaintKeyIndex(node.Spec.Taints, mtKey); idx != nil {
						node.Spec.Taints[*idx] = corev1.Taint{
							Key:       mtKey,
							Value:     pool.Spec.MachineGroupName,
							Effect:    corev1.TaintEffectNoSchedule,
							TimeAdded: &now,
						}
					}
				}
			}
		}

		taintDiff := cmp.Diff(originNode.Spec.Taints, node.Spec.Taints, consts.CmpSliceOpts...)
		if taintDiff != "" {
			logger.Info(taintDiff, "nodeName", node.Name)
		}

		labelDiff := cmp.Diff(originNode.Labels, node.Labels)
		if labelDiff != "" {
			logger.Info(labelDiff, "nodeName", node.Name)
		}

		if taintDiff == "" && labelDiff == "" {
			continue
		}

		if err := r.Update(ctx, node, &client.UpdateOptions{}); err != nil {
			logger.Error(err, fmt.Sprintf("unable to set Label and Taint to %s", node.Name), "MachineNodePool", pool.Name)
			return err
		}

	}
	logger.Info("reconcile Node successfully", "name", pool.Name)
	return nil
}

func (r *MachineNodePoolReconciler) updateStatus(ctx context.Context, pool *imperatorv1alpha1.MachineNodePool) (ctrl.Result, error) {
	var nodeConditions []imperatorv1alpha1.NodePoolCondition
	for _, p := range pool.Spec.NodePool {
		node := &corev1.Node{}
		if err := r.Get(ctx, client.ObjectKey{Name: p.Name}, node); err != nil {
			return ctrl.Result{Requeue: true}, err
		}

		nodeLabelCondition := node.Labels[consts.MachineStatusKey]
		if p.Taint {
			for _, t := range node.Spec.Taints {
				if t.Key != consts.MachineStatusKey {
					continue
				}
				nodeLabelCondition = t.Value
			}
		}

		nc := imperatorv1alpha1.MachineNodeCondition("")
		if p.Mode == imperatorv1alpha1.NodeModeMaintenance {
			nc = imperatorv1alpha1.NodeMaintenance
		} else if nodeLabelCondition == imperatorv1alpha1.NodeModeReady.Value() {
			nc = imperatorv1alpha1.NodeHealthy
		} else if nodeLabelCondition == imperatorv1alpha1.NodeModeNotReady.Value() {
			nc = imperatorv1alpha1.NodeUnhealthy
		}

		nodeConditions = append(nodeConditions, imperatorv1alpha1.NodePoolCondition{
			Name:          p.Name,
			NodeCondition: nc,
		})
	}

	if !cmp.Equal(pool.Status.NodePoolCondition, nodeConditions, consts.CmpSliceOpts...) {
		r.Recorder.Eventf(pool, corev1.EventTypeNormal, "Updated", "updated Node condition in status")
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
func (r *MachineNodePoolReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	nodeHandler := handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
		return r.nodeReconcileRequest(ctx, o)
	})

	nodePredicates := predicate.Funcs{
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(event event.UpdateEvent) bool {
			newLabels := event.ObjectNew.(*corev1.Node).Labels
			oldLabels := event.ObjectOld.(*corev1.Node).Labels
			if !cmp.Equal(newLabels, oldLabels) {
				return true
			}

			newAnnotations := event.ObjectNew.(*corev1.Node).Annotations
			oldAnnotations := event.ObjectOld.(*corev1.Node).Annotations
			if !cmp.Equal(newAnnotations, oldAnnotations) {
				return true
			}

			newTaints := event.ObjectNew.(*corev1.Node).Spec.Taints
			oldTaints := event.ObjectOld.(*corev1.Node).Spec.Taints
			return !cmp.Equal(newTaints, oldTaints, consts.CmpSliceOpts...)
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

func (r *MachineNodePoolReconciler) nodeReconcileRequest(ctx context.Context, o client.Object) []reconcile.Request {
	pools := &imperatorv1alpha1.MachineNodePoolList{}
	if err := r.List(ctx, pools, &client.ListOptions{}); err != nil {
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
