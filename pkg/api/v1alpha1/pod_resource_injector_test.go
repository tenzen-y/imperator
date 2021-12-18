package v1alpha1

import (
	"context"
	"fmt"
	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tenzen-y/imperator/pkg/api/consts"
	commonconsts "github.com/tenzen-y/imperator/pkg/consts"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	injectedNs    = "inject-ns"
	notInjectedNs = "not-inject-ns"
)

func newFakePod(podName, nsName string, podLabels map[string]string) *corev1.Pod {
	pod := &corev1.Pod{}
	pod.Name = podName
	pod.Namespace = nsName
	pod.Labels = podLabels
	pod.Spec.Containers = []corev1.Container{
		{
			Name:  "test1",
			Image: "test1",
		},
		{
			Name:  "test2",
			Image: "test2",
		},
	}
	return pod
}

func newTestGuestLabels(machineTypeName string) map[string]string {
	return map[string]string{
		commonconsts.MachineGroupKey: testMachineGroup,
		commonconsts.MachineTypeKey:  machineTypeName,
		commonconsts.PodRoleKey:      commonconsts.PodRoleGuest,
	}
}

func checkNoInjection(pod *corev1.Pod) {
	// Check Container Resource
	expectedResource := pod.Spec.Containers[0].Resources
	Eventually(func() string {
		getPod := &corev1.Pod{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, getPod)).NotTo(HaveOccurred())
		return cmp.Diff(getPod.Spec.Containers[0].Resources, expectedResource)
	}, commonconsts.SuiteTestTimeOut).Should(BeEmpty())

	//Check Pod Affinity
	Eventually(func() *corev1.Affinity {
		getPod := &corev1.Pod{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, getPod)).NotTo(HaveOccurred())
		return getPod.Spec.Affinity
	}, commonconsts.SuiteTestTimeOut).Should(BeNil())

	// Check Pod Toleration
	expectedToleration := []corev1.Toleration{
		{
			Key:               "node.kubernetes.io/not-ready",
			Operator:          corev1.TolerationOpExists,
			Effect:            corev1.TaintEffectNoExecute,
			TolerationSeconds: pointer.Int64(300),
		},
		{
			Key:               "node.kubernetes.io/unreachable",
			Operator:          corev1.TolerationOpExists,
			Effect:            corev1.TaintEffectNoExecute,
			TolerationSeconds: pointer.Int64(300),
		},
	}
	Eventually(func() []corev1.Toleration {
		getPod := &corev1.Pod{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, getPod)).NotTo(HaveOccurred())
		return getPod.Spec.Tolerations
	}, commonconsts.SuiteTestTimeOut).Should(ContainElements(expectedToleration))
}

var _ = Describe("Machine Webhook", func() {
	const testMachineTypeName = "test-machine1"

	ctx := context.Background()

	BeforeEach(func() {
		Expect(k8sClient.DeleteAllOf(ctx, &Machine{}, &client.DeleteAllOfOptions{})).NotTo(HaveOccurred())
		for _, nsName := range []string{injectedNs, notInjectedNs} {
			Expect(k8sClient.DeleteAllOf(ctx, &corev1.Pod{}, &client.DeleteAllOfOptions{ListOptions: client.ListOptions{
				Namespace: nsName,
			}})).NotTo(HaveOccurred())
		}
		Expect(k8sClient.DeleteAllOf(ctx, &corev1.Node{}, &client.DeleteAllOfOptions{})).NotTo(HaveOccurred())
		fakeNodes := []string{"test-node1", "test-node2", "test-node3"}
		for _, name := range fakeNodes {
			node := newFakeNode(name)
			Expect(k8sClient.Create(ctx, node, &client.CreateOptions{})).NotTo(HaveOccurred())
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Name: name}, &corev1.Node{})
			}, commonconsts.SuiteTestTimeOut).Should(BeNil())
		}
		Expect(os.Setenv("SKIP_OWNER_CHECK", "true")).NotTo(HaveOccurred())
	})

	It("Inject resources, affinity, and toleration to Pod", func() {
		const injectedPodName = "injected-pod"
		machine := newFakeMachine()
		Expect(k8sClient.Create(ctx, machine, &client.CreateOptions{})).NotTo(HaveOccurred())

		pod := newFakePod(injectedPodName, injectedNs, newTestGuestLabels(testMachineTypeName))
		Expect(k8sClient.Create(ctx, pod, &client.CreateOptions{})).NotTo(HaveOccurred())
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, &corev1.Pod{})).NotTo(HaveOccurred())

		// Check Container Resource
		resource := convertToResourceQuantity(&machine.Spec.MachineTypes[0])
		expectedResource := corev1.ResourceRequirements{
			Requests: resource,
			Limits:   resource,
		}
		Eventually(func() string {
			getPod := &corev1.Pod{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, getPod)).NotTo(HaveOccurred())
			return cmp.Diff(getPod.Spec.Containers[0].Resources, expectedResource)
		}, commonconsts.SuiteTestTimeOut).Should(BeEmpty())

		//Check Pod Affinity
		expectedAffinity := &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: GenerateAffinityMatchExpression(&machine.Spec.MachineTypes[0], testMachineGroup),
						},
					},
				},
			},
		}
		Eventually(func() string {
			getPod := &corev1.Pod{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, getPod)).NotTo(HaveOccurred())
			return cmp.Diff(getPod.Spec.Affinity, expectedAffinity)
		}, commonconsts.SuiteTestTimeOut).Should(BeEmpty())

		// Check Pod Toleration
		expectedToleration := GenerateToleration(testMachineTypeName, testMachineGroup)
		expectedToleration = append(expectedToleration, corev1.Toleration{
			Key:               "node.kubernetes.io/not-ready",
			Operator:          corev1.TolerationOpExists,
			Effect:            corev1.TaintEffectNoExecute,
			TolerationSeconds: pointer.Int64(300),
		})
		expectedToleration = append(expectedToleration, corev1.Toleration{
			Key:               "node.kubernetes.io/unreachable",
			Operator:          corev1.TolerationOpExists,
			Effect:            corev1.TaintEffectNoExecute,
			TolerationSeconds: pointer.Int64(300),
		})

		Eventually(func() []corev1.Toleration {
			getPod := &corev1.Pod{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, getPod)).NotTo(HaveOccurred())
			return getPod.Spec.Tolerations
		}, commonconsts.SuiteTestTimeOut).Should(ContainElements(expectedToleration))

	})

	It(fmt.Sprintf("Skip to inject resources, affinity, and toleration to Pod "+
		"since Pod will be deployed to namespace which does not have a label, <%s=%s>.",
		consts.ImperatorResourceInjectContainerNameKey, consts.ImperatorResourceInjectionEnabled), func() {

		const notInjectedPodName = "not-injected-pod"
		machine := newFakeMachine()
		Expect(k8sClient.Create(ctx, machine, &client.CreateOptions{})).NotTo(HaveOccurred())

		pod := newFakePod(notInjectedPodName, notInjectedNs, newTestGuestLabels(testMachineTypeName))
		Expect(k8sClient.Create(ctx, pod, &client.CreateOptions{})).NotTo(HaveOccurred())
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, &corev1.Pod{})).NotTo(HaveOccurred())

		checkNoInjection(pod)
	})

	It(fmt.Sprintf("Skip to inject resources, affinity, and toleration to Pod "+
		"since Pod does not have necessary labels, <%s>=*, <%s>=*, <%s>=<%s>", commonconsts.MachineGroupKey,
		commonconsts.MachineTypeKey, commonconsts.PodRoleKey, commonconsts.PodRoleGuest), func() {

		const notInjectedPodName = "injected-pod"
		machine := newFakeMachine()
		Expect(k8sClient.Create(ctx, machine, &client.CreateOptions{})).NotTo(HaveOccurred())

		// missing "imperator.tenzen-y.io/machine-group"
		pod := newFakePod(notInjectedPodName, injectedNs, newTestGuestLabels(testMachineTypeName))
		delete(pod.Labels, commonconsts.MachineGroupKey)
		Expect(k8sClient.Create(ctx, pod, &client.CreateOptions{})).NotTo(HaveOccurred())
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, &corev1.Pod{})).NotTo(HaveOccurred())
		checkNoInjection(pod)
		Expect(k8sClient.Delete(ctx, pod, &client.DeleteOptions{})).NotTo(HaveOccurred())

		// missing "imperator.tenzen-y.io/machine-type"
		pod = newFakePod(notInjectedPodName, injectedNs, newTestGuestLabels(testMachineTypeName))
		delete(pod.Labels, commonconsts.MachineTypeKey)
		Expect(k8sClient.Create(ctx, pod, &client.CreateOptions{})).NotTo(HaveOccurred())
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, &corev1.Pod{})).NotTo(HaveOccurred())
		checkNoInjection(pod)
		Expect(k8sClient.Delete(ctx, pod, &client.DeleteOptions{})).NotTo(HaveOccurred())

		// missing "imperator.tenzen-y.io/pod-role=guest"
		pod = newFakePod(notInjectedPodName, injectedNs, newTestGuestLabels(testMachineTypeName))
		delete(pod.Labels, commonconsts.PodRoleKey)
		Expect(k8sClient.Create(ctx, pod, &client.CreateOptions{})).NotTo(HaveOccurred())
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, &corev1.Pod{})).NotTo(HaveOccurred())
		checkNoInjection(pod)
		Expect(k8sClient.Delete(ctx, pod, &client.DeleteOptions{})).NotTo(HaveOccurred())
	})

	//It("Pod will be updated successfully", func() {
	//	const injectedPodName = "injected-pod"
	//	machine := newFakeMachine()
	//	Expect(k8sClient.Create(ctx, machine, &client.CreateOptions{})).NotTo(HaveOccurred())
	//
	//	pod := newFakePod(injectedPodName, injectedNs, newTestGuestLabels(testMachineTypeName))
	//	Expect(k8sClient.Create(ctx, pod, &client.CreateOptions{})).NotTo(HaveOccurred())
	//
	//	getPod := &corev1.Pod{}
	//	Expect(k8sClient.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, getPod)).NotTo(HaveOccurred())
	//	getPod.Labels[commonconsts.MachineTypeKey] = "test-machine2"
	//
	//	Expect(k8sClient.Update(ctx, getPod, &client.UpdateOptions{})).NotTo(HaveOccurred())
	//})

	It("Failed to update Pod", func() {
		const injectedPodName = "injected-pod"
		machine := newFakeMachine()
		Expect(k8sClient.Create(ctx, machine, &client.CreateOptions{})).NotTo(HaveOccurred())

		pod := newFakePod(injectedPodName, injectedNs, newTestGuestLabels(testMachineTypeName))
		Expect(k8sClient.Create(ctx, pod, &client.CreateOptions{})).NotTo(HaveOccurred())

		getPod := &corev1.Pod{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, getPod)).NotTo(HaveOccurred())
		container0 := getPod.Spec.Containers[0]
		getPod.Spec.Containers[0] = getPod.Spec.Containers[1]
		getPod.Spec.Containers[1] = container0

		Expect(k8sClient.Update(ctx, getPod, &client.UpdateOptions{})).ShouldNot(BeNil())
	})

	It("Failed to create Pod", func() {
		testCases := []struct {
			description string
			fakePod     *corev1.Pod
			err         bool
		}{
			{
				description: "There is not Pod namespace",
				fakePod:     newFakePod("missing-namespace-pod", "missing-namespace", newTestGuestLabels(testMachineTypeName)),
				err:         true,
			},
			{
				description: "There is not MachineGroup",
				fakePod: func() *corev1.Pod {
					pod := newFakePod("missing-machine-group", injectedNs, newTestGuestLabels(testMachineTypeName))
					pod.Labels[commonconsts.MachineGroupKey] = "null-machine-group"
					return pod
				}(),
				err: true,
			},
			{
				description: "MachineGroup does not have specified machineType",
				fakePod:     newFakePod("does-not-have-machine-type", injectedNs, newTestGuestLabels("null-machine-type")),
				err:         true,
			},
		}

		fakeMachine := newFakeMachine()
		Expect(k8sClient.Create(ctx, fakeMachine, &client.CreateOptions{})).NotTo(HaveOccurred())

		for _, test := range testCases {
			err := k8sClient.Create(ctx, test.fakePod, &client.CreateOptions{})
			if test.err {
				Expect(err).To(HaveOccurred(), test.description)
			} else {
				Expect(err).NotTo(HaveOccurred(), test.description)
			}
			Expect(k8sClient.DeleteAllOf(ctx, &Machine{}, &client.DeleteAllOfOptions{})).NotTo(HaveOccurred(), test.description)
		}
	})
})
