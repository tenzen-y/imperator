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
	"github.com/google/go-cmp/cmp"
	imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"strings"
	"testing"
)

func TestExtractKeyValueFromTaint(t *testing.T) {
	testCases := []struct {
		description string
		taints      []corev1.Taint
		expected    map[string]string
	}{
		{
			description: "Positive test",
			taints:      newFakeTaints(),
			expected: map[string]string{
				strings.Join([]string{imperatorv1alpha1.GroupVersion.Group, "compute-xlarge"}, "/"): "test-machine-type",
				consts.MachineStatusKey: imperatorv1alpha1.NodeModeReady.Value(),
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			actual := ExtractKeyValueFromTaint(test.taints)
			if diff := cmp.Diff(actual, test.expected); diff != "" {
				t.Errorf("DIFF: \n%v\n", diff)
			}
		})
	}
}

func TestGetTaintKeyIndex(t *testing.T) {
	testCases := []struct {
		description string
		taints      []corev1.Taint
		key         string
		expected    *int
	}{
		{
			description: "Positive test",
			taints:      newFakeTaints(),
			key:         consts.MachineStatusKey,
			expected:    pointer.Int(1),
		},
		{
			description: "There is not specified key in taints",
			taints:      newFakeTaints(),
			key:         "not-exist",
			expected:    nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			actual := GetTaintKeyIndex(test.taints, test.key)
			if test.expected == nil {
				if actual != nil {
					t.Errorf("GOT: \n%v\n, WANT: \n%v\n", actual, nil)
				}
			} else {
				if *actual != *test.expected {
					t.Errorf("GOT: \n%v\n, WANT: \n%v\n", *actual, *test.expected)
				}
			}
		})
	}
}

func newFakeTaints() []corev1.Taint {
	now := metav1.Now()
	return []corev1.Taint{
		{
			Key:       strings.Join([]string{imperatorv1alpha1.GroupVersion.Group, "compute-xlarge"}, "/"),
			Value:     "test-machine-type",
			Effect:    corev1.TaintEffectNoSchedule,
			TimeAdded: &now,
		},
		{
			Key:       consts.MachineStatusKey,
			Value:     imperatorv1alpha1.NodeModeReady.Value(),
			Effect:    corev1.TaintEffectNoSchedule,
			TimeAdded: &now,
		},
	}
}
