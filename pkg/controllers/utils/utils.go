package utils

import (
	"strings"
)

func GetMachineNodePoolName(machineGroupName string) string {
	return strings.Join([]string{
		machineGroupName,
		"node-pool",
	}, "-")
}
