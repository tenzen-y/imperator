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

package util

import (
	corev1 "k8s.io/api/core/v1"

	imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"
)

func GetMachineTypeUsage(availableMachine []imperatorv1alpha1.AvailableMachineCondition, machineTypeName string) *imperatorv1alpha1.UsageCondition {
	for _, cond := range availableMachine {
		if cond.Name != machineTypeName {
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
