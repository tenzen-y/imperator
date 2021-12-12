package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/tenzen-y/imperator/pkg/api/consts"
	commonconsts "github.com/tenzen-y/imperator/pkg/consts"
)

//+kubebuilder:webhook:path=/mutate-imperator-pod-resource,mutating=true,failurePolicy=fail,sideEffects=None,groups="",resources=pods,verbs=create;update,versions=v1alpha1,name=mmachinepod.kb.io,admissionReviewVersions={v1,v1beta1}
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list

func NewResourceInjector(c client.Client) *resourceInjector {
	return &resourceInjector{
		client: c,
	}
}

type resourceInjector struct {
	client  client.Client
	decoder *admission.Decoder
}

func (r *resourceInjector) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}

func (r *resourceInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	if err := r.decoder.Decode(req, pod); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	requiredInjection, err := r.requiredInjection(ctx, pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Inject resource to Pod
	if requiredInjection {
		if err = r.injectContainerResource(ctx, pod); err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
	}

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func (r *resourceInjector) requiredInjection(ctx context.Context, pod *corev1.Pod) (bool, error) {
	// check namespace label
	nsName := pod.Namespace
	ns := &corev1.Namespace{}
	if err := r.client.Get(ctx, client.ObjectKey{Name: nsName}, ns); err != nil {
		return false, err
	}
	if injectRequired := ns.Labels[consts.ImperatorResourceInjectionKey]; injectRequired != consts.ImperatorResourceInjectionEnabled {
		return false, nil
	}

	// check pod label
	if _, exist := pod.Labels[commonconsts.MachineGroupKey]; !exist {
		return false, nil
	}
	if _, exist := pod.Labels[commonconsts.MachineTypeKey]; !exist {
		return false, nil
	}
	if role := pod.Labels[commonconsts.PodRoleKey]; role != commonconsts.PodRoleGuest {
		return false, nil
	}

	return true, nil
}

func (r *resourceInjector) injectContainerResource(ctx context.Context, pod *corev1.Pod) error {
	var injectTargetContainerIdx *int
	if containerName, exist := pod.Labels[consts.ImperatorResourceInjectContainerNameKey]; exist {
		if injectTargetContainerIdx = findContainerIndex(pod, containerName); injectTargetContainerIdx == nil {
			injectTargetContainerIdx = pointer.Int(0)
		}
	}
	if injectTargetContainerIdx == nil {
		injectTargetContainerIdx = pointer.Int(0)
	}

	machineGroup := pod.Labels[commonconsts.MachineGroupKey]
	machineTypeName := pod.Labels[commonconsts.MachineTypeKey]

	machines := &MachineList{}
	if err := r.client.List(ctx, machines, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			commonconsts.MachineGroupKey: machineGroup,
		}),
	}); err != nil {
		return err
	}
	if len(machines.Items) == 0 {
		return fmt.Errorf("failed to find machine-group; <%s>", machineGroup)
	}

	machineTypes := machines.Items[0].Spec.MachineTypes
	targetMachineIdx := pointer.Int(0)
	for idx, mt := range machineTypes {
		if mt.Name != machineTypeName {
			continue
		}
		targetMachineIdx = &idx
	}
	if targetMachineIdx == nil {
		return fmt.Errorf("machine-group, <%s> does not have machine-type, <%s>", machineGroup, machineTypeName)
	}

	setResource(machineTypes[*targetMachineIdx], pod, *injectTargetContainerIdx)

	requiredMatchExpressions := GenerateAffinityMatchExpression(&machineTypes[*targetMachineIdx], machineGroup)
	setPodAffinity(pod, requiredMatchExpressions)

	toleration := GenerateToleration(machineTypeName, machineGroup)
	setPodToleration(pod, toleration)

	return nil
}

func findContainerIndex(pod *corev1.Pod, containerName string) *int {
	for idx, c := range pod.Spec.Containers {
		if c.Name == containerName {
			return &idx
		}
	}
	return nil
}

func setResource(machineType MachineType, pod *corev1.Pod, containerIdx int) {
	resourceList := corev1.ResourceList{
		corev1.ResourceCPU:    machineType.Spec.CPU,
		corev1.ResourceMemory: machineType.Spec.Memory,
	}
	if machineType.Spec.GPU != nil {
		resourceList[machineType.Spec.GPU.Type] = machineType.Spec.GPU.Num
	}

	// inject resources
	pod.Spec.Containers[containerIdx].Resources = corev1.ResourceRequirements{
		Requests: resourceList,
		Limits:   resourceList,
	}

}

func setPodAffinity(pod *corev1.Pod, requiredMatchExpressions []corev1.NodeSelectorRequirement) {
	for _, mExpression := range requiredMatchExpressions {
		matchIdx := findAffinityRequiredMatchExpressionsIdx(pod.Spec.Affinity.NodeAffinity.
			RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, mExpression.Key)
		if matchIdx == nil {
			continue
		}
		// remove matchExpression which has duplicated key
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.
			NodeSelectorTerms[matchIdx[0]].MatchExpressions =
			append(
				pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.
					NodeSelectorTerms[matchIdx[0]].MatchExpressions[:matchIdx[1]],
				pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.
					NodeSelectorTerms[matchIdx[0]].MatchExpressions[matchIdx[1]+1:]...,
			)
	}

	// inject affinity
	pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms =
		append(
			pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms,
			corev1.NodeSelectorTerm{MatchExpressions: requiredMatchExpressions},
		)
}

func findAffinityRequiredMatchExpressionsIdx(nodeSelectorTerms []corev1.NodeSelectorTerm, matchExpressionKey string) []int {
	for nsIdx, nsTerm := range nodeSelectorTerms {
		for mIdx, mExpression := range nsTerm.MatchExpressions {
			if mExpression.Key == matchExpressionKey {
				return []int{nsIdx, mIdx}
			}
		}
	}
	return nil
}

func setPodToleration(pod *corev1.Pod, toleration []corev1.Toleration) {
	podToleration := pod.Spec.Tolerations
	for _, t := range toleration {
		duplicatedIdx := findToleration(podToleration, t.Key)
		if duplicatedIdx == nil {
			continue
		}
		// remove toleration which has duplicated key
		podToleration = append(podToleration[:*duplicatedIdx], podToleration[*duplicatedIdx+1:]...)

		// inject toleration
		podToleration = append(podToleration, t)
	}
}

func findToleration(toleration []corev1.Toleration, tolerationKey string) *int {
	for tIdx, t := range toleration {
		if t.Key != tolerationKey {
			continue
		}
		return pointer.Int(tIdx)
	}
	return nil
}
