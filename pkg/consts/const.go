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

package consts

import (
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	appsv1 "k8s.io/api/apps/v1"
)

const (
	MachineGroupKey  = "imperator.tenzen-y.io/machine-group"
	MachineStatusKey = "imperator.tenzen-y.io/node-pool"
	MachineTypeKey   = "imperator.tenzen-y.io/machine-type"
	PodRoleKey       = "imperator.tenzen-y.io/pod-role"

	StatefulSetImage = "alpine:3.15.0"

	NvidiaGPUFamilyKey  = "nvidia.com/gpu.family"
	NvidiaGPUProductKey = "nvidia.com/gpu.product"
	NvidiaGPUMachineKey = "nvidia.com/gpu.machine"

	KindMachineNodePool = "MachineNodePool"
	KindMachine         = "Machine"
	PodRoleReservation  = "reservation"
	PodRoleGuest        = "guest"

	MachineNodePoolFinalizer = "imperator-machinenodepool-finalizer"
	NodeNotReadyTaint        = "node.kubernetes.io/not-ready"
	SuiteTestTimeOut         = time.Second * 5

	ImperatorResourceInjectionKey     = "imperator.tenzen.io/inject-resource"
	ImperatorResourceInjectionEnabled = "enabled"

	ImperatorResourceInjectContainerNameKey = "imperator.tenzen-y.io/injecting-container"
	PodResourceInjectorPath                 = "/mutate-core-v1-pod"
)

var (
	CannotUseNodeTaints = []string{
		"node.kubernetes.io/not-ready",
		"node.kubernetes.io/unschedulable",
		"node.kubernetes.io/network-unavailable",
		"node.kubernetes.io/unreachable",
	}
	ImperatorCoreNamespace = getEnvVarOrDefault("IMPERATOR_CORE_NAMESPACE", "imperator-system")
	CmpSliceOpts           = []cmp.Option{
		cmpopts.SortSlices(func(i, j int) bool {
			return i < j
		}),
	}
	CmpStatefulSetOpts = []cmp.Option{
		cmpopts.IgnoreFields(appsv1.StatefulSetSpec{},
			"Selector", "Template"),
	}
)
