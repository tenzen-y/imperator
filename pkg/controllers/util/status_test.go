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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
)

func TestGetMachineTypeUsage(t *testing.T) {

	testCases := []struct {
		description     string
		machineTypeName string
		expected        *imperatorv1alpha1.UsageCondition
	}{
		{
			description:     "Positive test",
			machineTypeName: "test-machine-type",
			expected: &imperatorv1alpha1.UsageCondition{
				Maximum:  2,
				Used:     1,
				Reserved: 1,
				Waiting:  0,
			},
		},
		{
			description:     "There is not specified machineType in availableMachineConditions",
			machineTypeName: "not-exist",
			expected:        nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			var fakeAvailableMachines []imperatorv1alpha1.AvailableMachineCondition
			if test.expected != nil {
				fakeAvailableMachines = newFakeAvailableMachineConditions(&imperatorv1alpha1.AvailableMachineCondition{
					Name:  test.machineTypeName,
					Usage: *test.expected,
				})
			} else {
				fakeAvailableMachines = newFakeAvailableMachineConditions(nil)
			}
			actual := GetMachineTypeUsage(fakeAvailableMachines, test.machineTypeName)
			if diff := cmp.Diff(actual, test.expected, consts.CmpSliceOpts...); diff != "" {
				t.Errorf("DIFF: \n%v\n", diff)
			}
		})
	}
}

func TestGetPodConditionTypeMap(t *testing.T) {
	now := metav1.Now()

	testCases := []struct {
		description   string
		podConditions []corev1.PodCondition
		expected      map[corev1.PodConditionType]corev1.PodCondition
	}{
		{
			description:   "Positive test",
			podConditions: newFakePodConditions(now),
			expected:      newFakeMapPodConditions(now),
		},
		{
			description:   "Input podConditions is empty ",
			podConditions: make([]corev1.PodCondition, 0),
			expected:      make(map[corev1.PodConditionType]corev1.PodCondition),
		},
		{
			description: "Duplicated Type in PodConditions",
			podConditions: func() []corev1.PodCondition {
				fake := newFakePodConditions(now)
				fake = append(fake, corev1.PodCondition{
					Type:               corev1.ContainersReady,
					LastTransitionTime: now,
					Status:             corev1.ConditionTrue,
				})
				return fake
			}(),
			expected: newFakeMapPodConditions(now),
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			actual := GetPodConditionTypeMap(test.podConditions)
			if diff := cmp.Diff(actual, test.expected); diff != "" {
				t.Errorf("DIFF: \n%v\n", diff)
			}
		})
	}
}

func newFakeAvailableMachineConditions(target *imperatorv1alpha1.AvailableMachineCondition) []imperatorv1alpha1.AvailableMachineCondition {
	availableMachineConditions := []imperatorv1alpha1.AvailableMachineCondition{
		{
			Name: "dummy-machine-type",
			Usage: imperatorv1alpha1.UsageCondition{
				Maximum:  2,
				Reserved: 1,
				Used:     1,
				Waiting:  0,
			},
		},
	}
	if target != nil {
		availableMachineConditions = append(availableMachineConditions, *target)
	}
	return availableMachineConditions
}

func newFakePodConditions(now metav1.Time) []corev1.PodCondition {
	return []corev1.PodCondition{
		{
			Type:               corev1.ContainersReady,
			LastTransitionTime: now,
			Status:             corev1.ConditionTrue,
		},
		{
			Type:               corev1.PodScheduled,
			LastTransitionTime: now,
			Status:             corev1.ConditionFalse,
			Reason:             corev1.PodReasonUnschedulable,
		},
	}
}

func newFakeMapPodConditions(now metav1.Time) map[corev1.PodConditionType]corev1.PodCondition {
	return map[corev1.PodConditionType]corev1.PodCondition{
		corev1.ContainersReady: {
			Type:               corev1.ContainersReady,
			LastTransitionTime: now,
			Status:             corev1.ConditionTrue,
		},
		corev1.PodScheduled: {
			Type:               corev1.PodScheduled,
			LastTransitionTime: now,
			Status:             corev1.ConditionFalse,
			Reason:             corev1.PodReasonUnschedulable,
		},
	}
}
