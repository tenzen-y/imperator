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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
)

func TestGetMachineGroup(t *testing.T) {

	testCases := []struct {
		description   string
		machineLabels map[string]string
		expected      string
	}{
		{
			description: "Positive test",
			machineLabels: map[string]string{
				consts.MachineGroupKey: "test-machine-group",
				"dummy-key":            "dummy-value",
			},
			expected: "test-machine-group",
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			actual := GetMachineGroup(test.machineLabels)
			if !strings.EqualFold(actual, test.expected) {
				t.Errorf("WANT: \n%v\n\n, GOT: \n%v\n\n", test.expected, actual)
			}
		})
	}

}

func TestGetScheduleMachineTypeKeys(t *testing.T) {

	fakeMachineTypes := []string{"compute-xsmall", "compute-large"}

	testCases := []struct {
		description  string
		machineTypes []imperatorv1alpha1.NodePoolMachineType
		expected     []string
	}{
		{
			description: "Positive test",
			machineTypes: func() []imperatorv1alpha1.NodePoolMachineType {
				var machineTypes []imperatorv1alpha1.NodePoolMachineType
				for _, mt := range fakeMachineTypes {
					machineTypes = append(machineTypes, imperatorv1alpha1.NodePoolMachineType{Name: mt})
				}
				return machineTypes
			}(),
			expected: func() []string {
				var expected []string
				for _, mt := range fakeMachineTypes {
					expected = append(expected, imperatorv1alpha1.GroupVersion.Group+"/"+mt)
				}
				return expected
			}(),
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			actual := GetScheduleMachineTypeKeys(test.machineTypes)
			if diff := cmp.Diff(actual, test.expected); diff != "" {
				t.Errorf("DIFF: \n%v\n", diff)
			}
		})
	}

}
