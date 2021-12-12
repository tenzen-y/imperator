package utils

import (
	imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
	"strings"
)

func GetMachineGroup(machineLabels map[string]string) string {
	return machineLabels[consts.MachineGroupKey]
}

func GetScheduleMachineTypeKeys(machineTypes []imperatorv1alpha1.NodePoolMachineType) []string {
	var machineTypeKeys []string
	for _, mt := range machineTypes {
		machineTypeKeys = append(machineTypeKeys, strings.Join([]string{
			imperatorv1alpha1.GroupVersion.Group,
			mt.Name,
		}, "/"))
	}
	return machineTypeKeys
}

func GetMachineTypeLabelTaintKey(machineTypeName string) string {
	return strings.Join([]string{imperatorv1alpha1.GroupVersion.Group, machineTypeName}, "/")
}
