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

package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/tenzen-y/imperator/pkg/consts"
)

var priLogger = ctrl.Log.WithName("pod-resource-injector")

// +kubebuilder:webhook:path=/mutate-core-v1-pod,matchPolicy=equivalent,mutating=true,failurePolicy=fail,sideEffects=None,groups=core,resources=pods,verbs=create;update,versions=v1,name=mutator.pod.imperator.tenzen-y.io,admissionReviewVersions={v1,v1beta1}

func NewResourceInjector(c client.Client) *resourceInjector {
	return &resourceInjector{
		Client: c,
	}
}

type resourceInjector struct {
	Client  client.Client
	decoder *admission.Decoder
}

func (r *resourceInjector) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}

func (r *resourceInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	if err := r.decoder.Decode(req, pod); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Inject resource to Pod
	if r.requiredInjection(pod) {

		priLogger.Info(fmt.Sprintf("name: <%s>, namespace: <%s>; required injection", pod.Name, pod.Namespace))

		if err := r.replacePods(ctx, pod); err != nil {
			return admission.Denied(err.Error())
		}
		if err := r.injectToPod(ctx, pod); err != nil {
			return admission.Denied(err.Error())
		}
	}

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func (r *resourceInjector) requiredInjection(pod *corev1.Pod) bool {
	// check pod label
	if _, exist := pod.Labels[consts.MachineGroupKey]; !exist {
		return false
	}
	if _, exist := pod.Labels[consts.MachineTypeKey]; !exist {
		return false
	}
	if role := pod.Labels[consts.PodRoleKey]; role != consts.PodRoleGuest {
		return false
	}

	return true
}

func (r *resourceInjector) replacePods(ctx context.Context, pod *corev1.Pod) error {

	// fetch old Pod
	oldPod := &corev1.Pod{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, oldPod); errors.IsNotFound(err) {
		return nil
	}
	return fmt.Errorf("it is forbidden to update the Pod")

	// TODO: Support to replace pod
	//if len(oldPod.OwnerReferences) == 0 && os.Getenv("SKIP_OWNER_CHECK") != "true" {
	//	return fmt.Errorf("name: %s, namespace: %s; pods that are not managed by the parent resource can not be updated",
	//		oldPod.Name, oldPod.Namespace)
	//}
	//
	//if !oldPod.GetDeletionTimestamp().IsZero() {
	//	return fmt.Errorf("name: %s, namespace: %s; the pod is replacing now", pod.Name, pod.Namespace)
	//}
	//
	//if err := r.Client.Delete(ctx, oldPod, &client.DeleteOptions{}); err != nil {
	//	return err
	//}
	//
	//return nil
}

func (r *resourceInjector) injectToPod(ctx context.Context, pod *corev1.Pod) error {
	injectingTargetContainerIdx := findInjectingTargetContainerIndex(pod)
	machineGroup := pod.Labels[consts.MachineGroupKey]
	machineTypeName := pod.Labels[consts.MachineTypeKey]

	targetMachineType, machineTypeUsage, err := r.findMachineType(ctx, machineGroup, machineTypeName)
	if err != nil {
		return err
	} else if targetMachineType == nil {
		return fmt.Errorf("machine-group, <%s> does not have machine-type, <%s>", machineGroup, machineTypeName)
	} else if machineTypeUsage == nil {
		return fmt.Errorf("imperator controller is preparing")
	}

	if machineTypeUsage.Reserved == 0 {
		return fmt.Errorf("name: <%s>, namespace: <%s>; there is no <%s> left", pod.Name, pod.Namespace, targetMachineType.Name)
	}

	// inject resources
	injectResource(targetMachineType, pod, injectingTargetContainerIdx)

	// inject Affinity
	requiredMatchExpressions := GenerateAffinityMatchExpression(targetMachineType, machineGroup)
	injectPodAffinity(pod, requiredMatchExpressions)

	// inject Toleration
	toleration := GenerateToleration(machineTypeName, machineGroup)
	injectPodToleration(pod, toleration)

	return nil
}

func (r *resourceInjector) findMachineType(ctx context.Context, machineGroup, machineTypeName string) (*MachineType, *UsageCondition, error) {
	machines := &MachineList{}
	if err := r.Client.List(ctx, machines, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			consts.MachineGroupKey: machineGroup,
		}),
	}); err != nil {
		return nil, nil, err
	}
	if len(machines.Items) == 0 {
		return nil, nil, fmt.Errorf("failed to find machine-group <%s>", machineGroup)
	}

	machineTypes := machines.Items[0].Spec.MachineTypes
	var targetMachineType *MachineType
	for _, mt := range machineTypes {
		if mt.Name == machineTypeName {
			targetMachineType = &mt
			break
		}
	}

	var targetMachineStatus *UsageCondition
	for _, mtStatus := range machines.Items[0].Status.AvailableMachines {
		if mtStatus.Name == machineTypeName {
			targetMachineStatus = &mtStatus.Usage
			break
		}
	}

	return targetMachineType, targetMachineStatus, nil
}

func findInjectingTargetContainerIndex(pod *corev1.Pod) int {
	if containerName, exist := pod.Labels[consts.ImperatorResourceInjectContainerNameKey]; exist {
		for idx, c := range pod.Spec.Containers {
			if c.Name != containerName {
				continue
			}
			return idx
		}
	}
	return 0
}

func injectResource(machineType *MachineType, pod *corev1.Pod, containerIdx int) {
	resourceList := convertToResourceQuantity(machineType)
	origin := pod.Spec.Containers[containerIdx].Resources.DeepCopy()

	// inject resources
	pod.Spec.Containers[containerIdx].Resources = corev1.ResourceRequirements{
		Requests: resourceList,
		Limits:   resourceList,
	}

	if diff := cmp.Diff(origin, pod.Spec.Containers[containerIdx].Resources); diff != "" {
		priLogger.Info(fmt.Sprintf("Injected resources; Name: <%s>, Namespace: <%s>", pod.Name, pod.Namespace))
		priLogger.Info(diff)
	}
}

func convertToResourceQuantity(machineType *MachineType) corev1.ResourceList {
	resourceList := corev1.ResourceList{
		corev1.ResourceCPU:    machineType.Spec.CPU,
		corev1.ResourceMemory: machineType.Spec.Memory,
	}
	if machineType.Spec.GPU != nil {
		resourceList[machineType.Spec.GPU.Type] = machineType.Spec.GPU.Num
	}
	return resourceList
}

func injectPodAffinity(pod *corev1.Pod, requiredMatchExpressions []corev1.NodeSelectorRequirement) {

	// create key-value map to inject
	injectedMExpressionKeys := make(map[string]string)
	for _, injectedMExpression := range requiredMatchExpressions {
		injectedMExpressionKeys[injectedMExpression.Key] = injectedMExpression.Values[0]
	}

	if pod.Spec.Affinity == nil {

		pod.Spec.Affinity = &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{},
				},
			},
		}

	} else if pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {

		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{},
		}

	}

	origin := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.DeepCopy()

	if len(pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) != 0 {

		// remove matchExpression which has duplicated key
		for nsIdx, nsTerm := range pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
			for meIdx, mExpression := range nsTerm.MatchExpressions {
				if injectedMExpressionKeys[mExpression.Key] != mExpression.Values[0] {
					continue
				}

				matchExpressionsNum := len(nsTerm.MatchExpressions)
				if meIdx == matchExpressionsNum-1 {
					pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[nsIdx].MatchExpressions =
						nsTerm.MatchExpressions[:matchExpressionsNum-1]
				} else {
					pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[nsIdx].MatchExpressions =
						append(nsTerm.MatchExpressions[:meIdx], nsTerm.MatchExpressions[meIdx+1:]...)
				}
			}
		}

	}

	pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms =
		append(
			pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms,
			corev1.NodeSelectorTerm{MatchExpressions: requiredMatchExpressions},
		)

	if diff := cmp.Diff(origin, pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution); diff != "" {
		priLogger.Info(fmt.Sprintf("Injected Pod Affinity; Name: <%s>, Namespace: <%s>", pod.Name, pod.Namespace))
		priLogger.Info(diff)
	}
}

func injectPodToleration(pod *corev1.Pod, toleration []corev1.Toleration) {
	origin := pod.Spec.DeepCopy()

	for _, t := range toleration {
		if len(pod.Spec.Tolerations) != 0 {
			duplicatedIdx := findToleration(pod.Spec.Tolerations, t.Key, t.Value)
			// remove toleration which has duplicated key
			if duplicatedIdx != nil {

				tolerationNum := len(pod.Spec.Tolerations)
				if *duplicatedIdx == tolerationNum-1 {
					pod.Spec.Tolerations = pod.Spec.Tolerations[:tolerationNum-1]
				} else {
					pod.Spec.Tolerations = append(pod.Spec.Tolerations[:*duplicatedIdx], pod.Spec.Tolerations[*duplicatedIdx+1:]...)
				}
			}
		}

		// inject toleration
		pod.Spec.Tolerations = append(pod.Spec.Tolerations, t)
	}

	if diff := cmp.Diff(origin.Tolerations, pod.Spec.Tolerations); diff != "" {
		priLogger.Info(fmt.Sprintf("Injected Pod Toleration; Name <%s>, Namespace: <%s>", pod.Name, pod.Namespace))
		priLogger.Info(diff)
	}
}

func findToleration(toleration []corev1.Toleration, tolerationKey, tolerationValue string) *int {
	for tIdx, t := range toleration {
		if t.Key == tolerationKey && t.Value == tolerationValue {
			return pointer.Int(tIdx)
		}
	}
	return nil
}
