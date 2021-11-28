package consts

// label key
const (
	MachineGroupKey    = "imperator.tenzen-y.io/machine-group"
	MachineStatusKey   = "imperator.tenzen-y.io/node-pool"
	MachineTypeKey     = "imperator.tenzen-y.io/machine-type"
	PodRoleKey         = "imperator.tenzen-y.io/pod-role"
	StatefulSetImage   = "alpine:3.15.0"
	NvidiaGPUFamilyKey = "nvidia.com/gpu.family"
	// label value
	KindMachineNodePool = "MachineNodePool"
	KindMachine         = "Machine"
	PodRoleReservation  = "reservation"
	PodRoleGuest        = "guest"
	// controller name
	OwnerControllerField     = ".metadata.ownerReference.controller"
	MachineNodePoolFinalizer = "imperator-machinenodepool-finalizer"
)

var (
	CannotUseNodeTaints = []string{
		"node.kubernetes.io/not-ready",
		"node.kubernetes.io/unschedulable",
		"node.kubernetes.io/network-unavailable",
		"node.kubernetes.io/unreachable",
	}
	ImperatorCoreNamespace = getEnvVarOrDefault("IMPERATOR_CORE_NAMESPACE", "imperator-system")
)
