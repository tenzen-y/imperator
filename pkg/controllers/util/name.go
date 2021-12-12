package util

import "strings"

func GenerateMachineNodePoolName(machineGroupName string) string {
	return strings.Join([]string{
		machineGroupName,
		"node-pool",
	}, "-")
}

func GenerateReservationResourceName(machineGroup, machineType string) string {
	return strings.Join([]string{
		machineGroup,
		machineType,
	}, "-")
}
