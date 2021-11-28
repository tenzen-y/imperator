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

func getReservationResourceLabel(machineGroup, machineType string) map[string]string {
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
	}, ".")
}

func GenerateStatefulSet(machineType *imperatorv1alpha1.MachineType, machineGroup string, replica int32) *appsv1.StatefulSet {
	machineTypeName := machineType.Name
	stsName := GetReservationResourceName(machineGroup, machineTypeName)
	stsLabels := getReservationResourceLabel(machineGroup, machineTypeName)
	machineTypeLabelKey := GetMachineTypeLabelTaintKey(machineTypeName)

	resourceList := corev1.ResourceList{
		corev1.ResourceCPU:    machineType.Spec.CPU,
		corev1.ResourceMemory: machineType.Spec.Memory,
	}
	if machineType.Spec.GPU != nil {
		resourceList[machineType.Spec.GPU.Type] = machineType.Spec.GPU.Num
	}

	sts := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      stsName,
			Namespace: consts.ImperatorCoreNamespace,
			Labels:    stsLabels,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: stsLabels,
			},
			ServiceName: stsName,
			Replicas:    pointer.Int32(replica),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: stsLabels,
				},
				Spec: corev1.PodSpec{
					Tolerations: []corev1.Toleration{
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
					},
					Affinity: &corev1.Affinity{
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
					},
					TerminationGracePeriodSeconds: pointer.Int64(10),
					Containers: []corev1.Container{{
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
						Resources: corev1.ResourceRequirements{
							Requests: resourceList,
							Limits:   resourceList,
						},
					}},
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
		sts.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.
			NodeSelectorTerms[0].MatchExpressions[2] = gpuAffinity
	}

	return sts
}

func GenerateService(machineType, machineGroup string) *corev1.Service {
	svcName := GetReservationResourceName(machineGroup, machineType)
	svcLabels := getReservationResourceLabel(machineGroup, machineType)
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: consts.ImperatorCoreNamespace,
			Labels:    svcLabels,
		},
		Spec: corev1.ServiceSpec{
			Selector: svcLabels,
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
}
