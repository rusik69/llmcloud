package api

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os/exec"
	"strings"

	llmcloudv1alpha1 "github.com/rusik69/llmcloud-operator/api/v1alpha1"
	"github.com/rusik69/llmcloud-operator/internal/auth"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	client client.Client
}

func NewServer(c client.Client) *Server {
	return &Server{client: c}
}

func (s *Server) Start(addr string) error {
	// Create custom handler that checks API routes first
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle API routes
		if strings.HasPrefix(r.URL.Path, "/api/") {
			s.handleAPI(w, r)
			return
		}

		// Handle static files and SPA routing
		s.handleStatic(w, r)
	})

	log.Log.Info("Starting API server", "address", addr)
	return http.ListenAndServe(addr, s.corsMiddleware(handler))
}

func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Auth routes (no authentication required)
	if path == "/api/v1/auth/login" {
		s.handleLogin(w, r)
		return
	}

	// All other API routes require authentication
	// Extract the auth middleware logic inline
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing authorization header", http.StatusUnauthorized)
		return
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := auth.ValidateJWT(tokenString)
	if err != nil {
		http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}
	ctx := context.WithValue(r.Context(), "claims", claims)
	r = r.WithContext(ctx)

	// Route to appropriate handler
	if path == "/api/v1/users" {
		s.handleUsers(w, r)
	} else if strings.HasPrefix(path, "/api/v1/users/") {
		s.handleUser(w, r)
	} else if path == "/api/v1/projects" {
		s.handleProjects(w, r)
	} else if strings.HasPrefix(path, "/api/v1/projects/") {
		s.handleProject(w, r)
	} else if path == "/api/v1/nodes" {
		s.handleClusterNodes(w, r)
	} else if strings.HasPrefix(path, "/api/v1/nodes/") {
		s.handleNodeActions(w, r)
	} else if strings.HasPrefix(path, "/api/v1/namespaces/") {
		s.handleNamespaceResources(w, r)
	} else if strings.HasPrefix(path, "/api/v1/actions/vm/") {
		s.handleVMActions(w, r)
	} else if strings.HasPrefix(path, "/api/v1/describe/vm/") {
		s.handleVMDescribe(w, r)
	} else if strings.HasPrefix(path, "/api/v1/events/vm/") {
		s.handleVMEvents(w, r)
	} else {
		http.NotFound(w, r)
	}
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		http.Error(w, "Static files not available", http.StatusInternalServerError)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// Try to open the file
	file, err := staticFS.Open(path)
	if err != nil {
		// File doesn't exist, serve index.html for SPA routing
		indexFile, err := staticFS.Open("index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}
		defer indexFile.Close()

		stat, _ := indexFile.Stat()
		http.ServeContent(w, r, "index.html", stat.ModTime(), indexFile.(io.ReadSeeker))
		return
	}
	file.Close()

	// File exists, serve it
	fileServer := http.FileServer(http.FS(staticFS))
	fileServer.ServeHTTP(w, r)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		var projects llmcloudv1alpha1.ProjectList
		if err := s.client.List(ctx, &projects); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.writeJSON(w, projects)

	case http.MethodPost:
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		project := &llmcloudv1alpha1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: req.Name,
			},
			Spec: llmcloudv1alpha1.ProjectSpec{
				Description: req.Description,
			},
		}
		if err := s.client.Create(ctx, project); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.writeJSON(w, project)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProject(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	name := r.URL.Path[len("/api/v1/projects/"):]

	switch r.Method {
	case http.MethodGet:
		var project llmcloudv1alpha1.Project
		if err := s.client.Get(ctx, client.ObjectKey{Name: name}, &project); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		s.writeJSON(w, project)

	case http.MethodDelete:
		project := &llmcloudv1alpha1.Project{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		}
		if err := s.client.Delete(ctx, project); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleNamespaceResources(w http.ResponseWriter, r *http.Request) {
	// Parse URL: /api/v1/namespaces/{namespace}/{resource}[/{name}]
	path := r.URL.Path[len("/api/v1/namespaces/"):]
	parts := splitPath(path)
	if len(parts) < 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	namespace := parts[0]
	resource := parts[1]
	var name string
	if len(parts) > 2 {
		name = parts[2]
	}

	ctx := context.Background()

	switch resource {
	case "vms":
		s.handleVMs(ctx, w, r, namespace, name)
	case "models":
		s.handleModels(ctx, w, r, namespace, name)
	case "services":
		s.handleServices(ctx, w, r, namespace, name)
	default:
		http.Error(w, "Unknown resource", http.StatusNotFound)
	}
}

func (s *Server) handleVMs(ctx context.Context, w http.ResponseWriter, r *http.Request, namespace, name string) {
	s.handleResource(ctx, w, r, namespace, name,
		&llmcloudv1alpha1.VirtualMachine{},
		&llmcloudv1alpha1.VirtualMachineList{})
}

func (s *Server) handleModels(ctx context.Context, w http.ResponseWriter, r *http.Request, namespace, name string) {
	s.handleResource(ctx, w, r, namespace, name,
		&llmcloudv1alpha1.LLMModel{},
		&llmcloudv1alpha1.LLMModelList{})
}

func (s *Server) handleServices(ctx context.Context, w http.ResponseWriter, r *http.Request, namespace, name string) {
	s.handleResource(ctx, w, r, namespace, name,
		&llmcloudv1alpha1.Service{},
		&llmcloudv1alpha1.ServiceList{})
}

func (s *Server) handleResource(ctx context.Context, w http.ResponseWriter, r *http.Request, namespace, name string, obj client.Object, list client.ObjectList) {
	switch r.Method {
	case http.MethodGet:
		if name == "" {
			if err := s.client.List(ctx, list, client.InNamespace(namespace)); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			s.writeJSON(w, list)
		} else {
			if err := s.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, obj); err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			s.writeJSON(w, obj)
		}

	case http.MethodPost:
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(obj); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.client.Create(ctx, obj); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.writeJSON(w, obj)

	case http.MethodDelete:
		obj.SetNamespace(namespace)
		obj.SetName(name)
		if err := s.client.Delete(ctx, obj); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func splitPath(path string) []string {
	var parts []string
	for _, p := range []byte(path) {
		if p != '/' {
			if len(parts) == 0 {
				parts = append(parts, string(p))
			} else {
				parts[len(parts)-1] += string(p)
			}
		} else if len(parts) > 0 && parts[len(parts)-1] != "" {
			parts = append(parts, "")
		}
	}
	return parts
}

// handleVMActions handles VM control actions (start, stop, reboot)
// URL format: /api/v1/actions/vm/{namespace}/{name}/{action}
func (s *Server) handleVMActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse URL: /api/v1/actions/vm/{namespace}/{name}/{action}
	path := r.URL.Path[len("/api/v1/actions/vm/"):]
	parts := splitPath(path)
	if len(parts) < 3 {
		http.Error(w, "Invalid path, expected: /api/v1/actions/vm/{namespace}/{name}/{action}", http.StatusBadRequest)
		return
	}

	namespace := parts[0]
	name := parts[1]
	action := parts[2]

	ctx := context.Background()

	// Get the VM
	vm := &llmcloudv1alpha1.VirtualMachine{}
	if err := s.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, vm); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Perform action by updating RunStrategy
	switch action {
	case "start":
		vm.Spec.RunStrategy = "Always"
	case "stop":
		vm.Spec.RunStrategy = "Halted"
	case "reboot":
		// For reboot, we'll use an annotation
		if vm.Annotations == nil {
			vm.Annotations = make(map[string]string)
		}
		vm.Annotations["llmcloud.io/reboot"] = "true"
	default:
		http.Error(w, "Unknown action, valid actions: start, stop, reboot", http.StatusBadRequest)
		return
	}

	if err := s.client.Update(ctx, vm); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.writeJSON(w, map[string]string{"status": "success", "action": action})
}

// authMiddleware verifies JWT tokens
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		claims, err := auth.ValidateJWT(tokenString)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Store claims in context for use in handlers
		ctx := context.WithValue(r.Context(), "claims", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// handleLogin handles user authentication
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	user, err := auth.AuthenticateUser(ctx, s.client, loginReq.Username, loginReq.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Update last login time
	now := metav1.Now()
	user.Status.LastLoginTime = &now
	_ = s.client.Status().Update(ctx, user)

	// Generate JWT
	token, err := auth.GenerateJWT(user)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	s.writeJSON(w, map[string]interface{}{
		"token":    token,
		"username": user.Spec.Username,
		"isAdmin":  user.Spec.IsAdmin,
		"projects": user.Spec.Projects,
	})
}

// handleUsers handles user listing and creation (admin only)
func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*auth.Claims)
	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		if !claims.IsAdmin {
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}

		var users llmcloudv1alpha1.UserList
		if err := s.client.List(ctx, &users); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Remove password hashes before sending
		for i := range users.Items {
			users.Items[i].Spec.PasswordHash = ""
		}

		s.writeJSON(w, users)

	case http.MethodPost:
		if !claims.IsAdmin {
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}

		var userReq struct {
			llmcloudv1alpha1.User
			Spec struct {
				llmcloudv1alpha1.UserSpec
				Password string `json:"password,omitempty"`
			} `json:"spec"`
		}
		if err := json.NewDecoder(r.Body).Decode(&userReq); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Hash the password if provided
		if userReq.Spec.Password != "" {
			hash, err := auth.HashPassword(userReq.Spec.Password)
			if err != nil {
				http.Error(w, "Failed to hash password", http.StatusInternalServerError)
				return
			}
			userReq.User.Spec.PasswordHash = hash
		}

		// Copy other spec fields
		userReq.User.Spec.Username = userReq.Spec.Username
		userReq.User.Spec.Email = userReq.Spec.Email
		userReq.User.Spec.IsAdmin = userReq.Spec.IsAdmin
		userReq.User.Spec.Projects = userReq.Spec.Projects
		userReq.User.Spec.Disabled = userReq.Spec.Disabled

		if err := s.client.Create(ctx, &userReq.User); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Remove password hash before sending response
		userReq.User.Spec.PasswordHash = ""
		s.writeJSON(w, userReq.User)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleUser handles individual user operations (admin only)
func (s *Server) handleUser(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*auth.Claims)
	if !claims.IsAdmin {
		http.Error(w, "Admin access required", http.StatusForbidden)
		return
	}

	ctx := context.Background()
	name := r.URL.Path[len("/api/v1/users/"):]

	switch r.Method {
	case http.MethodGet:
		var user llmcloudv1alpha1.User
		if err := s.client.Get(ctx, client.ObjectKey{Name: name}, &user); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		user.Spec.PasswordHash = ""
		s.writeJSON(w, user)

	case http.MethodPut:
		var user llmcloudv1alpha1.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := s.client.Update(ctx, &user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		user.Spec.PasswordHash = ""
		s.writeJSON(w, user)

	case http.MethodDelete:
		user := &llmcloudv1alpha1.User{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		}
		if err := s.client.Delete(ctx, user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleVMDescribe returns kubectl describe output for a KubeVirt VM
// URL format: /api/v1/describe/vm/{namespace}/{name}
func (s *Server) handleVMDescribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse URL: /api/v1/describe/vm/{namespace}/{name}
	path := r.URL.Path[len("/api/v1/describe/vm/"):]
	parts := splitPath(path)
	if len(parts) < 2 {
		http.Error(w, "Invalid path, expected: /api/v1/describe/vm/{namespace}/{name}", http.StatusBadRequest)
		return
	}

	namespace := parts[0]
	name := parts[1]
	ctx := context.Background()

	// Get the KubeVirt VirtualMachine
	kvVM := &unstructured.Unstructured{}
	kvVM.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kubevirt.io",
		Version: "v1",
		Kind:    "VirtualMachine",
	})

	if err := s.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, kvVM); err != nil {
		http.Error(w, fmt.Sprintf("Failed to get KubeVirt VM: %v", err), http.StatusNotFound)
		return
	}

	// Get the VirtualMachineInstance if it exists
	vmi := &unstructured.Unstructured{}
	vmi.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kubevirt.io",
		Version: "v1",
		Kind:    "VirtualMachineInstance",
	})
	vmiExists := true
	if err := s.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, vmi); err != nil {
		vmiExists = false
	}

	// Convert to standard kubectl-style YAML (clean format)
	// Remove managed fields for cleaner output
	cleanVM := kvVM.DeepCopy()
	unstructured.RemoveNestedField(cleanVM.Object, "metadata", "managedFields")

	// Convert to JSON first, then to YAML for proper formatting
	vmJSON, err := json.MarshalIndent(cleanVM.Object, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal VM to JSON: %v", err), http.StatusInternalServerError)
		return
	}

	vmYaml, err := yaml.JSONToYAML(vmJSON)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to convert VM to YAML: %v", err), http.StatusInternalServerError)
		return
	}

	var vmiYaml []byte
	if vmiExists {
		cleanVMI := vmi.DeepCopy()
		unstructured.RemoveNestedField(cleanVMI.Object, "metadata", "managedFields")

		vmiJSON, err := json.MarshalIndent(cleanVMI.Object, "", "  ")
		if err == nil {
			vmiYaml, err = yaml.JSONToYAML(vmiJSON)
			if err != nil {
				vmiExists = false
			}
		} else {
			vmiExists = false
		}
	}

	// Build describe-style output
	describe := buildVMDescribe(kvVM, vmi, vmiExists)

	s.writeJSON(w, map[string]interface{}{
		"describe": describe,
		"yaml": map[string]interface{}{
			"vm":  string(vmYaml),
			"vmi": string(vmiYaml),
		},
	})
}

// buildVMDescribe creates a kubectl describe-style output
func buildVMDescribe(vm *unstructured.Unstructured, vmi *unstructured.Unstructured, vmiExists bool) string {
	var output strings.Builder

	// VM Header
	output.WriteString(fmt.Sprintf("Name:         %s\n", vm.GetName()))
	output.WriteString(fmt.Sprintf("Namespace:    %s\n", vm.GetNamespace()))

	// Labels
	labels := vm.GetLabels()
	if len(labels) == 0 {
		output.WriteString("Labels:       <none>\n")
	} else {
		output.WriteString("Labels:       ")
		first := true
		for k, v := range labels {
			if !first {
				output.WriteString("              ")
			}
			output.WriteString(fmt.Sprintf("%s=%s\n", k, v))
			first = false
		}
	}

	// Annotations
	annotations := vm.GetAnnotations()
	if len(annotations) == 0 {
		output.WriteString("Annotations:  <none>\n")
	} else {
		output.WriteString("Annotations:  ")
		first := true
		for k, v := range annotations {
			if !first {
				output.WriteString("              ")
			}
			output.WriteString(fmt.Sprintf("%s: %s\n", k, v))
			first = false
		}
	}
	output.WriteString(fmt.Sprintf("API Version:  %s\n", vm.GetAPIVersion()))
	output.WriteString(fmt.Sprintf("Kind:         %s\n", vm.GetKind()))

	// Metadata
	output.WriteString("\nMetadata:\n")
	output.WriteString(fmt.Sprintf("  Creation Timestamp:  %v\n", vm.GetCreationTimestamp()))
	output.WriteString(fmt.Sprintf("  Generation:          %d\n", vm.GetGeneration()))
	output.WriteString(fmt.Sprintf("  Resource Version:    %s\n", vm.GetResourceVersion()))
	output.WriteString(fmt.Sprintf("  UID:                 %s\n", vm.GetUID()))

	// Spec
	spec, _, _ := unstructured.NestedMap(vm.Object, "spec")
	output.WriteString("\nSpec:\n")
	if runStrategy, ok, _ := unstructured.NestedString(vm.Object, "spec", "runStrategy"); ok {
		output.WriteString(fmt.Sprintf("  Run Strategy:  %s\n", runStrategy))
	}
	if running, ok, _ := unstructured.NestedBool(vm.Object, "spec", "running"); ok {
		output.WriteString(fmt.Sprintf("  Running:       %t\n", running))
	}

	// Template
	if template, ok := spec["template"].(map[string]interface{}); ok {
		output.WriteString("  Template:\n")
		if spec, ok := template["spec"].(map[string]interface{}); ok {
			if domain, ok := spec["domain"].(map[string]interface{}); ok {
				output.WriteString("    Domain:\n")
				if cpu, ok := domain["cpu"].(map[string]interface{}); ok {
					output.WriteString(fmt.Sprintf("      CPU: %v\n", cpu))
				}
				if resources, ok := domain["resources"].(map[string]interface{}); ok {
					output.WriteString(fmt.Sprintf("      Resources: %v\n", resources))
				}
				if devices, ok := domain["devices"].(map[string]interface{}); ok {
					output.WriteString("      Devices:\n")
					if disks, ok := devices["disks"].([]interface{}); ok {
						output.WriteString(fmt.Sprintf("        Disks: %d disk(s)\n", len(disks)))
						for i, disk := range disks {
							output.WriteString(fmt.Sprintf("          [%d]: %v\n", i, disk))
						}
					}
					if interfaces, ok := devices["interfaces"].([]interface{}); ok {
						output.WriteString(fmt.Sprintf("        Interfaces: %d interface(s)\n", len(interfaces)))
					}
				}
			}
			if volumes, ok := spec["volumes"].([]interface{}); ok {
				output.WriteString(fmt.Sprintf("    Volumes: %d volume(s)\n", len(volumes)))
				for i, vol := range volumes {
					output.WriteString(fmt.Sprintf("      [%d]: %v\n", i, vol))
				}
			}
		}
	}

	// Status
	status, _, _ := unstructured.NestedMap(vm.Object, "status")
	output.WriteString("\nStatus:\n")
	if len(status) == 0 {
		output.WriteString("  <none>\n")
	} else {
		for key, value := range status {
			output.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
		}
	}

	// VMI Info
	if vmiExists {
		output.WriteString("\n=== VirtualMachineInstance ===\n")
		output.WriteString(fmt.Sprintf("Name:      %s\n", vmi.GetName()))
		output.WriteString(fmt.Sprintf("Namespace: %s\n", vmi.GetNamespace()))

		if phase, ok, _ := unstructured.NestedString(vmi.Object, "status", "phase"); ok {
			output.WriteString(fmt.Sprintf("Phase:     %s\n", phase))
		}
		if nodeName, ok, _ := unstructured.NestedString(vmi.Object, "status", "nodeName"); ok {
			output.WriteString(fmt.Sprintf("Node:      %s\n", nodeName))
		}

		if interfaces, ok, _ := unstructured.NestedSlice(vmi.Object, "status", "interfaces"); ok {
			output.WriteString("\nInterfaces:\n")
			for i, iface := range interfaces {
				if ifaceMap, ok := iface.(map[string]interface{}); ok {
					output.WriteString(fmt.Sprintf("  [%d]:\n", i))
					if ip, ok := ifaceMap["ipAddress"].(string); ok {
						output.WriteString(fmt.Sprintf("    IP Address: %s\n", ip))
					}
					if name, ok := ifaceMap["name"].(string); ok {
						output.WriteString(fmt.Sprintf("    Name:       %s\n", name))
					}
				}
			}
		}

		if conditions, ok, _ := unstructured.NestedSlice(vmi.Object, "status", "conditions"); ok {
			output.WriteString("\nConditions:\n")
			for _, cond := range conditions {
				if condMap, ok := cond.(map[string]interface{}); ok {
					condType := condMap["type"]
					status := condMap["status"]
					output.WriteString(fmt.Sprintf("  %s: %v\n", condType, status))
				}
			}
		}
	}

	// Events (placeholder - would need to query events separately)
	output.WriteString("\nEvents: <use kubectl get events to see events>\n")

	return output.String()
}

// handleVMEvents handles GET /api/v1/events/vm/{namespace}/{name}
func (s *Server) handleVMEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse namespace and name from path: /api/v1/events/vm/{namespace}/{name}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/events/vm/"), "/")
	if len(parts) != 2 {
		http.Error(w, "Invalid path format", http.StatusBadRequest)
		return
	}
	namespace, name := parts[0], parts[1]

	ctx := context.Background()

	// Get events related to the VM
	eventList := &unstructured.UnstructuredList{}
	eventList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "EventList",
	})

	// List all events in the namespace
	if err := s.client.List(ctx, eventList, client.InNamespace(namespace)); err != nil {
		http.Error(w, fmt.Sprintf("Failed to list events: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter events related to this VM
	type Event struct {
		Type               string `json:"type"`
		Reason             string `json:"reason"`
		Message            string `json:"message"`
		FirstTimestamp     string `json:"firstTimestamp"`
		LastTimestamp      string `json:"lastTimestamp"`
		Count              int64  `json:"count"`
		InvolvedObjectName string `json:"involvedObjectName"`
		InvolvedObjectKind string `json:"involvedObjectKind"`
		Source             string `json:"source"`
	}

	var events []Event
	for _, item := range eventList.Items {
		involvedObj, _, _ := unstructured.NestedMap(item.Object, "involvedObject")
		if involvedObj == nil {
			continue
		}

		objName, _ := involvedObj["name"].(string)
		objKind, _ := involvedObj["kind"].(string)

		// Include events for the VM and related VMI
		if objName == name && (objKind == "VirtualMachine" || objKind == "VirtualMachineInstance") {
			event := Event{
				InvolvedObjectName: objName,
				InvolvedObjectKind: objKind,
			}

			if eventType, ok, _ := unstructured.NestedString(item.Object, "type"); ok {
				event.Type = eventType
			}
			if reason, ok, _ := unstructured.NestedString(item.Object, "reason"); ok {
				event.Reason = reason
			}
			if message, ok, _ := unstructured.NestedString(item.Object, "message"); ok {
				event.Message = message
			}
			if firstTimestamp, ok, _ := unstructured.NestedString(item.Object, "firstTimestamp"); ok {
				event.FirstTimestamp = firstTimestamp
			}
			if lastTimestamp, ok, _ := unstructured.NestedString(item.Object, "lastTimestamp"); ok {
				event.LastTimestamp = lastTimestamp
			}
			if count, ok, _ := unstructured.NestedInt64(item.Object, "count"); ok {
				event.Count = count
			}
			if source, ok, _ := unstructured.NestedMap(item.Object, "source"); ok {
				if component, ok := source["component"].(string); ok {
					event.Source = component
				}
			}

			events = append(events, event)
		}
	}

	s.writeJSON(w, map[string]interface{}{
		"events": events,
	})
}

// handleClusterNodes handles GET /api/v1/nodes and POST /api/v1/nodes (cluster-wide, admin only)
// Returns actual Kubernetes nodes, not custom Node CRD
func (s *Server) handleClusterNodes(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*auth.Claims)
	if !claims.IsAdmin {
		http.Error(w, "Admin access required", http.StatusForbidden)
		return
	}

	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		// Get actual Kubernetes nodes
		nodeList := &unstructured.UnstructuredList{}
		nodeList.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "",
			Version: "v1",
			Kind:    "NodeList",
		})

		if err := s.client.List(ctx, nodeList); err != nil {
			http.Error(w, fmt.Sprintf("Failed to list nodes: %v", err), http.StatusInternalServerError)
			return
		}

		s.writeJSON(w, nodeList)

	case http.MethodPost:
		// Add a new node
		var req struct {
			Host     string `json:"host"`     // SSH host (user@host or host)
			Role     string `json:"role"`     // "master" or "worker"
			SSHKey   string `json:"sshKey"`   // Optional SSH key path
			Password string `json:"password"` // Optional SSH password
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Host == "" {
			http.Error(w, "Host is required", http.StatusBadRequest)
			return
		}

		if req.Role != "master" && req.Role != "worker" {
			http.Error(w, "Role must be 'master' or 'worker'", http.StatusBadRequest)
			return
		}

		// Execute k0s join command via SSH
		if err := s.addNode(ctx, req.Host, req.Role, req.SSHKey, req.Password); err != nil {
			http.Error(w, fmt.Sprintf("Failed to add node: %v", err), http.StatusInternalServerError)
			return
		}

		s.writeJSON(w, map[string]string{"status": "success", "message": "Node join initiated"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleNodeActions handles DELETE /api/v1/nodes/:name
func (s *Server) handleNodeActions(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*auth.Claims)
	if !claims.IsAdmin {
		http.Error(w, "Admin access required", http.StatusForbidden)
		return
	}

	// Extract node name from path: /api/v1/nodes/:name
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/nodes/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Node name required", http.StatusBadRequest)
		return
	}
	nodeName := parts[0]

	ctx := context.Background()

	switch r.Method {
	case http.MethodDelete:
		// Remove node from cluster
		if err := s.removeNode(ctx, nodeName); err != nil {
			http.Error(w, fmt.Sprintf("Failed to remove node: %v", err), http.StatusInternalServerError)
			return
		}

		s.writeJSON(w, map[string]string{"status": "success", "message": "Node removed"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// addNode adds a new node to the k0s cluster via SSH
func (s *Server) addNode(ctx context.Context, host, role, sshKey, password string) error {
	// Get k0s token from the controller
	var tokenType string
	if role == "master" {
		tokenType = "controller"
	} else {
		tokenType = "worker"
	}

	// Generate k0s token
	tokenCmd := fmt.Sprintf("sudo k0s token create --role=%s", tokenType)
	token, err := s.executeSSHCommand("", tokenCmd) // Empty host means execute locally
	if err != nil {
		return fmt.Errorf("failed to generate k0s token: %v", err)
	}

	// Install k0s on the target node
	installCmd := "curl -sSLf https://get.k0s.sh | sudo sh"
	if _, err := s.executeSSHCommand(host, installCmd); err != nil {
		return fmt.Errorf("failed to install k0s: %v", err)
	}

	// Join the cluster
	joinCmd := fmt.Sprintf("sudo k0s install %s --token='%s'", tokenType, strings.TrimSpace(token))
	if _, err := s.executeSSHCommand(host, joinCmd); err != nil {
		return fmt.Errorf("failed to join cluster: %v", err)
	}

	// Start k0s service
	startCmd := "sudo k0s start"
	if _, err := s.executeSSHCommand(host, startCmd); err != nil {
		return fmt.Errorf("failed to start k0s: %v", err)
	}

	return nil
}

// removeNode removes a node from the k0s cluster
func (s *Server) removeNode(ctx context.Context, nodeName string) error {
	// First, drain the node
	drainCmd := fmt.Sprintf("kubectl drain %s --ignore-daemonsets --delete-emptydir-data --force --timeout=60s", nodeName)
	if _, err := s.executeSSHCommand("", drainCmd); err != nil {
		return fmt.Errorf("failed to drain node: %v", err)
	}

	// Delete the node from Kubernetes
	deleteCmd := fmt.Sprintf("kubectl delete node %s", nodeName)
	if _, err := s.executeSSHCommand("", deleteCmd); err != nil {
		return fmt.Errorf("failed to delete node: %v", err)
	}

	return nil
}

// executeSSHCommand executes a command via SSH
// If host is empty, executes locally
func (s *Server) executeSSHCommand(host, command string) (string, error) {
	var cmd *exec.Cmd

	if host == "" {
		// Execute locally
		cmd = exec.Command("bash", "-c", command)
	} else {
		// Execute via SSH
		cmd = exec.Command("ssh", "-o", "StrictHostKeyChecking=no", host, command)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command failed: %v, output: %s", err, string(output))
	}

	return string(output), nil
}
