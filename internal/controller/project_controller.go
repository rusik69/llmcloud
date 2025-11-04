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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	llmcloudv1alpha1 "github.com/rusik69/llmcloud-operator/api/v1alpha1"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=llmcloud.llmcloud.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=llmcloud.llmcloud.io,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=llmcloud.llmcloud.io,resources=projects/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete

const (
	projectFinalizer = "llmcloud.llmcloud.io/finalizer"
)

func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	project := &llmcloudv1alpha1.Project{}
	if err := r.Get(ctx, req.NamespacedName, project); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !project.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(project, projectFinalizer) {
			if err := r.finalizeProject(ctx, project); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(project, projectFinalizer)
			return ctrl.Result{}, r.Update(ctx, project)
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(project, projectFinalizer) {
		controllerutil.AddFinalizer(project, projectFinalizer)
		return ctrl.Result{Requeue: true}, r.Update(ctx, project)
	}

	namespace := fmt.Sprintf("project-%s", project.Name)
	if err := r.reconcileNamespace(ctx, project, namespace); err != nil {
		log.Error(err, "Failed to reconcile namespace")
		r.updateStatus(ctx, project, "Error", err.Error())
		return ctrl.Result{}, err
	}

	if err := r.reconcileRBAC(ctx, project, namespace); err != nil {
		log.Error(err, "Failed to reconcile RBAC")
		r.updateStatus(ctx, project, "Error", err.Error())
		return ctrl.Result{}, err
	}

	project.Status.Namespace = namespace
	project.Status.Phase = "Active"
	meta.SetStatusCondition(&project.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "ProjectReady",
		Message:            "Project is ready",
		ObservedGeneration: project.Generation,
	})

	return ctrl.Result{}, r.Status().Update(ctx, project)
}

func (r *ProjectReconciler) reconcileNamespace(ctx context.Context, project *llmcloudv1alpha1.Project, namespace string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"llmcloud.io/project": project.Name,
				"llmcloud.io/managed": "true",
			},
		},
	}

	if err := controllerutil.SetControllerReference(project, ns, r.Scheme); err != nil {
		return err
	}

	existingNS := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{Name: namespace}, existingNS); err != nil {
		if errors.IsNotFound(err) {
			return r.Create(ctx, ns)
		}
		return err
	}

	existingNS.Labels = ns.Labels
	return r.Update(ctx, existingNS)
}

func (r *ProjectReconciler) reconcileRBAC(ctx context.Context, project *llmcloudv1alpha1.Project, namespace string) error {
	for _, member := range project.Spec.Members {
		rb := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", project.Name, member.Username),
				Namespace: namespace,
				Labels: map[string]string{
					"llmcloud.io/project": project.Name,
					"llmcloud.io/managed": "true",
				},
			},
			Subjects: []rbacv1.Subject{{Kind: "User", Name: member.Username}},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     r.getRoleForMember(member.Role),
			},
		}

		existingRB := &rbacv1.RoleBinding{}
		if err := r.Get(ctx, client.ObjectKey{Name: rb.Name, Namespace: namespace}, existingRB); err != nil {
			if errors.IsNotFound(err) {
				if err := r.Create(ctx, rb); err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			existingRB.Subjects = rb.Subjects
			existingRB.RoleRef = rb.RoleRef
			if err := r.Update(ctx, existingRB); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *ProjectReconciler) getRoleForMember(role string) string {
	roleMap := map[string]string{"owner": "admin", "admin": "admin", "developer": "edit"}
	if r, ok := roleMap[role]; ok {
		return r
	}
	return "view"
}

func (r *ProjectReconciler) finalizeProject(ctx context.Context, project *llmcloudv1alpha1.Project) error {
	return nil // Namespace auto-deleted by garbage collection via owner reference
}

func (r *ProjectReconciler) updateStatus(ctx context.Context, project *llmcloudv1alpha1.Project, phase, message string) {
	project.Status.Phase = phase
	meta.SetStatusCondition(&project.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             "ReconciliationError",
		Message:            message,
		ObservedGeneration: project.Generation,
	})
	_ = r.Status().Update(ctx, project)
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&llmcloudv1alpha1.Project{}).Named("project").Complete(r)
}
