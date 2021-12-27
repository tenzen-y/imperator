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
	"testing"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
)

const (
	testMachineGroup = "test-machine-group"
)

func TestGenerateStatefulSet(t *testing.T) {
	testCases := []struct {
		description  string
		machineType  *imperatorv1alpha1.MachineType
		machineGroup string
		replica      int32
	}{
		{
			description: "Normal machineType",
			machineType: newFakeMachineType(false),
		},
		{
			description: "machineType with GPUs",
			machineType: newFakeMachineType(true),
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			expected := newFakeStatefulSet(test.machineType)
			actual := &appsv1.StatefulSet{}
			GenerateStatefulSet(test.machineType, testMachineGroup, 1, actual)
			if diff := cmp.Diff(actual, expected, consts.CmpSliceOpts...); diff != "" {
				t.Errorf("DIFF: \n%v\n", diff)
			}
		})
	}
}

func TestGenerateService(t *testing.T) {
	testCases := []struct {
		description     string
		machineTypeName string
	}{
		{
			description:     "Positive test",
			machineTypeName: "test-machine-type",
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			expected := newFakeService(test.machineTypeName)
			actual := &corev1.Service{}
			GenerateService(test.machineTypeName, testMachineGroup, actual)
			if diff := cmp.Diff(actual, expected, consts.CmpSliceOpts...); diff != "" {
				t.Errorf("DIFF: \n%v\n", diff)
			}
		})
	}
}

func newFakeMachineType(useGpu bool) *imperatorv1alpha1.MachineType {
	fakeMachineType := &imperatorv1alpha1.MachineType{
		Name: "fake-machine-type",
		Spec: imperatorv1alpha1.MachineDetailSpec{
			CPU:    resource.MustParse("4000m"),
			Memory: resource.MustParse("12Gi"),
		},
		Available: 1,
	}
	if useGpu {
		fakeMachineType.Spec.GPU = &imperatorv1alpha1.GPUSpec{
			Type:   "nvidia.com/gpu",
			Num:    resource.MustParse("1"),
			Family: "turing",
		}
	}
	return fakeMachineType
}

func newFakeStatefulSet(machineType *imperatorv1alpha1.MachineType) *appsv1.StatefulSet {
	machineTypeName := machineType.Name
	stsName := GenerateReservationResourceName(testMachineGroup, machineTypeName)
	stsLabels := GenerateReservationResourceLabel(testMachineGroup, machineTypeName)

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Labels: stsLabels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: stsName,
			Replicas:    pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: stsLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: stsLabels,
				},
				Spec: corev1.PodSpec{
					Tolerations: imperatorv1alpha1.GenerateToleration(machineTypeName, testMachineGroup),
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: imperatorv1alpha1.GenerateAffinityMatchExpression(machineType, testMachineGroup),
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						GenerateSleeperContainer(),
					},
				},
			},
		},
	}

	resourceList := corev1.ResourceList{
		corev1.ResourceCPU:    machineType.Spec.CPU,
		corev1.ResourceMemory: machineType.Spec.Memory,
	}
	if machineType.Spec.GPU != nil {
		resourceList[machineType.Spec.GPU.Type] = machineType.Spec.GPU.Num
	}

	sts.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
		Requests: resourceList,
		Limits:   resourceList,
	}

	return sts
}

func newFakeService(machineTypeName string) *corev1.Service {
	svcLabels := GenerateReservationResourceLabel(testMachineGroup, machineTypeName)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels: svcLabels,
		},
		Spec: corev1.ServiceSpec{
			Selector: svcLabels,
			Ports: []corev1.ServicePort{{
				Port: 80,
			}},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}
