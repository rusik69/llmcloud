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

// LLMModel phase constants
const (
	LLMModelPhasePending = "Pending"
)

// LLMModelSpec defines the desired state of LLMModel
type LLMModelSpec struct {
	// ModelName is the name of the model (e.g., "llama2", "mistral")
	ModelName string `json:"modelName"`

	// ModelSize is the size variant (e.g., "7b", "13b", "70b")
	// +optional
	ModelSize string `json:"modelSize,omitempty"`

	// Provider is the model provider (e.g., "ollama", "huggingface")
	// +optional
	Provider string `json:"provider,omitempty"`

	// Quantization level (e.g., "q4_0", "q8_0")
	// +optional
	Quantization string `json:"quantization,omitempty"`

	// Image is the container image to use for running the model
	// +optional
	Image string `json:"image,omitempty"`

	// Resources defines resource requirements
	// +optional
	Resources ResourceRequirements `json:"resources,omitempty"`

	// Replicas is the number of model instances
	// +optional
	Replicas int32 `json:"replicas,omitempty"`
}

// ResourceRequirements defines resource requirements
type ResourceRequirements struct {
	// CPU cores required
	// +optional
	CPU string `json:"cpu,omitempty"`

	// Memory required
	// +optional
	Memory string `json:"memory,omitempty"`

	// GPU devices required
	// +optional
	GPU int32 `json:"gpu,omitempty"`
}

// LLMModelStatus defines the observed state of LLMModel
type LLMModelStatus struct {
	// Phase represents the current phase of the model
	// +optional
	Phase string `json:"phase,omitempty"`

	// ReadyReplicas is the number of ready replicas
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// Endpoint is the service endpoint for accessing the model
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Conditions represent the latest available observations of the model's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced

// LLMModel is the Schema for the llmmodels API
type LLMModel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LLMModelSpec   `json:"spec,omitempty"`
	Status LLMModelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LLMModelList contains a list of LLMModel
type LLMModelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LLMModel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LLMModel{}, &LLMModelList{})
}
