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

package v1alpha1

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/tenzen-y/imperator/pkg/consts"
)

func GenerateMachineTypeLabelTaintKey(machineTypeName string) string {
	return strings.Join([]string{GroupVersion.Group, machineTypeName}, "/")
}

func getGPUSelector(gpuSpec GPUSpec) []string {
	switch {
	case gpuSpec.Family != "":
		return []string{consts.NvidiaGPUFamilyKey, gpuSpec.Family}
	case gpuSpec.Product != "":
		return []string{consts.NvidiaGPUProductKey, gpuSpec.Product}
	case gpuSpec.Machine != "":
		return []string{consts.NvidiaGPUMachineKey, gpuSpec.Machine}
	}
	return make([]string, 0)
}

func GenerateAffinityMatchExpression(machineType *MachineType, machineGroup string) []corev1.NodeSelectorRequirement {
	machineTypeName := machineType.Name
	machineTypeLabelKey := GenerateMachineTypeLabelTaintKey(machineTypeName)

	affinityMatchExpressions := []corev1.NodeSelectorRequirement{
		{
			Key:      machineTypeLabelKey,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{machineGroup},
		},
		{
			Key:      consts.MachineStatusKey,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{NodeModeReady.Value()},
		},
	}
	if machineType.Spec.GPU != nil {
		// gpuSelector: []string{"nvidia.com/gpu.family", "ampere"}
		if gpuSelector := getGPUSelector(*machineType.Spec.GPU); len(gpuSelector) == 2 {
			affinityMatchExpressions = append(affinityMatchExpressions, corev1.NodeSelectorRequirement{
				Key:      gpuSelector[0],
				Operator: corev1.NodeSelectorOpIn,
				Values:   []string{gpuSelector[1]},
			})
		}
	}

	return affinityMatchExpressions
}

func GenerateToleration(machineTypeName, machineGroup string) []corev1.Toleration {
	machineTypeTolerationKey := GenerateMachineTypeLabelTaintKey(machineTypeName)
	return []corev1.Toleration{
		{
			Key:      machineTypeTolerationKey,
			Effect:   corev1.TaintEffectNoSchedule,
			Operator: corev1.TolerationOpEqual,
			Value:    machineGroup,
		},
		{
			Key:      consts.MachineStatusKey,
			Effect:   corev1.TaintEffectNoSchedule,
			Operator: corev1.TolerationOpEqual,
			Value:    NodeModeReady.Value(),
		},
	}
}
