package consts

// label key
const (
	MachineGroupKey  = "imperator.tenzen-y.io/machine-group"
	MachineStatusKey = "imperator.tenzen-y.io/node-pool"
	MachineName      = "imperator.tenzen-y.io/machine"
	K8sAppNameKey    = "app.kubernetes.io/name"
	K8sAppVersionKey = "app.kubernetes.io/version"
	// label value
	K8sAppNameValue          = "imperator"
	KindMachineNodePool      = "MachineNodePool"
	KindMachine              = "Machine"
	MachineStatusReady       = "ready"
	MachineStatusNotReady    = "not-ready"
	MachineStatusMaintenance = "maintenance"
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
)
