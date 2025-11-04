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

// ServiceSpec defines the desired state of Service
type ServiceSpec struct {
	// Type is the type of service (e.g., "api", "web", "worker")
	Type string `json:"type"`

	// Image is the container image to run
	Image string `json:"image"`

	// Replicas is the desired number of replicas
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Ports define the ports to expose
	// +optional
	Ports []ServicePort `json:"ports,omitempty"`

	// Env defines environment variables
	// +optional
	Env []EnvVar `json:"env,omitempty"`

	// Resources defines resource requirements
	// +optional
	Resources ResourceRequirements `json:"resources,omitempty"`

	// Command overrides the default container command
	// +optional
	Command []string `json:"command,omitempty"`

	// Args overrides the default container args
	// +optional
	Args []string `json:"args,omitempty"`
}

// ServicePort defines a port to expose
type ServicePort struct {
	// Name is the name of the port
	// +optional
	Name string `json:"name,omitempty"`

	// Port is the port number
	Port int32 `json:"port"`

	// TargetPort is the container port (defaults to Port if not specified)
	// +optional
	TargetPort int32 `json:"targetPort,omitempty"`

	// Protocol is the protocol (TCP/UDP)
	// +optional
	Protocol string `json:"protocol,omitempty"`
}

// EnvVar represents an environment variable
type EnvVar struct {
	// Name of the environment variable
	Name string `json:"name"`

	// Value of the environment variable
	// +optional
	Value string `json:"value,omitempty"`

	// ValueFrom allows referencing a secret or configmap
	// +optional
	ValueFrom *EnvVarSource `json:"valueFrom,omitempty"`
}

// EnvVarSource represents a source for the value of an EnvVar
type EnvVarSource struct {
	// SecretKeyRef selects a key from a Secret
	// +optional
	SecretKeyRef *SecretKeySelector `json:"secretKeyRef,omitempty"`
}

// SecretKeySelector selects a key from a Secret
type SecretKeySelector struct {
	// Name of the secret
	Name string `json:"name"`

	// Key in the secret
	Key string `json:"key"`
}

// ServiceStatus defines the observed state of Service
type ServiceStatus struct {
	// Phase represents the current phase of the service
	// +optional
	Phase string `json:"phase,omitempty"`

	// ReadyReplicas is the number of ready replicas
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// Endpoint is the service endpoint
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced

// Service is the Schema for the services API
type Service struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceSpec   `json:"spec,omitempty"`
	Status ServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceList contains a list of Service
type ServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Service `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Service{}, &ServiceList{})
}
