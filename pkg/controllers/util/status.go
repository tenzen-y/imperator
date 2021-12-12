package util

import (
	corev1 "k8s.io/api/core/v1"

	imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"
)

func GetMachineTypeUsage(availableMachine []imperatorv1alpha1.AvailableMachineCondition, machineType string) *imperatorv1alpha1.UsageCondition {
	for _, cond := range availableMachine {
		if cond.Name != machineType {
			continue
		}
		return &cond.Usage
	}
	return nil
}

func GetPodConditionTypeMap(podConditions []corev1.PodCondition) map[corev1.PodConditionType]corev1.PodCondition {
	result := make(map[corev1.PodConditionType]corev1.PodCondition)
	if len(podConditions) == 0 {
		return result
	}
	for _, c := range podConditions {
		if _, exist := result[c.Type]; exist {
			continue
		}
		result[c.Type] = c
	}
	return result
}
