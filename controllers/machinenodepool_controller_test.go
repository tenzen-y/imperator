package controllers

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	imperatorv1alpha1 "github.com/tenzen-y/imperator/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

const (
	readyTestNodeA       = "ready-node-a"
	readyTestNodeB       = "ready-node-b"
	maintenanceTestNode  = "maintenance-node"
	testMachineGroupName = "test-machines"
	suiteTestTimeOut     = time.Second * 30
)

var testMachineNodePoolName = strings.Join([]string{testMachineGroupName, "node-pool"}, "-")

func createNode(ctx context.Context, testNodes []testNode) {

	for _, n := range testNodes {
		node := &corev1.Node{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Node",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   n.name,
				Labels: map[string]string{},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				}},
			},
		}

		Expect(k8sClient.Create(ctx, node)).NotTo(HaveOccurred())
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: n.name}, &corev1.Node{})).NotTo(HaveOccurred())
	}
}

type testNode struct {
	name           string
	mode           string
	status         imperatorv1alpha1.MachineNodeCondition
	assignmentType string
}

func newTestMachineNodePool(testNodes []testNode) *imperatorv1alpha1.MachineNodePool {
	pool := &imperatorv1alpha1.MachineNodePool{}
	pool.TypeMeta = metav1.TypeMeta{
		Kind: "MachineNodePool",
		APIVersion: strings.Join([]string{
			imperatorv1alpha1.GroupVersion.Group,
			imperatorv1alpha1.GroupVersion.Version,
		}, "/"),
	}
	pool.ObjectMeta = metav1.ObjectMeta{
		Name: testMachineNodePoolName,
		Labels: map[string]string{
			consts.MachineGroupKey: testMachineGroupName,
		},
	}
	pool.Spec.MachineGroupName = testMachineGroupName
	for _, node := range testNodes {
		pool.Spec.NodePool = append(pool.Spec.NodePool, imperatorv1alpha1.NodePool{
			Name:           node.name,
			Mode:           node.mode,
			AssignmentType: node.assignmentType,
		})
	}
	return pool
}

func extractKeyValueFromTaint(taints []corev1.Taint) map[string]string {
	result := map[string]string{}
	for _, t := range taints {
		result[t.Key] = t.Value
	}
	return result
}

func waitUpdateTestNode(ctx context.Context, testNodes []testNode) {
	for _, n := range testNodes {
		target := &corev1.Node{}
		Eventually(func() map[string]string {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: n.name}, target)).NotTo(HaveOccurred())
			// label
			if n.assignmentType == consts.AssignLabel {
				return target.Labels
			}
			// taint
			resultTaints := extractKeyValueFromTaint(target.Spec.Taints)
			delete(resultTaints, "node.kubernetes.io/not-ready")
			return resultTaints
		}, suiteTestTimeOut).ShouldNot(BeEmpty())

		actual := target.Labels
		if n.assignmentType == consts.AssignTaint {
			actual = extractKeyValueFromTaint(target.Spec.Taints)
		}
		Expect(actual).To(HaveKeyWithValue(consts.MachineStatusKey, n.mode))
		Expect(target.Annotations).To(HaveKeyWithValue(consts.MachineGroupKey, testMachineGroupName))
	}
}

func waitUpdateTestMachineNodePoolCondition(ctx context.Context, testNodes []testNode) {
	getPool := &imperatorv1alpha1.MachineNodePool{}
	Eventually(func() []imperatorv1alpha1.NodePoolCondition {
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: testMachineNodePoolName}, getPool)).NotTo(HaveOccurred())
		return getPool.Status.NodePoolCondition
	}, suiteTestTimeOut).Should(HaveLen(3))

	for _, n := range testNodes {
		Eventually(func() imperatorv1alpha1.MachineNodeCondition {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: testMachineNodePoolName}, getPool)).NotTo(HaveOccurred())
			for _, pc := range getPool.Status.NodePoolCondition {
				if n.name != pc.Name {
					continue
				}
				return pc.NodeCondition
			}
			return ""
		}, suiteTestTimeOut).Should(Equal(n.status))
	}
}

func updateTestMachineNodePoolNodePool(ctx context.Context, patch []testNode) {
	machineNodePool := &imperatorv1alpha1.MachineNodePool{}
	Expect(k8sClient.Get(ctx, client.ObjectKey{Name: testMachineNodePoolName}, machineNodePool)).NotTo(HaveOccurred())

	testParams := map[string]testNode{}
	for _, node := range patch {
		testParams[node.name] = node
	}
	for _, pool := range machineNodePool.Spec.NodePool {
		pool.Mode = testParams[pool.Name].mode
	}

	Expect(k8sClient.Update(ctx, machineNodePool, &client.UpdateOptions{})).NotTo(HaveOccurred())
}

var _ = Describe("imperator reconciler", func() {
	ctx := context.Background()
	var stopFunc func()
	testNodes := []testNode{
		{
			name:           readyTestNodeA,
			mode:           consts.MachineStatusReady,
			status:         imperatorv1alpha1.NodeHealthy,
			assignmentType: consts.AssignLabel,
		},
		{
			name:           readyTestNodeB,
			mode:           consts.MachineStatusReady,
			status:         imperatorv1alpha1.NodeHealthy,
			assignmentType: consts.AssignTaint,
		},
		{
			name:           maintenanceTestNode,
			mode:           consts.MachineStatusMaintenance,
			status:         imperatorv1alpha1.NodeMaintenance,
			assignmentType: consts.AssignLabel,
		},
	}

	BeforeEach(func() {
		pools := &imperatorv1alpha1.MachineNodePoolList{}
		Expect(k8sClient.List(ctx, pools, &client.ListOptions{})).NotTo(HaveOccurred())
		for _, pool := range pools.Items {
			pool.Finalizers = nil
			Expect(k8sClient.Update(ctx, &pool, &client.UpdateOptions{})).NotTo(HaveOccurred())
		}
		Expect(k8sClient.DeleteAllOf(ctx, &imperatorv1alpha1.MachineNodePool{}, &client.DeleteAllOfOptions{})).NotTo(HaveOccurred())
		Expect(k8sClient.DeleteAllOf(ctx, &corev1.Node{}, &client.DeleteAllOfOptions{})).NotTo(HaveOccurred())
		createNode(ctx, testNodes)

		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme,
		})
		Expect(err).ToNot(HaveOccurred())

		Expect((&MachineNodePoolReconciler{
			Client:   k8sClient,
			Scheme:   scheme,
			Recorder: mgr.GetEventRecorderFor("imperator"),
		}).SetupWithManager(mgr)).NotTo(HaveOccurred())

		if err := os.Setenv("ENVTEST", "true"); err != nil {
			panic(err)
		}
		ctx, cancel := context.WithCancel(ctx)
		stopFunc = cancel
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

	// create -> delete
	It("Should set label or taint to all nodes, update MachineNodePool status", func() {
		pool := newTestMachineNodePool(testNodes)
		Expect(k8sClient.Create(ctx, pool, &client.CreateOptions{})).NotTo(HaveOccurred())
		waitUpdateTestNode(ctx, testNodes)
		waitUpdateTestMachineNodePoolCondition(ctx, testNodes)

		Expect(k8sClient.Delete(ctx, pool, &client.DeleteOptions{})).NotTo(HaveOccurred())
		Eventually(func() error {
			getPool := &corev1.Node{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: testMachineNodePoolName}, getPool); err != nil {
				return err
			}
			return nil
		}, suiteTestTimeOut).ShouldNot(BeNil())

		for _, n := range testNodes {

			target := &corev1.Node{}
			actual := map[string]string{}
			Eventually(func() map[string]string {
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: n.name}, target)).NotTo(HaveOccurred())
				// label
				actual = target.Labels
				// taint
				if n.assignmentType == consts.AssignTaint {
					actual = extractKeyValueFromTaint(target.Spec.Taints)
					delete(actual, "node.kubernetes.io/not-ready")
				}
				return actual
			}, suiteTestTimeOut).Should(BeEmpty())

			Eventually(func() string {
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: n.name}, target)).NotTo(HaveOccurred())
				return target.Annotations[consts.MachineGroupKey]
			}, suiteTestTimeOut).Should(BeEmpty())

		}
	})

	It("change node status", func() {
		pool := newTestMachineNodePool(testNodes)
		Expect(k8sClient.Create(ctx, pool, &client.CreateOptions{})).NotTo(HaveOccurred())
		waitUpdateTestNode(ctx, testNodes)
		waitUpdateTestMachineNodePoolCondition(ctx, testNodes)

		// node.kubernetes.io/unschedulable:NoSchedule
		getReadyTestNodeA := &corev1.Node{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: readyTestNodeA}, getReadyTestNodeA)).NotTo(HaveOccurred())

		now := metav1.Now()
		getReadyTestNodeA.Spec.Taints = append(getReadyTestNodeA.Spec.Taints, corev1.Taint{
			Key:       consts.CannotUseNodeTaints[1],
			Effect:    corev1.TaintEffectNoSchedule,
			TimeAdded: &now,
		})
		Expect(k8sClient.Update(ctx, getReadyTestNodeA, &client.UpdateOptions{})).NotTo(HaveOccurred())

		getReadyTestNodeA = &corev1.Node{}
		Eventually(func() corev1.TaintEffect {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: readyTestNodeA}, getReadyTestNodeA)).NotTo(HaveOccurred())
			for _, t := range getReadyTestNodeA.Spec.Taints {
				if t.Key != consts.CannotUseNodeTaints[1] {
					continue
				}
				return t.Effect
			}
			return ""
		}, suiteTestTimeOut).Should(Equal(corev1.TaintEffectNoSchedule))

		readyTestNodeAAssignmentType := ""
		for _, n := range testNodes {
			if n.name != readyTestNodeA {
				continue
			}
			readyTestNodeAAssignmentType = n.assignmentType
		}
		getReadyTestNodeA = &corev1.Node{}
		Eventually(func() string {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: readyTestNodeA}, getReadyTestNodeA)).NotTo(HaveOccurred())
			actual := getReadyTestNodeA.Labels[consts.MachineStatusKey]
			if readyTestNodeAAssignmentType == consts.AssignTaint {
				actual = extractKeyValueFromTaint(getReadyTestNodeA.Spec.Taints)[consts.MachineStatusKey]
			}
			return actual
		}, suiteTestTimeOut).Should(Equal(consts.MachineStatusNotReady))

		getPool := &imperatorv1alpha1.MachineNodePool{}
		Eventually(func() imperatorv1alpha1.MachineNodeCondition {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: testMachineNodePoolName}, getPool)).NotTo(HaveOccurred())
			for _, c := range getPool.Status.NodePoolCondition {
				if c.Name != readyTestNodeA {
					continue
				}
				return c.NodeCondition
			}
			return ""
		}, suiteTestTimeOut).Should(Equal(imperatorv1alpha1.NodeUnhealthy))

		// "node.kubernetes.io/network-unavailable":NoSchedule
		getReadyTestNodeB := &corev1.Node{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: readyTestNodeB}, getReadyTestNodeB)).NotTo(HaveOccurred())

		now = metav1.Now()
		getReadyTestNodeB.Spec.Taints = append(getReadyTestNodeB.Spec.Taints, corev1.Taint{
			Key:       consts.CannotUseNodeTaints[2],
			Effect:    corev1.TaintEffectNoSchedule,
			TimeAdded: &now,
		})
		Expect(k8sClient.Update(ctx, getReadyTestNodeB, &client.UpdateOptions{})).NotTo(HaveOccurred())

		getReadyTestNodeB = &corev1.Node{}
		Eventually(func() corev1.TaintEffect {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: readyTestNodeB}, getReadyTestNodeB)).NotTo(HaveOccurred())
			for _, t := range getReadyTestNodeB.Spec.Taints {
				if t.Key != consts.CannotUseNodeTaints[2] {
					continue
				}
				return t.Effect
			}
			return ""
		}, suiteTestTimeOut).Should(Equal(corev1.TaintEffectNoSchedule))

		readyTestNodeBAssignmentType := ""
		for _, n := range testNodes {
			if n.name != readyTestNodeB {
				continue
			}
			readyTestNodeBAssignmentType = n.assignmentType
		}
		getReadyTestNodeB = &corev1.Node{}
		Eventually(func() string {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: readyTestNodeB}, getReadyTestNodeB)).NotTo(HaveOccurred())

			actual := getReadyTestNodeB.Labels[consts.MachineStatusKey]
			if readyTestNodeBAssignmentType == consts.AssignTaint {
				actual = extractKeyValueFromTaint(getReadyTestNodeB.Spec.Taints)[consts.MachineStatusKey]
			}
			return actual
		}, suiteTestTimeOut).Should(Equal(consts.MachineStatusNotReady))

		getPool = &imperatorv1alpha1.MachineNodePool{}
		Eventually(func() imperatorv1alpha1.MachineNodeCondition {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: testMachineNodePoolName}, getPool)).NotTo(HaveOccurred())
			for _, c := range getPool.Status.NodePoolCondition {
				if c.Name != readyTestNodeB {
					continue
				}
				return c.NodeCondition
			}
			return ""
		}, suiteTestTimeOut).Should(Equal(imperatorv1alpha1.NodeUnhealthy))

	})

	It("change node mode in MachineNodePool", func() {
		pool := newTestMachineNodePool(testNodes)
		Expect(k8sClient.Create(ctx, pool, &client.CreateOptions{})).NotTo(HaveOccurred())
		waitUpdateTestNode(ctx, testNodes)
		waitUpdateTestMachineNodePoolCondition(ctx, testNodes)

		// change node mode
		newTestNodes := testNodes
		for _, node := range newTestNodes {
			switch node.name {
			case readyTestNodeA:
				node.mode = consts.MachineStatusMaintenance
				node.status = imperatorv1alpha1.NodeMaintenance
			case maintenanceTestNode:
				node.mode = consts.MachineStatusReady
				node.status = imperatorv1alpha1.ConditionReady
			}
		}
		updateTestMachineNodePoolNodePool(ctx, newTestNodes)

		for _, node := range newTestNodes {
			pool := &imperatorv1alpha1.MachineNodePool{}
			Eventually(func() imperatorv1alpha1.MachineNodeCondition {

				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: testMachineNodePoolName}, pool)).NotTo(HaveOccurred())
				for _, poolStatusCondition := range pool.Status.NodePoolCondition {
					if node.name != poolStatusCondition.Name {
						continue
					}
					return poolStatusCondition.NodeCondition
				}
				return ""

			}, suiteTestTimeOut).Should(Equal(node.status))

			getNode := &corev1.Node{}
			Eventually(func() string {
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: node.name}, getNode)).NotTo(HaveOccurred())

				actual := getNode.Labels[consts.MachineStatusKey]
				if node.assignmentType == consts.AssignTaint {
					actual = extractKeyValueFromTaint(getNode.Spec.Taints)[consts.MachineStatusKey]
				}
				return actual
			}, suiteTestTimeOut).Should(Equal(node.mode))

		}
	})

	It("change assignment type", func() {
		pool := newTestMachineNodePool(testNodes)
		Expect(k8sClient.Create(ctx, pool, &client.CreateOptions{})).NotTo(HaveOccurred())
		waitUpdateTestNode(ctx, testNodes)
		waitUpdateTestMachineNodePoolCondition(ctx, testNodes)

		newTestNodes := testNodes
		for _, n := range newTestNodes {
			switch n.name {
			case readyTestNodeA:
				n.assignmentType = consts.AssignTaint
			case readyTestNodeB:
				n.assignmentType = consts.AssignLabel
			case maintenanceTestNode:
				n.assignmentType = consts.AssignTaint
			}
		}
		updateTestMachineNodePoolNodePool(ctx, newTestNodes)

		for _, tn := range newTestNodes {

			getNode := &corev1.Node{}
			actual := map[string]string{}
			Eventually(func() map[string]string {
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: tn.name}, getNode)).NotTo(HaveOccurred())
				actual = getNode.Labels
				if tn.assignmentType == consts.AssignTaint {
					actual = extractKeyValueFromTaint(getNode.Spec.Taints)
					delete(actual, "node.kubernetes.io/not-ready")
				}
				return actual
			}, suiteTestTimeOut).Should(HaveLen(1))

			Expect(actual).To(HaveKeyWithValue(consts.MachineStatusKey, tn.mode))
			Expect(getNode.Annotations).To(HaveKeyWithValue(consts.MachineGroupKey, testMachineGroupName))

			// must not have any elements
			actual = extractKeyValueFromTaint(getNode.Spec.Taints)
			delete(actual, "node.kubernetes.io/not-ready")
			if tn.assignmentType == consts.AssignTaint {
				actual = getNode.Labels
			}
			Expect(actual).Should(BeEmpty())

			getPool := &imperatorv1alpha1.MachineNodePool{}
			Eventually(func() imperatorv1alpha1.MachineNodeCondition {
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: testMachineNodePoolName}, getPool)).NotTo(HaveOccurred())
				for _, s := range getPool.Status.NodePoolCondition {
					if s.Name != tn.name {
						continue
					}
					return s.NodeCondition
				}
				return ""
			}, suiteTestTimeOut).Should(Equal(tn.status))

		}
	})

	It("Should not complete reconcile because controller try to register fake-node.", func() {
		newTestNodes := testNodes
		newTestNodes = append(newTestNodes, testNode{
			name:           "fake-node",
			mode:           consts.MachineStatusReady,
			status:         imperatorv1alpha1.NodeHealthy,
			assignmentType: consts.AssignTaint,
		})

		pool := newTestMachineNodePool(newTestNodes)
		Expect(k8sClient.Create(ctx, pool, &client.CreateOptions{})).NotTo(HaveOccurred())

		pool = &imperatorv1alpha1.MachineNodePool{}
		Eventually(func() string {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: testMachineNodePoolName}, pool)).NotTo(HaveOccurred())
			return pool.Annotations[consts.MachineGroupKey]
		}, suiteTestTimeOut).Should(Equal(pool.Spec.MachineGroupName))

		getPool := &imperatorv1alpha1.MachineNodePool{}
		Eventually(func() metav1.ConditionStatus {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: testMachineNodePoolName}, getPool)).NotTo(HaveOccurred())
			for _, s := range getPool.Status.Conditions {
				if s.Type != imperatorv1alpha1.ConditionReady {
					continue
				}
				return s.Status
			}
			return ""
		}, suiteTestTimeOut).Should(Equal(metav1.ConditionFalse))
	})
})
