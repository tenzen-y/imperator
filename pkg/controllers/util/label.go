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
