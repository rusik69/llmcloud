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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// UserSpec defines the desired state of User
type UserSpec struct {
	// Username is the unique username for authentication
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:MaxLength=50
	Username string `json:"username"`

	// PasswordHash is the bcrypt hash of the user's password
	// +kubebuilder:validation:Required
	PasswordHash string `json:"passwordHash"`

	// Email is the user's email address
	// +optional
	Email string `json:"email,omitempty"`

	// IsAdmin indicates if the user has admin privileges
	// +kubebuilder:default=false
	IsAdmin bool `json:"isAdmin,omitempty"`

	// Projects is a list of project names the user has access to
	// +optional
	Projects []string `json:"projects,omitempty"`

	// Disabled indicates if the user account is disabled
	// +kubebuilder:default=false
	Disabled bool `json:"disabled,omitempty"`
}

// UserStatus defines the observed state of User.
type UserStatus struct {
	// LastLoginTime is the timestamp of the last successful login
	// +optional
	LastLoginTime *metav1.Time `json:"lastLoginTime,omitempty"`

	// conditions represent the current state of the User resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// User is the Schema for the users API
type User struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of User
	// +required
	Spec UserSpec `json:"spec"`

	// status defines the observed state of User
	// +optional
	Status UserStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// UserList contains a list of User
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
