package utils

import (
	imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
	"strings"
)

func GetMachineGroup(machineLabels map[string]string) string {
	return machineLabels[consts.MachineGroupKey]
}

func GetMachineTypeLabelTaintKey(machineType string) string {
	return strings.Join([]string{imperatorv1alpha1.GroupVersion.Group, machineType}, "/")
}
