package controllers

import (
	"context"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
	"github.com/tenzen-y/imperator/pkg/controllers/utils"
)

const (
	readyTestNodeA       = "ready-node-a"
	readyTestNodeB       = "ready-node-b"
	maintenanceTestNode  = "maintenance-node"
	testMachineGroupName = "test-machines"
	suiteTestTimeOut     = time.Second * 3
)

var testMachineNodePoolName = strings.Join([]string{testMachineGroupName, "node-pool"}, "-")

type testNode struct {
	name        string
	mode        imperatorv1alpha1.NodePoolMode
	status      imperatorv1alpha1.MachineNodeCondition
	taint       *bool
	machineType []imperatorv1alpha1.NodePoolMachineType
}

func createNode(ctx context.Context, testNodes []testNode) {

	for _, n := range testNodes {
		node := &corev1.Node{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "Node",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: n.name,
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

func newFakeMachineNodePool(testNodes []testNode, testMachineTypeStock []imperatorv1alpha1.NodePoolMachineTypeStock) *imperatorv1alpha1.MachineNodePool {
	pool := &imperatorv1alpha1.MachineNodePool{}
	pool.TypeMeta = metav1.TypeMeta{
		Kind:       consts.KindMachineNodePool,
		APIVersion: imperatorv1alpha1.GroupVersion.String(),
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
			Name:        node.name,
			Mode:        node.mode,
			Taint:       node.taint,
			MachineType: node.machineType,
		})
	}
	pool.Spec.MachineTypeStock = testMachineTypeStock

	return pool
}

func waitUpdateTestNode(ctx context.Context, testNodes []testNode) {
	for _, n := range testNodes {

		// check node label
		target := &corev1.Node{}
		Eventually(func() map[string]string {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: n.name}, target)).NotTo(HaveOccurred())
			return target.Labels
		}, suiteTestTimeOut).ShouldNot(BeEmpty())

		// check node taint
		if *n.taint {
			target = &corev1.Node{}
			Eventually(func() map[string]string {
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: n.name}, target)).NotTo(HaveOccurred())
				resultTaints := utils.ExtractKeyValueFromTaint(target.Spec.Taints)
				delete(resultTaints, consts.NodeNotReadyTaint)
				return resultTaints
			}, suiteTestTimeOut).ShouldNot(BeEmpty())
		}

		nodeLabels := target.Labels
		var nodeTaints map[string]string
		if *n.taint {
			nodeTaints = utils.ExtractKeyValueFromTaint(target.Spec.Taints)
			Expect(nodeTaints).To(HaveKeyWithValue(consts.MachineStatusKey, n.mode.Value()))
		}
		Expect(nodeLabels).To(HaveKeyWithValue(consts.MachineStatusKey, n.mode.Value()))
		Expect(target.Annotations).To(HaveKeyWithValue(consts.MachineGroupKey, testMachineGroupName))
		for _, mt := range n.machineType {
			Expect(nodeLabels).To(HaveKeyWithValue(utils.GetMachineTypeLabelTaintKey(mt.Name), testMachineGroupName))
			if *n.taint {
				Expect(nodeTaints).To(HaveKeyWithValue(utils.GetMachineTypeLabelTaintKey(mt.Name), testMachineGroupName))
			}
		}
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

func updateTestMachineNodePoolNodeMode(ctx context.Context, patch []testNode) {
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

func updateTestMachineNodePoolMachineType(ctx context.Context, patch []testNode) {
	machineNodePool := &imperatorv1alpha1.MachineNodePool{}
	Expect(k8sClient.Get(ctx, client.ObjectKey{Name: testMachineNodePoolName}, machineNodePool)).NotTo(HaveOccurred())

	testParams := map[string]testNode{}
	for _, node := range patch {
		testParams[node.name] = node
	}
	for _, pool := range machineNodePool.Spec.NodePool {
		pool.MachineType = testParams[pool.Name].machineType
	}

	Expect(k8sClient.Update(ctx, machineNodePool, &client.UpdateOptions{})).NotTo(HaveOccurred())
}

var _ = Describe("machinenodepool controller envtest", func() {
	ctx := context.Background()
	var stopFunc func()

	testMachineTypeStock := []imperatorv1alpha1.NodePoolMachineTypeStock{
		{
			Name: "compute-xlarge",
		},
		{
			Name: "compute-xmedium",
		},
		{
			Name: "compute-xsmall",
		},
	}
	testNodes := []testNode{
		{
			name:        readyTestNodeA,
			mode:        imperatorv1alpha1.NodeModeReady,
			status:      imperatorv1alpha1.NodeHealthy,
			taint:       pointer.Bool(false),
			machineType: []imperatorv1alpha1.NodePoolMachineType{{Name: "compute-xlarge"}},
		},
		{
			name:        readyTestNodeB,
			mode:        imperatorv1alpha1.NodeModeReady,
			status:      imperatorv1alpha1.NodeHealthy,
			taint:       pointer.Bool(true),
			machineType: []imperatorv1alpha1.NodePoolMachineType{{Name: "compute-xmedium"}},
		},
		{
			name:        maintenanceTestNode,
			mode:        imperatorv1alpha1.NodeModeMaintenance,
			status:      imperatorv1alpha1.NodeMaintenance,
			taint:       pointer.Bool(false),
			machineType: []imperatorv1alpha1.NodePoolMachineType{{Name: "compute-xsmall"}},
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
	It("Should set label to all nodes, update MachineNodePool status", func() {
		pool := newFakeMachineNodePool(testNodes, testMachineTypeStock)
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

			// check label
			target := &corev1.Node{}
			Eventually(func() map[string]string {
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: n.name}, target)).NotTo(HaveOccurred())
				return target.Labels
			}, suiteTestTimeOut).Should(BeEmpty())

			// check taint
			if *n.taint {
				target = &corev1.Node{}
				Eventually(func() map[string]string {
					Expect(k8sClient.Get(ctx, client.ObjectKey{Name: n.name}, target)).NotTo(HaveOccurred())
					nodeTaints := utils.ExtractKeyValueFromTaint(target.Spec.Taints)
					delete(nodeTaints, consts.NodeNotReadyTaint)
					return nodeTaints
				}, suiteTestTimeOut).Should(BeEmpty())
			}

			// check annotation
			Eventually(func() string {
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: n.name}, target)).NotTo(HaveOccurred())
				return target.Annotations[consts.MachineGroupKey]
			}, suiteTestTimeOut).Should(BeEmpty())

		}
	})

	It("Change node status", func() {
		pool := newFakeMachineNodePool(testNodes, testMachineTypeStock)
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
		Eventually(func() error {
			return k8sClient.Update(ctx, getReadyTestNodeA, &client.UpdateOptions{})
		}, suiteTestTimeOut).Should(BeNil())

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

		getReadyTestNodeA = &corev1.Node{}
		Eventually(func() string {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: readyTestNodeA}, getReadyTestNodeA)).NotTo(HaveOccurred())
			return getReadyTestNodeA.Labels[consts.MachineStatusKey]
		}, suiteTestTimeOut).Should(Equal(imperatorv1alpha1.NodeModeNotReady.Value()))

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

		// check node label
		getReadyTestNodeB = &corev1.Node{}
		Eventually(func() string {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: readyTestNodeB}, getReadyTestNodeB)).NotTo(HaveOccurred())
			return getReadyTestNodeB.Labels[consts.MachineStatusKey]
		}, suiteTestTimeOut).Should(Equal(imperatorv1alpha1.NodeModeNotReady.Value()))

		// check node taint
		getReadyTestNodeB = &corev1.Node{}
		Eventually(func() string {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: readyTestNodeB}, getReadyTestNodeB)).NotTo(HaveOccurred())
			return utils.ExtractKeyValueFromTaint(getReadyTestNodeB.Spec.Taints)[consts.MachineStatusKey]
		}, suiteTestTimeOut).Should(Equal(imperatorv1alpha1.NodeModeNotReady.Value()))

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

	It("Change node mode in MachineNodePool", func() {
		pool := newFakeMachineNodePool(testNodes, testMachineTypeStock)
		Expect(k8sClient.Create(ctx, pool, &client.CreateOptions{})).NotTo(HaveOccurred())
		waitUpdateTestNode(ctx, testNodes)
		waitUpdateTestMachineNodePoolCondition(ctx, testNodes)

		// change node mode
		newTestNodes := testNodes
		for _, node := range newTestNodes {
			switch node.name {
			case readyTestNodeA:
				node.mode = imperatorv1alpha1.NodeModeMaintenance
				node.status = imperatorv1alpha1.NodeMaintenance
			case maintenanceTestNode:
				node.mode = imperatorv1alpha1.NodeModeReady
				node.status = imperatorv1alpha1.ConditionReady
			}
		}
		updateTestMachineNodePoolNodeMode(ctx, newTestNodes)

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
				return getNode.Labels[consts.MachineStatusKey]
			}, suiteTestTimeOut).Should(Equal(node.mode.Value()))

			if *node.taint {
				Eventually(func() string {
					Expect(k8sClient.Get(ctx, client.ObjectKey{Name: node.name}, getNode)).NotTo(HaveOccurred())
					return utils.ExtractKeyValueFromTaint(getNode.Spec.Taints)[consts.MachineStatusKey]
				}, suiteTestTimeOut).Should(Equal(node.mode.Value()))
			}

		}
	})

	It("Change taint in MachineNodePool", func() {
		pool := newFakeMachineNodePool(testNodes, testMachineTypeStock)
		Expect(k8sClient.Create(ctx, pool, &client.CreateOptions{})).NotTo(HaveOccurred())
		waitUpdateTestNode(ctx, testNodes)
		waitUpdateTestMachineNodePoolCondition(ctx, testNodes)

		newTestNodes := testNodes
		for _, n := range newTestNodes {
			switch n.name {
			case readyTestNodeA:
				n.taint = pointer.Bool(true)
			case readyTestNodeB:
				n.taint = pointer.Bool(false)
			case maintenanceTestNode:
				n.taint = pointer.Bool(true)
			}
		}
		updateTestMachineNodePoolNodeMode(ctx, newTestNodes)

		for _, tn := range newTestNodes {
			getNode := &corev1.Node{}
			actual := map[string]string{}
			expectedTaintNum := 0
			if *tn.taint {
				expectedTaintNum = 1 + len(tn.machineType)
			}
			Eventually(func() map[string]string {
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: tn.name}, getNode)).NotTo(HaveOccurred())
				actual = utils.ExtractKeyValueFromTaint(getNode.Spec.Taints)
				delete(actual, consts.NodeNotReadyTaint)
				return actual
			}, suiteTestTimeOut).Should(HaveLen(expectedTaintNum))

			if *tn.taint {
				Expect(actual).To(HaveKeyWithValue(consts.MachineStatusKey, tn.mode.Value()))
			}
			Expect(getNode.Annotations).To(HaveKeyWithValue(consts.MachineGroupKey, testMachineGroupName))

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

	It("Change machineType in MachineNodePool", func() {
		pool := newFakeMachineNodePool(testNodes, testMachineTypeStock)
		Expect(k8sClient.Create(ctx, pool, &client.CreateOptions{})).NotTo(HaveOccurred())
		waitUpdateTestNode(ctx, testNodes)
		waitUpdateTestMachineNodePoolCondition(ctx, testNodes)

		newTestNodes := testNodes
		for _, n := range newTestNodes {
			switch n.name {
			case readyTestNodeA:
				n.machineType = []imperatorv1alpha1.NodePoolMachineType{{Name: "compute-xmedium"}}
			case readyTestNodeB:
				n.machineType = []imperatorv1alpha1.NodePoolMachineType{{Name: "compute-xsmall"}}
			case maintenanceTestNode:
				n.machineType = []imperatorv1alpha1.NodePoolMachineType{{Name: "compute-xlarge"}}
			}
		}

		updateTestMachineNodePoolMachineType(ctx, newTestNodes)

		for _, tn := range newTestNodes {
			getNode := &corev1.Node{}
			nodeLabels := make(map[string]string)
			nodeTaints := make(map[string]string)
			expectedKeyNum := 1 + len(tn.machineType)
			Eventually(func() map[string]string {
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: tn.name}, getNode))
				nodeLabels = getNode.Labels
				return nodeLabels
			}, suiteTestTimeOut).Should(HaveLen(expectedKeyNum))

			if *tn.taint {
				Eventually(func() map[string]string {
					Expect(k8sClient.Get(ctx, client.ObjectKey{Name: tn.name}, getNode))
					nodeTaints = utils.ExtractKeyValueFromTaint(getNode.Spec.Taints)
					delete(nodeTaints, consts.NodeNotReadyTaint)
					return nodeTaints
				}, suiteTestTimeOut).Should(HaveLen(expectedKeyNum))
			}

			Expect(getNode.Annotations).To(HaveKeyWithValue(consts.MachineGroupKey, testMachineGroupName))
			for _, mt := range tn.machineType {
				Expect(nodeLabels).To(HaveKeyWithValue(utils.GetMachineTypeLabelTaintKey(mt.Name), testMachineGroupName))
				if *tn.taint {
					Expect(nodeTaints).To(HaveKeyWithValue(utils.GetMachineTypeLabelTaintKey(mt.Name), testMachineGroupName))
				}
			}
		}
	})

	It("Should not complete reconcile because controller try to register fake-node.", func() {
		newTestNodes := testNodes
		newTestNodes = append(newTestNodes, testNode{
			name:        "fake-node",
			mode:        imperatorv1alpha1.NodeModeReady,
			status:      imperatorv1alpha1.NodeHealthy,
			machineType: []imperatorv1alpha1.NodePoolMachineType{{Name: "machine-xmedium"}},
			taint:       pointer.Bool(false),
		})

		pool := newFakeMachineNodePool(newTestNodes, testMachineTypeStock)
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
