package v1alpha1

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGroupVersion(t *testing.T) {
	if GroupVersion.Group != "llmcloud.llmcloud.io" {
		t.Errorf("Expected group 'llmcloud.llmcloud.io', got '%s'", GroupVersion.Group)
	}

	if GroupVersion.Version != "v1alpha1" {
		t.Errorf("Expected version 'v1alpha1', got '%s'", GroupVersion.Version)
	}
}

func TestSchemeBuilder(t *testing.T) {
	if SchemeBuilder == nil {
		t.Fatal("SchemeBuilder should not be nil")
	}

	if SchemeBuilder.GroupVersion != GroupVersion {
		t.Error("SchemeBuilder GroupVersion should match GroupVersion")
	}
}

func TestAddToScheme(t *testing.T) {
	scheme := runtime.NewScheme()

	if err := AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme failed: %v", err)
	}

	// Verify all types are registered
	gvk := schema.GroupVersionKind{
		Group:   "llmcloud.llmcloud.io",
		Version: "v1alpha1",
		Kind:    "Project",
	}

	obj, err := scheme.New(gvk)
	if err != nil {
		t.Errorf("Failed to create Project from scheme: %v", err)
	}

	if _, ok := obj.(*Project); !ok {
		t.Error("Created object is not a Project")
	}
}

func TestAllTypesRegistered(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme failed: %v", err)
	}

	types := []struct {
		kind string
		obj  runtime.Object
	}{
		{"Project", &Project{}},
		{"ProjectList", &ProjectList{}},
		{"VirtualMachine", &VirtualMachine{}},
		{"VirtualMachineList", &VirtualMachineList{}},
		{"LLMModel", &LLMModel{}},
		{"LLMModelList", &LLMModelList{}},
		{"Service", &Service{}},
		{"ServiceList", &ServiceList{}},
	}

	for _, tt := range types {
		t.Run(tt.kind, func(t *testing.T) {
			gvk := schema.GroupVersionKind{
				Group:   GroupVersion.Group,
				Version: GroupVersion.Version,
				Kind:    tt.kind,
			}

			_, err := scheme.New(gvk)
			if err != nil {
				t.Errorf("Type %s not properly registered: %v", tt.kind, err)
			}
		})
	}
}

func TestGroupVersionString(t *testing.T) {
	expected := "llmcloud.llmcloud.io/v1alpha1"
	if got := GroupVersion.String(); got != expected {
		t.Errorf("GroupVersion.String() = %v, want %v", got, expected)
	}
}
