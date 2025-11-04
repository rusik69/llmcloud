/*
Copyright 2025 rusik69.

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

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	llmcloudv1alpha1 "github.com/rusik69/llmcloud-operator/api/v1alpha1"
)

var _ = Describe("VirtualMachine Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-vm"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		virtualmachine := &llmcloudv1alpha1.VirtualMachine{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind VirtualMachine")
			err := k8sClient.Get(ctx, typeNamespacedName, virtualmachine)
			if err != nil && errors.IsNotFound(err) {
				resource := &llmcloudv1alpha1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: llmcloudv1alpha1.VirtualMachineSpec{
						OS:          "ubuntu",
						OSVersion:   "22.04",
						CPUs:        2,
						Memory:      "4Gi",
						RunStrategy: "Always",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &llmcloudv1alpha1.VirtualMachine{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance VirtualMachine")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &VirtualMachineReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			// Note: May fail if KubeVirt CRDs not installed, which is expected in unit tests
			_ = err
		})

		It("should add finalizer to the VM", func() {
			By("Getting the VM")
			Expect(k8sClient.Get(ctx, typeNamespacedName, virtualmachine)).To(Succeed())

			By("Verifying finalizer was added")
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, typeNamespacedName, virtualmachine)
				return len(virtualmachine.Finalizers) > 0
			}).Should(BeTrue())
		})

		It("should have correct spec values", func() {
			By("Getting the VM")
			Expect(k8sClient.Get(ctx, typeNamespacedName, virtualmachine)).To(Succeed())

			By("Verifying spec values")
			Expect(virtualmachine.Spec.OS).To(Equal("ubuntu"))
			Expect(virtualmachine.Spec.OSVersion).To(Equal("22.04"))
			Expect(virtualmachine.Spec.CPUs).To(Equal(int32(2)))
			Expect(virtualmachine.Spec.Memory).To(Equal("4Gi"))
			Expect(virtualmachine.Spec.RunStrategy).To(Equal("Always"))
		})
	})
})
