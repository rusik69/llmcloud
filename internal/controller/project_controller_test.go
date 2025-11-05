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

var _ = Describe("Project Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		project := &llmcloudv1alpha1.Project{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Project")
			resource := &llmcloudv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: llmcloudv1alpha1.ProjectSpec{
					Description: "Test project",
					Members: []llmcloudv1alpha1.ProjectMember{
						{Username: "testuser", Role: "owner"},
					},
				},
			}
			err := k8sClient.Create(ctx, resource)
			if err != nil && !errors.IsAlreadyExists(err) {
				Expect(err).NotTo(HaveOccurred())
			}
			// Wait for resource to be available
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, &llmcloudv1alpha1.Project{})
			}, "5s", "1s").Should(Succeed())
		})

		AfterEach(func() {
			By("Cleanup the specific resource instance Project")
			resource := &llmcloudv1alpha1.Project{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				// Remove finalizers to allow deletion
				resource.Finalizers = []string{}
				_ = k8sClient.Update(ctx, resource)

				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

				// Wait for deletion to complete
				Eventually(func() bool {
					err := k8sClient.Get(ctx, typeNamespacedName, &llmcloudv1alpha1.Project{})
					return errors.IsNotFound(err)
				}, "5s", "1s").Should(BeTrue())
			}
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ProjectReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Reconcile again to update status
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the project status is updated")
			Expect(k8sClient.Get(ctx, typeNamespacedName, project)).To(Succeed())
			Eventually(func() string {
				_ = k8sClient.Get(ctx, typeNamespacedName, project)
				return project.Status.Phase
			}, "5s", "1s").Should(Equal("Active"))
		})

		It("should create a namespace for the project", func() {
			By("Reconciling the project")
			controllerReconciler := &ProjectReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Reconcile again to update status
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying namespace was created")
			Eventually(func() error {
				return k8sClient.Get(ctx, typeNamespacedName, project)
			}, "5s", "1s").Should(Succeed())
			Eventually(func() string {
				_ = k8sClient.Get(ctx, typeNamespacedName, project)
				return project.Status.Namespace
			}, "5s", "1s").Should(Equal("project-" + resourceName))
		})

		It("should add finalizer to the project", func() {
			By("Reconciling to add finalizer")
			controllerReconciler := &ProjectReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Getting the project")
			Expect(k8sClient.Get(ctx, typeNamespacedName, project)).To(Succeed())

			By("Verifying finalizer was added")
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, typeNamespacedName, project)
				return len(project.Finalizers) > 0
			}, "5s", "1s").Should(BeTrue())
		})
	})

	Context("Helper functions", func() {
		It("should map roles correctly", func() {
			r := &ProjectReconciler{}

			Expect(r.getRoleForMember("owner")).To(Equal("admin"))
			Expect(r.getRoleForMember("admin")).To(Equal("admin"))
			Expect(r.getRoleForMember("developer")).To(Equal("edit"))
			Expect(r.getRoleForMember("viewer")).To(Equal("view"))
			Expect(r.getRoleForMember("unknown")).To(Equal("view"))
		})
	})
})
