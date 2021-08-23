package controllers

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	imperatorv1alpha1 "github.com/tenzen-y/imperator/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

const (
	readyTestNode0       = "ready-node0"
	readyTestNode1       = "ready-node1"
	maintenanceTestNode  = "maintenance-node"
	notReadyTestNode     = "not-ready-node"
	testMachineGroupName = "test-machines"
)

func createNode(ctx context.Context, name string, labels map[string]string) {
	node := &corev1.Node{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Node",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
	Expect(k8sClient.Create(ctx, node)).NotTo(HaveOccurred())
}

type testNode struct {
	name   string
	mode   string
	status string
}

func newTestMachineNodePool(testNodes []testNode) *imperatorv1alpha1.MachineNodePool {
	pool := &imperatorv1alpha1.MachineNodePool{}
	pool.ObjectMeta = metav1.ObjectMeta{
		Name: strings.Join([]string{testMachineGroupName, "node-pool"}, "-"),
		Labels: map[string]string{
			consts.MachineGroupKey: testMachineGroupName,
		},
	}
	pool.Spec.MachineGroupName = testMachineGroupName
	for _, node := range testNodes {
		pool.Spec.NodePool = append(pool.Spec.NodePool, imperatorv1alpha1.NodePool{
			Name: node.name,
			Mode: node.mode,
		})
	}

	return pool
}

var _ = Describe("imperator reconciler", func() {
	ctx := context.Background()
	var stopFunc func()
	testNodes := []testNode{
		{
			name: readyTestNode0,
			mode: consts.MachineStatusReady,
		},
		{
			name: maintenanceTestNode,
			mode: "maintenance",
		},
		{
			name: readyTestNode1,
			mode: consts.MachineStatusReady,
		},
	}

	BeforeEach(func() {
		err := k8sClient.DeleteAllOf(ctx, &imperatorv1alpha1.MachineNodePool{}, &client.DeleteAllOfOptions{})
		Expect(err).NotTo(HaveOccurred())
		err = k8sClient.DeleteAllOf(ctx, &corev1.Node{}, &client.DeleteAllOfOptions{})
		Expect(err).NotTo(HaveOccurred())
		for _, node := range testNodes {
			createNode(ctx, node.name, map[string]string{})
		}
		time.Sleep(100 * time.Millisecond)

		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme,
		})
		Expect(err).ToNot(HaveOccurred())

		err = (&MachineNodePoolReconciler{
			Client:   k8sClient,
			Scheme:   scheme,
			Recorder: mgr.GetEventRecorderFor("imperator"),
		}).SetupWithManager(mgr)
		Expect(err).NotTo(HaveOccurred())

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

	It("Should set label to all nodes", func() {
		pool := newTestMachineNodePool(testNodes)
		err := k8sClient.Create(ctx, pool, &client.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})
})
