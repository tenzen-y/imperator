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
)

func TestGenerateMachineNodePoolName(t *testing.T) {
	testCases := []struct {
		description  string
		machineGroup string
		expected     string
	}{
		{
			description:  "Positive test",
			machineGroup: "test-machine-group",
			expected:     "test-machine-group" + "-" + "node-pool",
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			actual := GenerateMachineNodePoolName(test.machineGroup)
			if !strings.EqualFold(actual, test.expected) {
				t.Errorf("WANT: \n%v\n, GOT: \n%v\n", test.expected, actual)
			}
		})
	}
}
