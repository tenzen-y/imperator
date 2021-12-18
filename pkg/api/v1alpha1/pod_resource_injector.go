package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/tenzen-y/imperator/pkg/api/consts"
	commonconsts "github.com/tenzen-y/imperator/pkg/consts"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var priLogger = ctrl.Log.WithName("pod-resource-injector")

//+kubebuilder:webhook:path=/mutate-core-v1-pod,mutating=true,failurePolicy=fail,sideEffects=None,groups=core,resources=pods,verbs=create;update,versions=v1,name=mutator.pod.imperator.tenzen-y.io,admissionReviewVersions={v1,v1beta1}
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update;patch;delete

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
		return admission.Errored(http.StatusBadRequest, err)
	}

	requiredInjection, err := r.requiredInjection(ctx, pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Inject resource to Pod
	if requiredInjection {
		if err := r.replacePods(ctx, pod); err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
		if err = r.injectToPod(ctx, pod); err != nil {
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
	if err := r.Client.Get(ctx, client.ObjectKey{Name: nsName}, ns); err != nil {
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

func (r *resourceInjector) replacePods(ctx context.Context, pod *corev1.Pod) error {

	// fetch old Pod
	oldPod := &corev1.Pod{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, oldPod); errors.IsNotFound(err) {
		return nil
	}
	return fmt.Errorf("it is forbidden to update the Pod")

	// TODO: Implementing replace pod
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
	machineGroup := pod.Labels[commonconsts.MachineGroupKey]
	machineTypeName := pod.Labels[commonconsts.MachineTypeKey]

	targetMachineType, err := r.findMachineType(ctx, machineGroup, machineTypeName)
	if err != nil {
		return err
	} else if targetMachineType == nil {
		return fmt.Errorf("machine-group, <%s> does not have machine-type, <%s>", machineGroup, machineTypeName)
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

func (r *resourceInjector) findMachineType(ctx context.Context, machineGroup, machineTypeName string) (*MachineType, error) {
	machines := &MachineList{}
	if err := r.Client.List(ctx, machines, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			commonconsts.MachineGroupKey: machineGroup,
		}),
	}); err != nil {
		return nil, err
	}
	if len(machines.Items) == 0 {
		return nil, fmt.Errorf("failed to find machine-group <%s>", machineGroup)
	}

	machineTypes := machines.Items[0].Spec.MachineTypes
	for _, mt := range machineTypes {
		if mt.Name != machineTypeName {
			continue
		}
		return &mt, nil
	}

	return nil, nil
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
