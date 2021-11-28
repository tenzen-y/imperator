package controllers

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	testMachineGroup = "test-machine"
)

func newFakeMachine() *imperatorv1alpha1.Machine {
	return &imperatorv1alpha1.Machine {
		TypeMeta: metav1.TypeMeta{
			APIVersion: imperatorv1alpha1.GroupVersion.String(),
			Kind: consts.KindMachine,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: testMachineGroup,
			Labels: map[string]string{
				consts.MachineGroupKey: testMachineGroup,
			},
		},
		Spec: imperatorv1alpha1.MachineSpec{
			NodePool: []imperatorv1alpha1.NodePool {
				{
					Name: "test-node1",
					Mode: imperatorv1alpha1.NodeModeReady,
					AssignmentType: imperatorv1alpha1.AssignmentTypeLabel,
					MachineType: imperatorv1alpha1.NodePoolMachineType{
						Name: "test1-parent",
						ScheduleChildren: pointer.Bool(true),
					},
				},
			},
		},
	}
}

var _ = Describe("machine controller envtest", func() {
	ctx := context.TODO()
	var stopFunc func()

	BeforeEach(func() {
		machines := &imperatorv1alpha1.MachineList{}
		Expect(k8sClient.List(ctx, machines, &client.ListOptions{})).NotTo(HaveOccurred())
		Expect(k8sClient.DeleteAllOf(ctx, &imperatorv1alpha1.Machine{}, &client.DeleteAllOfOptions{})).NotTo(HaveOccurred())

		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme,
		})
		Expect(err).ToNot(HaveOccurred())

		Expect((&MachineReconciler{
			Client: k8sClient,
			Scheme: scheme,
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

	It("Should create StatefulSet", func() {
		machine := newFakeMachine()
		Expect(k8sClient.Create(ctx, machine, &client.CreateOptions{})).NotTo(HaveOccurred())
	})
})
