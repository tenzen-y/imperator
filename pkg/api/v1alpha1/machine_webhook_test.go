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
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/tenzen-y/imperator/pkg/consts"
)

const (
	testMachineGroup = "test-machine-group"
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
					Taint: false,
					MachineType: []NodePoolMachineType{{
						Name: "test-machine1",
					}},
				},
				{
					Name:  "test-node2",
					Mode:  NodeModeMaintenance,
					Taint: false,
					MachineType: []NodePoolMachineType{{
						Name: "test-machine2",
					}},
				},
				{
					Name:  "test-node3",
					Mode:  NodeModeReady,
					Taint: true,
					MachineType: []NodePoolMachineType{{
						Name: "test-machine1",
					}},
				},
			},
			MachineTypes: []MachineType{
				{
					Name: "test-machine1",
					Spec: MachineDetailSpec{
						CPU:    resource.MustParse("4"),
						Memory: resource.MustParse("24Gi"),
						GPU: &GPUSpec{
							Type:   "nvidia.com/gpu",
							Num:    resource.MustParse("2"),
							Family: "ampere",
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
							Type:    "nvidia.com/gpu",
							Num:     resource.MustParse("1"),
							Product: "NVIDIA-GeForce-RTX-3090",
						},
					},
					Available: 2,
				},
			},
		},
	}
}

func newFakeNode(nodeName string) *corev1.Node {
	return &corev1.Node{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Node",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
	}
}

var _ = Describe("Machine Webhook", func() {
	BeforeEach(func() {
		Expect(k8sClient.DeleteAllOf(ctx, &Machine{}, &client.DeleteAllOfOptions{})).NotTo(HaveOccurred())
		Expect(k8sClient.DeleteAllOf(ctx, &corev1.Node{}, &client.DeleteAllOfOptions{})).NotTo(HaveOccurred())
		fakeNodes := []string{"test-node1", "test-node2", "test-node3"}
		for _, name := range fakeNodes {
			node := newFakeNode(name)
			Expect(k8sClient.Create(ctx, node, &client.CreateOptions{})).NotTo(HaveOccurred())
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Name: name}, &corev1.Node{})
			}, consts.SuiteTestTimeOut).Should(BeNil())
		}
	})

	It("Create Machine resource successfully", func() {
		fakeMachine := newFakeMachine()
		Expect(k8sClient.Create(ctx, fakeMachine, &client.CreateOptions{})).NotTo(HaveOccurred())
		Expect(k8sClient.Delete(ctx, fakeMachine, &client.DeleteOptions{})).NotTo(HaveOccurred())
	})

	It("Create Machine resource", func() {
		testCases := []struct {
			description   string
			fakeMachine   *Machine
			kubeResources []client.Object
			err           bool
		}{
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
				description: "GPU family, product or machine must be set value",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.MachineTypes[0].Spec.GPU.Family = ""
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "Only one GPU family, product or machine cane be set",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.MachineTypes[1].Spec.GPU.Family = "ampere"
					fakeMachine.Spec.MachineTypes[1].Spec.GPU.Machine = "DGX-1"
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "Support only one machineType in NodePool",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.NodePool[0].MachineType = append(fakeMachine.Spec.NodePool[0].MachineType, NodePoolMachineType{
						Name: "test-machine2",
					})
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "Failed to find machineType using in nodePool",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.NodePool[0].MachineType = []NodePoolMachineType{{
						Name: "missing-machine",
					}}
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "MachineType name is duplicated",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					duplicatedMachineType := fakeMachine.Spec.MachineTypes[0]
					fakeMachine.Spec.MachineTypes = append(fakeMachine.Spec.MachineTypes, duplicatedMachineType)
					return fakeMachine
				}(),
				err: true,
			},
			{
				description: "Not specified GPU",
				fakeMachine: func() *Machine {
					fakeMachine := newFakeMachine()
					fakeMachine.Spec.MachineTypes[1].Spec.GPU = nil
					return fakeMachine
				}(),
				err: false,
			},
		}

		for _, test := range testCases {
			By(test.description)
			if len(test.kubeResources) > 0 {
				for _, o := range test.kubeResources {
					Expect(k8sClient.Create(ctx, o, &client.CreateOptions{})).NotTo(HaveOccurred(), test.description)
				}
			}
			err := k8sClient.Create(ctx, test.fakeMachine, &client.CreateOptions{})
			if test.err {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(k8sClient.DeleteAllOf(ctx, &Machine{}, &client.DeleteAllOfOptions{})).NotTo(HaveOccurred())
			if len(test.kubeResources) > 0 {
				for _, o := range test.kubeResources {
					Expect(k8sClient.DeleteAllOf(ctx, o, &client.DeleteAllOfOptions{})).NotTo(HaveOccurred())
				}
			}
		}
	})
})
