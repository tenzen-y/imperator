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
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/tenzen-y/imperator/pkg/consts"
)

func TestGetGPUSelector(t *testing.T) {
	tests := []struct {
		description string
		gpuSpec     *GPUSpec
		expected    []string
	}{
		{
			description: "Use Family",
			gpuSpec:     &GPUSpec{Family: "turing"},
			expected:    []string{consts.NvidiaGPUFamilyKey, "turing"},
		}, {
			description: "Use Product",
			gpuSpec:     &GPUSpec{Product: "NVIDIA-GeForce-RTX-3080"},
			expected:    []string{consts.NvidiaGPUProductKey, "NVIDIA-GeForce-RTX-3080"},
		}, {
			description: "Use Machine",
			gpuSpec:     &GPUSpec{Machine: "DGX-A100"},
			expected:    []string{consts.NvidiaGPUMachineKey, "DGX-A100"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual := getGPUSelector(*test.gpuSpec)
			if diff := cmp.Diff(actual, test.expected); diff != "" {
				t.Fatalf("\ndiff: %v\n; actual and expected are different", diff)
			}
		})
	}
}
