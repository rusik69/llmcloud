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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectMember defines a member of the project
type ProjectMember struct {
	// Username is the Kubernetes user or service account name
	// +kubebuilder:validation:Required
	Username string `json:"username"`

	// Role is the role of the member (owner, admin, developer, viewer)
	// +kubebuilder:validation:Enum=owner;admin;developer;viewer
	// +kubebuilder:default=viewer
	Role string `json:"role"`
}

// ProjectSpec defines the desired state of Project
type ProjectSpec struct {
	// Description is a human-readable description of the project
	// +optional
	Description string `json:"description,omitempty"`

	// Members is a list of project members with their roles
	// +optional
	Members []ProjectMember `json:"members,omitempty"`

	// ResourceQuotas defines resource limits for the project
	// +optional
	ResourceQuotas *ProjectResourceQuotas `json:"resourceQuotas,omitempty"`
}

// ProjectResourceQuotas defines resource quotas for a project
type ProjectResourceQuotas struct {
	// MaxVMs is the maximum number of VMs allowed
	// +optional
	MaxVMs *int32 `json:"maxVMs,omitempty"`

	// MaxLLMModels is the maximum number of LLM models allowed
	// +optional
	MaxLLMModels *int32 `json:"maxLLMModels,omitempty"`

	// MaxCPU is the maximum CPU allocation (e.g., "10")
	// +optional
	MaxCPU *string `json:"maxCPU,omitempty"`

	// MaxMemory is the maximum memory allocation (e.g., "20Gi")
	// +optional
	MaxMemory *string `json:"maxMemory,omitempty"`
}

// ProjectStatus defines the observed state of Project
type ProjectStatus struct {
	// Namespace is the Kubernetes namespace created for this project
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Phase represents the current phase of the project
	// +optional
	Phase string `json:"phase,omitempty"`

	// VMCount is the current number of VMs in the project
	// +optional
	VMCount int32 `json:"vmCount,omitempty"`

	// LLMModelCount is the current number of LLM models in the project
	// +optional
	LLMModelCount int32 `json:"llmModelCount,omitempty"`

	// ServiceCount is the current number of services in the project
	// +optional
	ServiceCount int32 `json:"serviceCount,omitempty"`

	// Conditions represent the current state of the Project resource
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Namespace",type="string",JSONPath=".status.namespace"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="VMs",type="integer",JSONPath=".status.vmCount"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Project is the Schema for the projects API
// A Project represents a multi-tenant workspace with its own namespace and RBAC rules
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectList contains a list of Project
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
