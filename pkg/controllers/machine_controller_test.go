package controllers

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
	"github.com/tenzen-y/imperator/pkg/controllers/util"
)

const (
	testNode1                   = "test-node1"
	testNode2                   = "test-node2"
	testMachine1                = "test-machine1"
	testMachine2                = "test-machine2"
	testMachineMachineGroupName = "test-machine-machine-group"
	testGuestNs                 = "test-guest-ns"
)

func waitStartedReservationResource(ctx context.Context, machineType imperatorv1alpha1.MachineType, stsReplicas int32) {
	stsName := util.GenerateReservationResourceName(testMachineMachineGroupName, machineType.Name)
	Eventually(func() error {
		return k8sClient.Get(ctx, client.ObjectKey{Name: stsName, Namespace: consts.ImperatorCoreNamespace}, &appsv1.StatefulSet{})
	}, consts.SuiteTestTimeOut).Should(BeNil())

	testDescription := fmt.Sprintf("check Replicas of StatefulSet for machineType: <%s>", machineType.Name)
	Eventually(func() int32 {
		sts := &appsv1.StatefulSet{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: stsName, Namespace: consts.ImperatorCoreNamespace}, sts)).NotTo(HaveOccurred())
		return *sts.Spec.Replicas
	}, consts.SuiteTestTimeOut).Should(Equal(stsReplicas), testDescription)

	svc := &corev1.Service{}
	svcName := util.GenerateReservationResourceName(testMachineMachineGroupName, machineType.Name)
	Eventually(func() error {
		return k8sClient.Get(ctx, client.ObjectKey{Name: svcName, Namespace: consts.ImperatorCoreNamespace}, svc)
	}, consts.SuiteTestTimeOut).Should(BeNil())
}

func checkMachineAvailableStatus(ctx context.Context, targetName string, expected gstruct.Fields) {
	testDescription := fmt.Sprintf("check machine available status for machineType: <%s>", targetName)

	Eventually(func() imperatorv1alpha1.UsageCondition {
		machine := &imperatorv1alpha1.Machine{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: testMachineMachineGroupName}, machine)).NotTo(HaveOccurred())
		for _, am := range machine.Status.AvailableMachines {
			if am.Name != targetName {
				continue
			}
			return am.Usage
		}
		return imperatorv1alpha1.UsageCondition{}
	}, consts.SuiteTestTimeOut).Should(gstruct.MatchAllFields(expected), testDescription)
}

func updatePodContainerStatus(ctx context.Context, objKey client.ObjectKey, status string) {
	// Update Pod Status
	pod := &corev1.Pod{}
	Expect(k8sClient.Get(ctx, objKey, pod)).NotTo(HaveOccurred())

	now := metav1.Now()

	switch status {
	case "running":
		pod.Status.Phase = corev1.PodRunning
		pod.Status.Conditions = []corev1.PodCondition{{
			Type:               corev1.ContainersReady,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: now,
		}}
	case "creating":
		pod.Status.Phase = corev1.PodPending
		pod.Status.Conditions = []corev1.PodCondition{{
			Type:               corev1.ContainersReady,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: now,
		}}
	case "pending":
		pod.Status.Phase = corev1.PodPending
		pod.Status.Conditions = []corev1.PodCondition{{
			Type:               corev1.PodScheduled,
			Status:             corev1.ConditionFalse,
			Reason:             corev1.PodReasonUnschedulable,
			LastTransitionTime: now,
		}}
		pod.Status.Conditions = append(pod.Status.Conditions, corev1.PodCondition{
			Type:               corev1.ContainersReady,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: now,
		})
	}

	Eventually(func() error {
		return k8sClient.Status().Update(ctx, pod, &client.UpdateOptions{})
	}, consts.SuiteTestTimeOut).Should(BeNil())

}

func getPodTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: corev1.SchemeGroupVersion.String(),
		Kind:       "Pod",
	}
}

func newFakeMachine(testNodePool map[string]imperatorv1alpha1.NodePool, testMachineTypes map[string]imperatorv1alpha1.MachineType) *imperatorv1alpha1.Machine {
	machine := &imperatorv1alpha1.Machine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: imperatorv1alpha1.GroupVersion.String(),
			Kind:       consts.KindMachine,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: testMachineMachineGroupName,
			Labels: map[string]string{
				consts.MachineGroupKey: testMachineMachineGroupName,
			},
		},
		Spec: imperatorv1alpha1.MachineSpec{},
	}

	var nodePools []imperatorv1alpha1.NodePool
	for _, p := range testNodePool {
		nodePools = append(nodePools, p)
	}
	machine.Spec.NodePool = nodePools

	var machineTypes []imperatorv1alpha1.MachineType
	for _, mt := range testMachineTypes {
		machineTypes = append(machineTypes, mt)
	}
	machine.Spec.MachineTypes = machineTypes
	return machine
}

func newFakeReservationPod(machineTypeName, podNumber string) *corev1.Pod {
	return &corev1.Pod{
		TypeMeta: getPodTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: strings.Join([]string{
				util.GenerateReservationResourceName(testMachineMachineGroupName, machineTypeName),
				podNumber,
			}, "-"),
			Namespace: consts.ImperatorCoreNamespace,
			Labels:    util.GenerateReservationResourceLabel(testMachineMachineGroupName, machineTypeName),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{util.GenerateSleeperContainer()},
		},
	}
}

func newFakeGuestPod(machineTypeName string) *corev1.Pod {
	return &corev1.Pod{
		TypeMeta: getPodTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-guest-pod",
			Namespace: testGuestNs,
			Labels: map[string]string{
				consts.MachineGroupKey: testMachineMachineGroupName,
				consts.MachineTypeKey:  machineTypeName,
				consts.PodRoleKey:      consts.PodRoleGuest,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "test-container",
				Image: "test-image",
			}},
		},
	}
}

var _ = Describe("machine controller envtest", func() {
	ctx := context.TODO()
	var stopFunc func()

	defaultTestNodePool := map[string]imperatorv1alpha1.NodePool{
		testNode1: {
			Name:  testNode1,
			Mode:  imperatorv1alpha1.NodeModeReady,
			Taint: pointer.Bool(false),
			MachineType: []imperatorv1alpha1.NodePoolMachineType{{
				Name: testMachine1,
			}},
		},
		testNode2: {
			Name:  testNode2,
			Mode:  imperatorv1alpha1.NodeModeReady,
			Taint: pointer.Bool(true),
			MachineType: []imperatorv1alpha1.NodePoolMachineType{{
				Name: testMachine2,
			}},
		},
	}
	defaultTestMachineType := map[string]imperatorv1alpha1.MachineType{
		testMachine1: {
			Name: testMachine1,
			Spec: imperatorv1alpha1.MachineDetailSpec{
				CPU:    resource.MustParse("2000m"),
				Memory: resource.MustParse("8Gi"),
				GPU: &imperatorv1alpha1.GPUSpec{
					Type:   "nvidia.com/gpu",
					Num:    resource.MustParse("1"),
					Family: "ampere",
				},
			},
			Available: 2,
		},
		testMachine2: {
			Name: testMachine2,
			Spec: imperatorv1alpha1.MachineDetailSpec{
				CPU:    resource.MustParse("6000m"),
				Memory: resource.MustParse("32Gi"),
			},
			Available: 1,
		},
	}

	BeforeEach(func() {
		machines := &imperatorv1alpha1.MachineList{}
		Expect(k8sClient.List(ctx, machines, &client.ListOptions{})).NotTo(HaveOccurred())
		Expect(k8sClient.DeleteAllOf(ctx, &imperatorv1alpha1.MachineNodePool{}, &client.DeleteAllOfOptions{})).NotTo(HaveOccurred())
		Expect(k8sClient.DeleteAllOf(ctx, &imperatorv1alpha1.Machine{}, &client.DeleteAllOfOptions{})).NotTo(HaveOccurred())
		Expect(k8sClient.DeleteAllOf(ctx, &corev1.Node{}, &client.DeleteAllOfOptions{})).NotTo(HaveOccurred())

		// create node
		for _, np := range defaultTestNodePool {
			node := newFakeNode(np.Name)
			Expect(k8sClient.Create(ctx, node, &client.CreateOptions{})).NotTo(HaveOccurred())
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Name: node.Name}, &corev1.Node{})
			}, consts.SuiteTestTimeOut).Should(BeNil())
		}

		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme,
		})
		Expect(err).ToNot(HaveOccurred())

		Expect((&MachineReconciler{
			Client:   k8sClient,
			Scheme:   scheme,
			Recorder: mgr.GetEventRecorderFor("imperator"),
		}).SetupWithManager(mgr)).NotTo(HaveOccurred())

		if err := os.Setenv("ENVTEST", "true"); err != nil {
			panic(err)
		}
		ctx, stopFunc = context.WithCancel(ctx)
		go func() {
			err := mgr.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
		time.Sleep(100 * time.Millisecond)
	})

	AfterEach(func() {
		stopFunc()
		time.Sleep(100 * time.Millisecond)
	})

	It("Should create Reservation role resources", func() {
		machine := newFakeMachine(defaultTestNodePool, defaultTestMachineType)
		Expect(k8sClient.Create(ctx, machine, &client.CreateOptions{})).NotTo(HaveOccurred())

		// Check {APIVersion: imperator.tenzen-y.io/v1alpha1, Kind: Machine}
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Name: testMachineMachineGroupName}, &imperatorv1alpha1.Machine{})
		}, consts.SuiteTestTimeOut).Should(BeNil())

		// Check {APIVersion: imperator.tenzen-y.io/v1alpha1, Kind: MachineNodePool}
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Name: util.GenerateMachineNodePoolName(testMachineMachineGroupName)}, &imperatorv1alpha1.MachineNodePool{})
		}, consts.SuiteTestTimeOut).Should(BeNil())

		// Check {APIVersion: apps/v1, Kind: StatefulSet}, {APIVersion: v1, Kind: Service}, {APIVersion: v1, Kind: Pod}
		for _, mt := range defaultTestMachineType {
			waitStartedReservationResource(ctx, mt, mt.Available)

			// Create Reservation Pod
			for podNum := 0; podNum < int(mt.Available); podNum++ {
				pod := newFakeReservationPod(mt.Name, strconv.Itoa(podNum))
				Expect(k8sClient.Create(ctx, pod, &client.CreateOptions{})).NotTo(HaveOccurred())

				// Update Status of Reservation Pod
				updatePodContainerStatus(ctx, client.ObjectKey{
					Name: strings.Join([]string{
						util.GenerateReservationResourceName(testMachineMachineGroupName, mt.Name),
						strconv.Itoa(podNum),
					}, "-"),
					Namespace: consts.ImperatorCoreNamespace,
				}, "running")
			}

			// Check number of Reservation Pods
			pods := &corev1.PodList{}
			Eventually(func() int {
				Expect(k8sClient.List(ctx, pods, &client.ListOptions{
					LabelSelector: labels.SelectorFromSet(util.GenerateReservationResourceLabel(testMachineMachineGroupName, mt.Name)),
				})).NotTo(HaveOccurred())
				return len(pods.Items)
			}, consts.SuiteTestTimeOut).Should(Equal(int(mt.Available)))

			// Check Machine Status
			checkMachineAvailableStatus(ctx, mt.Name, gstruct.Fields{
				"Used":        Equal(int32(0)),
				"Maximum":     Equal(mt.Available),
				"Reservation": Equal(mt.Available),
				"Waiting":     Equal(int32(0)),
			})
		}

		// Create Guest Pod
		guestPod := newFakeGuestPod(testMachine2)
		Expect(k8sClient.Create(ctx, guestPod, &client.CreateOptions{})).NotTo(HaveOccurred())
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Name: guestPod.Name, Namespace: guestPod.Namespace}, &corev1.Pod{})
		}, consts.SuiteTestTimeOut).Should(BeNil())

		// Update Status of Guest Pod to Pending
		updatePodContainerStatus(ctx, client.ObjectKey{
			Name:      guestPod.Name,
			Namespace: guestPod.Namespace,
		}, "pending")

		testMachine2MachineAvailable := defaultTestMachineType[testMachine2].Available
		testMachine2ReservationPodName := strings.Join([]string{
			util.GenerateReservationResourceName(testMachineMachineGroupName, testMachine2),
			"0",
		}, "-")

		// Check Replicas of StatefulSet for reservation
		waitStartedReservationResource(ctx, defaultTestMachineType[testMachine2], testMachine2MachineAvailable-1)

		// Delete Reservation Pod
		pod := &corev1.Pod{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{
			Name:      testMachine2ReservationPodName,
			Namespace: consts.ImperatorCoreNamespace,
		}, pod)).NotTo(HaveOccurred())
		Expect(k8sClient.Delete(ctx, pod, &client.DeleteOptions{})).NotTo(HaveOccurred())

		// Check Machine Status
		checkMachineAvailableStatus(ctx, testMachine2, gstruct.Fields{
			"Used":        Equal(int32(0)),
			"Maximum":     Equal(testMachine2MachineAvailable),
			"Reservation": Equal(testMachine2MachineAvailable - 1),
			"Waiting":     Equal(int32(1)),
		})

		// Update Status of Guest Pod to Running
		updatePodContainerStatus(ctx, client.ObjectKey{
			Name:      guestPod.Name,
			Namespace: guestPod.Namespace,
		}, "running")

		// Check Machine Status
		checkMachineAvailableStatus(ctx, testMachine2, gstruct.Fields{
			"Used":        Equal(int32(1)),
			"Maximum":     Equal(testMachine2MachineAvailable),
			"Reservation": Equal(testMachine2MachineAvailable - 1),
			"Waiting":     Equal(int32(0)),
		})

		// Delete Guest Pod
		pod = &corev1.Pod{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{
			Name:      guestPod.Name,
			Namespace: guestPod.Namespace,
		}, pod)).NotTo(HaveOccurred())
		Expect(k8sClient.Delete(ctx, pod, &client.DeleteOptions{})).NotTo(HaveOccurred())

		// Check Replicas of StatefulSet for reservation
		waitStartedReservationResource(ctx, defaultTestMachineType[testMachine2], testMachine2MachineAvailable)

		// Create Reservation Pod for test-machine2
		pod = newFakeReservationPod(testMachine2, "0")
		Expect(k8sClient.Create(ctx, pod, &client.CreateOptions{})).NotTo(HaveOccurred())

		// Update Status of Reservation Pod for test-machine2
		updatePodContainerStatus(ctx, client.ObjectKey{
			Name:      testMachine2ReservationPodName,
			Namespace: consts.ImperatorCoreNamespace,
		}, "running")

		// Check Machine Status
		checkMachineAvailableStatus(ctx, testMachine2, gstruct.Fields{
			"Used":        Equal(int32(0)),
			"Maximum":     Equal(testMachine2MachineAvailable),
			"Reservation": Equal(testMachine2MachineAvailable),
			"Waiting":     Equal(int32(0)),
		})
	})
})
