package utils

import (
	imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"strings"
)

func GetReservationResourceLabel(machineGroup, machineType string) map[string]string {
	return map[string]string{
		consts.MachineGroupKey: machineGroup,
		consts.MachineTypeKey:  machineType,
		consts.PodRoleKey:      consts.PodRoleReservation,
	}
}

func GetReservationResourceName(machineGroup, machineType string) string {
	return strings.Join([]string{
		machineGroup,
		machineType,
	}, "-")
}

func GenerateSleeperContainer() corev1.Container {
	return corev1.Container{
		Name:  "sleeper",
		Image: consts.StatefulSetImage,
		Command: []string{
			"sh",
			"-c",
		},
		Args: []string{
			"sleep",
			"inf",
		},
	}
}

func GenerateStatefulSet(machineType *imperatorv1alpha1.MachineType, machineGroup string, replica int32, sts *appsv1.StatefulSet) {
	machineTypeName := machineType.Name
	svcName := GetReservationResourceName(machineGroup, machineTypeName)
	stsLabels := GetReservationResourceLabel(machineGroup, machineTypeName)
	machineTypeLabelKey := GetMachineTypeLabelTaintKey(machineTypeName)

	sts.Labels = stsLabels
	sts.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: stsLabels,
	}
	sts.Spec.ServiceName = svcName
	sts.Spec.Replicas = pointer.Int32(replica)

	sts.Spec.Template.Labels = stsLabels

	sts.Spec.Template.Spec.Tolerations = []corev1.Toleration{
		{
			Key:      machineTypeLabelKey,
			Effect:   corev1.TaintEffectNoSchedule,
			Operator: corev1.TolerationOpEqual,
			Value:    machineGroup,
		},
		{
			Key:      consts.MachineStatusKey,
			Effect:   corev1.TaintEffectNoSchedule,
			Operator: corev1.TolerationOpEqual,
			Value:    imperatorv1alpha1.NodeModeReady.Value(),
		},
	}

	stsAffinity := &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      machineTypeLabelKey,
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{machineGroup},
							},
							{
								Key:      consts.MachineStatusKey,
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{imperatorv1alpha1.NodeModeReady.Value()},
							},
						},
					},
				},
			},
		},
	}

	if machineType.Spec.GPU != nil {
		gpuAffinity := corev1.NodeSelectorRequirement{
			Key:      consts.NvidiaGPUFamilyKey,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{machineType.Spec.GPU.Generation},
		}
		stsAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions =
			append(stsAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions,
				gpuAffinity)
	}
	sts.Spec.Template.Spec.Affinity = stsAffinity

	resourceList := corev1.ResourceList{
		corev1.ResourceCPU:    machineType.Spec.CPU,
		corev1.ResourceMemory: machineType.Spec.Memory,
	}
	if machineType.Spec.GPU != nil {
		resourceList[machineType.Spec.GPU.Type] = machineType.Spec.GPU.Num
	}
	sts.Spec.Template.Spec.Containers = []corev1.Container{GenerateSleeperContainer()}
	sts.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
		Requests: resourceList,
		Limits:   resourceList,
	}
}

func GenerateService(machineType, machineGroup string, svc *corev1.Service) {
	svcLabels := GetReservationResourceLabel(machineGroup, machineType)

	svc.Labels = svcLabels
	svc.Spec.Selector = svcLabels
	svc.Spec.Ports = []corev1.ServicePort{{
		Port: 80,
	}}
	svc.Spec.Type = corev1.ServiceTypeClusterIP
}
