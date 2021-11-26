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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	testMachineGroup = "test-machine"
)

func newFakeMachine() *Machine {
	return &Machine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: strings.Join([]string{GroupVersion.Group, GroupVersion.Version}, "/"),
			Kind:       "Machine",
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
					Name:           "test-parent-1",
					Mode:           NodeModeReady,
					AssignmentType: AssignmentTypeLabel,
				},
				{
					Name:           "test-child-2",
					Mode:           NodeModeMaintenance,
					AssignmentType: AssignmentTypeTaint,
				},
			},
			MachineTypes: []MachineType{
				{
					Name: "test-parent-1",
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
					Name: "test-child-2",
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
					Dependence: &Dependence{
						Parent:         "test-parent-1",
						AvailableRatio: "0.5",
					},
				},
			},
		},
	}
}

func createNodes(ctx context.Context) {
	testParent1 := &corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: strings.Join([]string{corev1.SchemeGroupVersion.Group, corev1.SchemeGroupVersion.Version}, "/"),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-parent-1",
		},
	}
	testChild1 := &corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: strings.Join([]string{corev1.SchemeGroupVersion.Group, corev1.SchemeGroupVersion.Version}, "/"),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-child-1",
		},
	}
	testChild2 := &corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: strings.Join([]string{corev1.SchemeGroupVersion.Group, corev1.SchemeGroupVersion.Version}, "/"),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-child-2",
		},
	}

	Expect(k8sClient.Create(ctx, testParent1, &client.CreateOptions{})).NotTo(HaveOccurred())
	Expect(k8sClient.Create(ctx, testChild1, &client.CreateOptions{})).NotTo(HaveOccurred())
	Expect(k8sClient.Create(ctx, testChild2, &client.CreateOptions{})).NotTo(HaveOccurred())
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
			description string
			fakeMachine *Machine
			err         bool
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
				description: "Specified non exist node name to nodePool",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.NodePool[0].Name = "non-exist"
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "Specified non exist node name to machineType",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.MachineTypes[0].Name = "non-exist"
					return fakeMachine
				}(),
				err: true,
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
			{
				description: "Missing parent machine",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.MachineTypes[1].Dependence.Parent = ""
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "Missing available ratio for parent machine",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.MachineTypes[1].Dependence.AvailableRatio = ""
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "Parent machine name is not exist.",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.MachineTypes[1].Dependence.Parent = "non-exist"
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "AvailableRatio for parent machine must bet set as float",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.MachineTypes[1].Dependence.AvailableRatio = "foo-bar"
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "AvailableRatio for parent machine must not be set greater than 1",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.MachineTypes[1].Dependence.AvailableRatio = "1.2"
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "Unmatch CPU resource size between parent and child",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.MachineTypes[0].Spec.CPU = resource.MustParse("500m")
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "Unmatch Memory resource size between parent and child",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.MachineTypes[1].Spec.Memory = resource.MustParse("128Gi")
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "Unmatch GPU between parent and child",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.MachineTypes[1].Spec.GPU = nil
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "Unmatch GPU resource size between parent and child",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.MachineTypes[0].Spec.GPU.Num = resource.MustParse("10")
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "Unmatch GPU type between parent and child",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.MachineTypes[1].Spec.GPU.Type = "foo.bar"
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "Unmatch GPU generation between parent and child",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.MachineTypes[1].Spec.GPU.Generation = "Foo"
					return fakeMachine
				}(),
				err: true,
			},
		}

		for _, test := range testCases {
			err := k8sClient.Create(ctx, test.fakeMachine, &client.CreateOptions{})
			if test.err {
				Expect(err).To(HaveOccurred(), test.description)
			} else {
				Expect(err).NotTo(HaveOccurred(), test.description)
			}
			Expect(k8sClient.DeleteAllOf(ctx, &Machine{}, &client.DeleteAllOfOptions{})).NotTo(HaveOccurred(), test.description)
		}
	})
})
