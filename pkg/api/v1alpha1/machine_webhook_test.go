package v1alpha1

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tenzen-y/imperator/pkg/consts"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testMachineGroup = "test-machine"
)

func newFakeMachine() *Machine {
	return &Machine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       consts.KindMachine,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: testMachineGroup,
			Labels: map[string]string{
				consts.MachineGroupKey: testMachineGroup,
			},
		},
		Spec: MachineSpec{
			NodePool: []NodePool{
				{
					Name:  "test-node1",
					Mode:  NodeModeReady,
					Taint: pointer.Bool(false),
					MachineType: []NodePoolMachineType{{
						Name: "test-machine1",
					}},
				},
				{
					Name:  "test-node2",
					Mode:  NodeModeMaintenance,
					Taint: pointer.Bool(false),
					MachineType: []NodePoolMachineType{{
						Name: "test-machine2",
					}},
				},
				{
					Name:  "test-node3",
					Mode:  NodeModeReady,
					Taint: pointer.Bool(true),
					MachineType: []NodePoolMachineType{{
						Name: "test-machine1",
					}},
				},
			},
			MachineTypes: []MachineType{
				{
					Name: "test-machine1",
					Spec: MachineDetailSpec{
						CPU:    resource.MustParse("4000m"),
						Memory: resource.MustParse("24Gi"),
						GPU: &GPUSpec{
							Type:       "nvidia.com/gpu",
							Num:        resource.MustParse("2"),
							Generation: "ampere",
						},
					},
					Available: 2,
				},
				{
					Name: "test-machine2",
					Spec: MachineDetailSpec{
						CPU:    resource.MustParse("2000m"),
						Memory: resource.MustParse("12Gi"),
						GPU: &GPUSpec{
							Type:       "nvidia.com/gpu",
							Num:        resource.MustParse("1"),
							Generation: "ampere",
						},
					},
					Available: 2,
				},
			},
		},
	}
}

func createNodes(ctx context.Context) {
	testParent1 := &corev1.Node{
		TypeMeta: getNodeTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node1",
		},
	}
	testChild1 := &corev1.Node{
		TypeMeta: getNodeTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node2",
		},
	}
	testChild2 := &corev1.Node{
		TypeMeta: getNodeTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node3",
		},
	}

	Expect(k8sClient.Create(ctx, testParent1, &client.CreateOptions{})).NotTo(HaveOccurred())
	Expect(k8sClient.Create(ctx, testChild1, &client.CreateOptions{})).NotTo(HaveOccurred())
	Expect(k8sClient.Create(ctx, testChild2, &client.CreateOptions{})).NotTo(HaveOccurred())
}

func getNodeTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: corev1.SchemeGroupVersion.String(),
		Kind:       "Node",
	}
}

var _ = Describe("Machine Webhook", func() {
	ctx := context.Background()

	BeforeEach(func() {
		Expect(k8sClient.DeleteAllOf(ctx, &Machine{}, &client.DeleteAllOfOptions{})).NotTo(HaveOccurred())
		Expect(k8sClient.DeleteAllOf(ctx, &corev1.Node{}, &client.DeleteAllOfOptions{})).NotTo(HaveOccurred())
		createNodes(ctx)
	})

	It("Create Machine resource successfully", func() {
		fakeMachine := newFakeMachine()
		Expect(k8sClient.Create(ctx, fakeMachine, &client.CreateOptions{})).NotTo(HaveOccurred())
		Expect(k8sClient.Delete(ctx, fakeMachine, &client.DeleteOptions{})).NotTo(HaveOccurred())
	})

	It("Failed to create Machine resource", func() {
		testCases := []struct {
			description   string
			fakeMachine   *Machine
			kubeResources []client.Object
			err           bool
		}{
			{
				description: "All items are valid",
				fakeMachine: newFakeMachine(),
				err:         false,
			},
			{
				description: fmt.Sprintf("Missing key, %s in labels", consts.MachineGroupKey),
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					delete(fakeMachine.Labels, consts.MachineGroupKey)
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "MachineGroup label is duplicated",
				fakeMachine: newFakeMachine(),
				kubeResources: func() []client.Object {
					duplicatedMachineGroupMachine := newFakeMachine()
					duplicatedMachineGroupMachine.Name = "duplicated-machine-group"
					return []client.Object{duplicatedMachineGroupMachine}
				}(),
				err: true,
			},
			{
				description: "Specified non exist node name to nodePool",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.NodePool[0].Name = "non-exist"
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "Taint field is nil",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.NodePool[0].Taint = nil
					return fakeMachine
				}(),
				err: false,
			},
			{
				description: "Type of GPU must be set value",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.MachineTypes[0].Spec.GPU.Type = ""
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "Number of GPU must be set 0 or more value",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.MachineTypes[0].Spec.GPU.Num = resource.MustParse("-1")
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "GPU generation must be set value",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.MachineTypes[0].Spec.GPU.Generation = ""
					return fakeMachine
				}(),
				err: true,
			},
		}

		for _, test := range testCases {
			if len(test.kubeResources) > 0 {
				for _, o := range test.kubeResources {
					Expect(k8sClient.Create(ctx, o, &client.CreateOptions{})).NotTo(HaveOccurred(), test.description)
				}
			}
			err := k8sClient.Create(ctx, test.fakeMachine, &client.CreateOptions{})
			if test.err {
				Expect(err).To(HaveOccurred(), test.description)
			} else {
				Expect(err).NotTo(HaveOccurred(), test.description)
			}
			Expect(k8sClient.DeleteAllOf(ctx, &Machine{}, &client.DeleteAllOfOptions{})).NotTo(HaveOccurred(), test.description)
			if len(test.kubeResources) > 0 {
				for _, o := range test.kubeResources {
					Expect(k8sClient.DeleteAllOf(ctx, o, &client.DeleteAllOfOptions{})).NotTo(HaveOccurred(), test.description)
				}
			}
		}
	})
})
