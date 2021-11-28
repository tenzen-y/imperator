package utils

import imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"

func GetMachineTypeUsage(availableMachine []imperatorv1alpha1.AvailableMachineCondition, machineType string) *imperatorv1alpha1.UsageCondition {
	for _, cond := range availableMachine {
		if cond.Name != machineType {
			continue
		}
		return &cond.Usage
	}
	return nil
}
