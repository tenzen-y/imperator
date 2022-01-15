package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/controllers/util"
)

func (r *MachineReconciler) getUnscheduledPodNum(ctx context.Context, podLabels map[string]string, ns string) (int32, error) {
	var unscheduledPodNum int32 = 0

	pods := &corev1.PodList{}
	if err := r.List(ctx, pods, &client.ListOptions{
		Namespace:     ns,
		LabelSelector: labels.SelectorFromSet(podLabels),
	}); err != nil {
		return unscheduledPodNum, err
	}

	if len(pods.Items) == 0 {
		return unscheduledPodNum, nil
	}

	for _, p := range pods.Items {
		if !p.ObjectMeta.DeletionTimestamp.IsZero() {
			continue
		}

		if p.Status.Phase != corev1.PodPending {
			continue
		}

		if p.Spec.NodeName != "" {
			continue
		}

		podConditionTypeMap := util.GetPodConditionTypeMap(p.Status.Conditions)
		scheduledCondition, exist := podConditionTypeMap[corev1.PodScheduled]
		if !exist {
			continue
		}

		if scheduledCondition.Reason == corev1.PodReasonUnschedulable &&
			scheduledCondition.Status == corev1.ConditionFalse {
			unscheduledPodNum++
		}
	}

	return unscheduledPodNum, nil
}

func (r *MachineReconciler) updateReconcileFailedStatus(ctx context.Context, machine *imperatorv1alpha1.Machine, reconcileErr error) (ctrl.Result, error) {
	meta.SetStatusCondition(&machine.Status.Conditions, metav1.Condition{
		Type:               imperatorv1alpha1.ConditionReady,
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             metav1.StatusFailure,
		Message:            reconcileErr.Error(),
	})
	if err := r.Status().Update(ctx, machine, &client.UpdateOptions{}); err != nil {
		return ctrl.Result{Requeue: true}, err
	}
	return ctrl.Result{Requeue: true}, nil
}

func (r *MachineReconciler) updateReconcileSuccessStatus(ctx context.Context, machine *imperatorv1alpha1.Machine) error {
	meta.SetStatusCondition(&machine.Status.Conditions, metav1.Condition{
		Type:               imperatorv1alpha1.ConditionReady,
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             metav1.StatusSuccess,
		Message:            "update status conditions",
	})
	if err := r.Status().Update(ctx, machine, &client.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}

func (r *MachineReconciler) updateReconcileConditions(ctx context.Context, opeResult controllerutil.OperationResult, machine *imperatorv1alpha1.Machine) error {
	if opeResult == controllerutil.OperationResultUpdated || opeResult == controllerutil.OperationResultCreated {
		return r.updateReconcileSuccessStatus(ctx, machine)
	}
	return nil
}
