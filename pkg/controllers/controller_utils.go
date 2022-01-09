package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
