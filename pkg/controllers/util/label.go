package util

import (
	"strings"

	imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
)

func GetMachineGroup(machineLabels map[string]string) string {
	return machineLabels[consts.MachineGroupKey]
}

func GenerateReservationResourceLabel(machineGroup, machineType string) map[string]string {
	return map[string]string{
		consts.MachineGroupKey: machineGroup,
		consts.MachineTypeKey:  machineType,
		consts.PodRoleKey:      consts.PodRoleReservation,
	}
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
