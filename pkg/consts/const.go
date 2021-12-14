package consts

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	appsv1 "k8s.io/api/apps/v1"
	"time"
)

const (
	MachineGroupKey    = "imperator.tenzen-y.io/machine-group"
	MachineStatusKey   = "imperator.tenzen-y.io/node-pool"
	MachineTypeKey     = "imperator.tenzen-y.io/machine-type"
	PodRoleKey         = "imperator.tenzen-y.io/pod-role"
	StatefulSetImage   = "alpine:3.15.0"
	NvidiaGPUFamilyKey = "nvidia.com/gpu.family"

	KindMachineNodePool = "MachineNodePool"
	KindMachine         = "Machine"
	PodRoleReservation  = "reservation"
	PodRoleGuest        = "guest"

	MachineNodePoolFinalizer = "imperator-machinenodepool-finalizer"
	NodeNotReadyTaint        = "node.kubernetes.io/not-ready"
	SuiteTestTimeOut         = time.Second * 5
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
