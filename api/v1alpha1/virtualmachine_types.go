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

// VirtualMachine phase constants
const (
	PhaseRunning = "Running"
	PhasePending = "Pending"
)

// VirtualMachineSpec defines the desired state of VirtualMachine
type VirtualMachineSpec struct {
	// CPUs is the number of CPUs for the VM
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	CPUs int32 `json:"cpus,omitempty"`

	// Memory is the amount of memory for the VM (e.g., "2Gi")
	// +kubebuilder:default="1Gi"
	Memory string `json:"memory,omitempty"`

	// DiskSize is the size of the persistent disk (e.g., "10Gi")
	// +kubebuilder:default="10Gi"
	DiskSize string `json:"diskSize,omitempty"`

	// OS is the operating system for the VM
	// +kubebuilder:validation:Enum=ubuntu;fedora;debian;centos;alpine;cirros;freebsd
	// +kubebuilder:validation:Required
	OS string `json:"os"`

	// OSVersion is the version of the OS (optional, uses latest if not specified)
	// +optional
	OSVersion string `json:"osVersion,omitempty"`

	// CloudInit is the cloud-init user data
	// +optional
	CloudInit string `json:"cloudInit,omitempty"`

	// SSHKeys is a list of SSH public keys to inject
	// +optional
	SSHKeys []string `json:"sshKeys,omitempty"`

	// RunStrategy defines the VM run strategy (Always, RerunOnFailure, Manual, Halted)
	// +kubebuilder:validation:Enum=Always;RerunOnFailure;Manual;Halted
	// +kubebuilder:default=Always
	RunStrategy string `json:"runStrategy,omitempty"`

	// StorageClass is the storage class for the VM disk
	// +optional
	StorageClass string `json:"storageClass,omitempty"`
}

// VirtualMachineStatus defines the observed state of VirtualMachine
type VirtualMachineStatus struct {
	// Phase is the current phase of the VM (Pending, Running, Stopped, Failed)
	// +optional
	Phase string `json:"phase,omitempty"`

	// Node is the node where the VM is running
	// +optional
	Node string `json:"node,omitempty"`

	// IPAddress is the IP address of the VM
	// +optional
	IPAddress string `json:"ipAddress,omitempty"`

	// Ready indicates if the VM is ready
	// +optional
	Ready bool `json:"ready,omitempty"`

	// Conditions represent the current state of the VirtualMachine resource
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Node",type="string",JSONPath=".status.node"
// +kubebuilder:printcolumn:name="IP",type="string",JSONPath=".status.ipAddress"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// VirtualMachine is the Schema for the virtualmachines API
// A VirtualMachine represents a KubeVirt virtual machine
type VirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineSpec   `json:"spec,omitempty"`
	Status VirtualMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VirtualMachineList contains a list of VirtualMachine
type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualMachine{}, &VirtualMachineList{})
}

// OSImageMap maps OS types to their container disk images
var OSImageMap = map[string]string{
	"ubuntu":  "quay.io/containerdisks/ubuntu:22.04",
	"fedora":  "quay.io/containerdisks/fedora:39",
	"debian":  "quay.io/containerdisks/debian:12",
	"centos":  "quay.io/containerdisks/centos-stream:9",
	"alpine":  "quay.io/containerdisks/alpine:3.19",
	"cirros":  "quay.io/kubevirt/cirros-container-disk-demo:latest",
	"freebsd": "quay.io/containerdisks/freebsd:13.2",
}

// GetImageForOS returns the container disk image for a given OS and optional version
func GetImageForOS(os, version string) string {
	if version != "" {
		// Check if versioned image exists
		versionedKey := os + ":" + version
		if img, ok := OSImageMap[versionedKey]; ok {
			return img
		}
		// Return custom versioned image
		baseImage := OSImageMap[os]
		if baseImage != "" {
			// Replace version tag
			parts := splitImageTag(baseImage)
			return parts[0] + ":" + version
		}
	}
	// Return default image for OS
	if img, ok := OSImageMap[os]; ok {
		return img
	}
	// Fallback to cirros for unknown OS
	return OSImageMap["cirros"]
}

// splitImageTag splits image into repo and tag parts
func splitImageTag(image string) []string {
	parts := make([]string, 2)
	idx := -1
	for i := len(image) - 1; i >= 0; i-- {
		if image[i] == ':' {
			idx = i
			break
		}
	}
	if idx > 0 {
		parts[0] = image[:idx]
		parts[1] = image[idx+1:]
	} else {
		parts[0] = image
		parts[1] = "latest"
	}
	return parts
}
