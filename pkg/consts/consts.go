package consts

// label key
const (
	MachineGroupKey  = "imperator.io/machine-group"
	MachineStatusKey = "imperator.io/nodePool"
	MachineName      = "imperator.io/machine"
	K8sAppNameKey    = "app.kubernetes.io/name"
	K8sAppVersionKey = "app.kubernetes.io/version"
)

const (
	K8sAppNameValue       = "imperator"
	KindMachineNodePool   = "MachineNodePool"
	KindMachine           = "Machine"
	MachineStatusReady    = "ready"
	MachineStatusNotReady = "not-ready"
)

const (
	OwnerControllerField = ".metadata.ownerReference.controller"
	ControllerName       = "imperator-machinenodepool-controller"
)
