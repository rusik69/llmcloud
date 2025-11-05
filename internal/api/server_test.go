package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	llmcloudv1alpha1 "github.com/rusik69/llmcloud-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func setupTestClient() client.Client {
	scheme := runtime.NewScheme()
	_ = llmcloudv1alpha1.AddToScheme(scheme)
	return fake.NewClientBuilder().WithScheme(scheme).Build()
}

func TestNewServer(t *testing.T) {
	c := setupTestClient()
	server := NewServer(c)

	if server == nil {
		t.Fatal("Expected non-nil server")
	}

	if server.client == nil {
		t.Fatal("Expected non-nil client")
	}
}

func TestCorsMiddleware(t *testing.T) {
	s := &Server{client: setupTestClient()}
	handler := s.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS header not set correctly")
	}
}

func TestCorsOptionsRequest(t *testing.T) {
	s := &Server{client: setupTestClient()}
	handler := s.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK for OPTIONS, got %d", w.Code)
	}
}

func TestHandleProjectsGet(t *testing.T) {
	c := setupTestClient()
	s := &Server{client: c}

	// Create test project
	project := &llmcloudv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "test-project"},
		Spec:       llmcloudv1alpha1.ProjectSpec{Description: "Test"},
	}
	_ = c.Create(context.Background(), project)

	req := httptest.NewRequest("GET", "/api/v1/projects", nil)
	w := httptest.NewRecorder()

	s.handleProjects(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	var result llmcloudv1alpha1.ProjectList
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result.Items) != 1 {
		t.Errorf("Expected 1 project, got %d", len(result.Items))
	}
}

func TestHandleProjectsPost(t *testing.T) {
	s := &Server{client: setupTestClient()}

	project := llmcloudv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "new-project"},
		Spec: llmcloudv1alpha1.ProjectSpec{
			Description: "New project",
			Members: []llmcloudv1alpha1.ProjectMember{
				{Username: "testuser", Role: "owner"},
			},
		},
	}

	body, _ := json.Marshal(project)
	req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewReader(body))
	w := httptest.NewRecorder()

	s.handleProjects(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandleProjectsInvalidMethod(t *testing.T) {
	s := &Server{client: setupTestClient()}

	req := httptest.NewRequest("PUT", "/api/v1/projects", nil)
	w := httptest.NewRecorder()

	s.handleProjects(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status MethodNotAllowed, got %d", w.Code)
	}
}

func TestHandleProjectGet(t *testing.T) {
	c := setupTestClient()
	s := &Server{client: c}

	project := &llmcloudv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "test-project"},
		Spec:       llmcloudv1alpha1.ProjectSpec{Description: "Test"},
	}
	_ = c.Create(context.Background(), project)

	req := httptest.NewRequest("GET", "/api/v1/projects/test-project", nil)
	w := httptest.NewRecorder()

	s.handleProject(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}
}

func TestHandleProjectDelete(t *testing.T) {
	c := setupTestClient()
	s := &Server{client: c}

	project := &llmcloudv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "test-project"},
		Spec:       llmcloudv1alpha1.ProjectSpec{Description: "Test"},
	}
	_ = c.Create(context.Background(), project)

	req := httptest.NewRequest("DELETE", "/api/v1/projects/test-project", nil)
	w := httptest.NewRecorder()

	s.handleProject(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status NoContent, got %d", w.Code)
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		count int
	}{
		{
			name:  "simple path",
			path:  "namespace/resource/name",
			count: 3,
		},
		{
			name:  "path with trailing slash",
			path:  "namespace/resource/",
			count: 3, // splitPath keeps empty segments
		},
		{
			name:  "single segment",
			path:  "namespace",
			count: 1,
		},
		{
			name:  "empty path",
			path:  "",
			count: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitPath(tt.path)
			if len(got) != tt.count {
				t.Errorf("splitPath(%q) returned %d segments, want %d; got: %v", tt.path, len(got), tt.count, got)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	s := &Server{client: setupTestClient()}
	w := httptest.NewRecorder()

	data := map[string]string{"key": "value"}
	s.writeJSON(w, data)

	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Content-Type header not set to application/json")
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	if result["key"] != "value" {
		t.Errorf("Expected key='value', got key='%s'", result["key"])
	}
}

func TestHandleNamespaceResourcesInvalidPath(t *testing.T) {
	s := &Server{client: setupTestClient()}

	req := httptest.NewRequest("GET", "/api/v1/namespaces/", nil)
	w := httptest.NewRecorder()

	s.handleNamespaceResources(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}

func TestHandleNamespaceResourcesUnknownResource(t *testing.T) {
	s := &Server{client: setupTestClient()}

	req := httptest.NewRequest("GET", "/api/v1/namespaces/default/unknown", nil)
	w := httptest.NewRecorder()

	s.handleNamespaceResources(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status NotFound, got %d", w.Code)
	}
}

func TestHandleVMsGet(t *testing.T) {
	c := setupTestClient()
	s := &Server{client: c}

	vm := &llmcloudv1alpha1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vm",
			Namespace: "default",
		},
		Spec: llmcloudv1alpha1.VirtualMachineSpec{
			OS:        "ubuntu",
			OSVersion: "22.04",
			CPUs:      2,
			Memory:    "4Gi",
		},
	}
	_ = c.Create(context.Background(), vm)

	req := httptest.NewRequest("GET", "/api/v1/namespaces/default/vms", nil)
	w := httptest.NewRecorder()

	s.handleNamespaceResources(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	var result llmcloudv1alpha1.VirtualMachineList
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result.Items) != 1 {
		t.Errorf("Expected 1 VM, got %d", len(result.Items))
	}
}

func TestHandleVMsGetSingle(t *testing.T) {
	c := setupTestClient()
	s := &Server{client: c}

	vm := &llmcloudv1alpha1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vm",
			Namespace: "default",
		},
		Spec: llmcloudv1alpha1.VirtualMachineSpec{
			OS:        "ubuntu",
			OSVersion: "22.04",
			CPUs:      2,
			Memory:    "4Gi",
		},
	}
	_ = c.Create(context.Background(), vm)

	req := httptest.NewRequest("GET", "/api/v1/namespaces/default/vms/test-vm", nil)
	w := httptest.NewRecorder()

	s.handleNamespaceResources(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	var result llmcloudv1alpha1.VirtualMachine
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Name != "test-vm" {
		t.Errorf("Expected VM name 'test-vm', got '%s'", result.Name)
	}
}

func TestHandleVMsPost(t *testing.T) {
	s := &Server{client: setupTestClient()}

	vm := llmcloudv1alpha1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "new-vm",
			Namespace: "default",
		},
		Spec: llmcloudv1alpha1.VirtualMachineSpec{
			OS:     "fedora",
			CPUs:   4,
			Memory: "8Gi",
		},
	}

	body, _ := json.Marshal(vm)
	req := httptest.NewRequest("POST", "/api/v1/namespaces/default/vms", bytes.NewReader(body))
	w := httptest.NewRecorder()

	s.handleNamespaceResources(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}
}

func TestHandleVMsDelete(t *testing.T) {
	c := setupTestClient()
	s := &Server{client: c}

	vm := &llmcloudv1alpha1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vm",
			Namespace: "default",
		},
		Spec: llmcloudv1alpha1.VirtualMachineSpec{
			OS:        "ubuntu",
			OSVersion: "22.04",
			CPUs:      2,
			Memory:    "4Gi",
		},
	}
	_ = c.Create(context.Background(), vm)

	req := httptest.NewRequest("DELETE", "/api/v1/namespaces/default/vms/test-vm", nil)
	w := httptest.NewRecorder()

	s.handleNamespaceResources(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status NoContent, got %d", w.Code)
	}
}

func TestHandleModelsGet(t *testing.T) {
	c := setupTestClient()
	s := &Server{client: c}

	model := &llmcloudv1alpha1.LLMModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-model",
			Namespace: "default",
		},
		Spec: llmcloudv1alpha1.LLMModelSpec{
			ModelName: "llama2",
			Replicas:  1,
		},
	}
	_ = c.Create(context.Background(), model)

	req := httptest.NewRequest("GET", "/api/v1/namespaces/default/models", nil)
	w := httptest.NewRecorder()

	s.handleNamespaceResources(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	var result llmcloudv1alpha1.LLMModelList
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result.Items) != 1 {
		t.Errorf("Expected 1 model, got %d", len(result.Items))
	}
}

func TestHandleServicesGet(t *testing.T) {
	c := setupTestClient()
	s := &Server{client: c}

	service := &llmcloudv1alpha1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
		Spec: llmcloudv1alpha1.ServiceSpec{
			Type:  "postgresql",
			Image: "postgres:14.0",
		},
	}
	_ = c.Create(context.Background(), service)

	req := httptest.NewRequest("GET", "/api/v1/namespaces/default/services", nil)
	w := httptest.NewRecorder()

	s.handleNamespaceResources(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	var result llmcloudv1alpha1.ServiceList
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result.Items) != 1 {
		t.Errorf("Expected 1 service, got %d", len(result.Items))
	}
}

func TestHandleResourceInvalidMethod(t *testing.T) {
	s := &Server{client: setupTestClient()}

	req := httptest.NewRequest("PATCH", "/api/v1/namespaces/default/vms", nil)
	w := httptest.NewRecorder()

	s.handleNamespaceResources(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status MethodNotAllowed, got %d", w.Code)
	}
}

func TestHandleResourceInvalidJSON(t *testing.T) {
	s := &Server{client: setupTestClient()}

	req := httptest.NewRequest("POST", "/api/v1/namespaces/default/vms", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	s.handleNamespaceResources(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}

func TestHandleResourceNotFound(t *testing.T) {
	s := &Server{client: setupTestClient()}

	req := httptest.NewRequest("GET", "/api/v1/namespaces/default/vms/nonexistent", nil)
	w := httptest.NewRecorder()

	s.handleNamespaceResources(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status NotFound, got %d", w.Code)
	}
}

func TestHandleProjectGetNotFound(t *testing.T) {
	s := &Server{client: setupTestClient()}

	req := httptest.NewRequest("GET", "/api/v1/projects/nonexistent", nil)
	w := httptest.NewRecorder()

	s.handleProject(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status NotFound, got %d", w.Code)
	}
}

func TestHandleProjectsPostInvalidJSON(t *testing.T) {
	s := &Server{client: setupTestClient()}

	req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewReader([]byte("invalid")))
	w := httptest.NewRecorder()

	s.handleProjects(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status BadRequest, got %d", w.Code)
	}
}
