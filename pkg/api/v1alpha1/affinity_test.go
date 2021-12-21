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
