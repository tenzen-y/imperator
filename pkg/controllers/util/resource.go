package util

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
)

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
	svcName := GenerateReservationResourceName(machineGroup, machineTypeName)
	stsLabels := GenerateReservationResourceLabel(machineGroup, machineTypeName)

	sts.Labels = stsLabels
	sts.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: stsLabels,
	}
	sts.Spec.ServiceName = svcName
	sts.Spec.Replicas = pointer.Int32(replica)

	sts.Spec.Template.Labels = stsLabels

	sts.Spec.Template.Spec.Tolerations = imperatorv1alpha1.GenerateToleration(machineTypeName, machineGroup)

	sts.Spec.Template.Spec.Affinity = &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: imperatorv1alpha1.GenerateAffinityMatchExpression(machineType, machineGroup),
					},
				},
			},
		},
	}

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
	svcLabels := GenerateReservationResourceLabel(machineGroup, machineType)

	svc.Labels = svcLabels
	svc.Spec.Selector = svcLabels
	svc.Spec.Ports = []corev1.ServicePort{{
		Port: 80,
	}}
	svc.Spec.Type = corev1.ServiceTypeClusterIP
}
