package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestProjectSpec(t *testing.T) {
	tests := []struct {
		name string
		spec ProjectSpec
		want int
	}{
		{
			name: "empty members",
			spec: ProjectSpec{},
			want: 0,
		},
		{
			name: "with members",
			spec: ProjectSpec{
				Members: []ProjectMember{
					{Username: "user1", Role: "admin"},
					{Username: "user2", Role: "viewer"},
				},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := len(tt.spec.Members); got != tt.want {
				t.Errorf("ProjectSpec members count = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProjectCreation(t *testing.T) {
	project := &Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
		Spec: ProjectSpec{
			Description: "Test project",
			Members: []ProjectMember{
				{Username: "admin", Role: "owner"},
			},
		},
	}

	if project.Name != "test-project" {
		t.Errorf("Expected project name 'test-project', got '%s'", project.Name)
	}

	if len(project.Spec.Members) != 1 {
		t.Errorf("Expected 1 member, got %d", len(project.Spec.Members))
	}

	if project.Spec.Members[0].Role != "owner" {
		t.Errorf("Expected role 'owner', got '%s'", project.Spec.Members[0].Role)
	}
}

func TestProjectResourceQuotas(t *testing.T) {
	maxVMs := int32(10)
	maxModels := int32(5)
	maxCPU := "20"
	maxMemory := "40Gi"

	quotas := &ProjectResourceQuotas{
		MaxVMs:       &maxVMs,
		MaxLLMModels: &maxModels,
		MaxCPU:       &maxCPU,
		MaxMemory:    &maxMemory,
	}

	if *quotas.MaxVMs != 10 {
		t.Errorf("Expected MaxVMs 10, got %d", *quotas.MaxVMs)
	}

	if *quotas.MaxLLMModels != 5 {
		t.Errorf("Expected MaxLLMModels 5, got %d", *quotas.MaxLLMModels)
	}
}

func TestProjectStatus(t *testing.T) {
	status := ProjectStatus{
		Namespace:     "project-test",
		Phase:         "Active",
		VMCount:       3,
		LLMModelCount: 2,
		ServiceCount:  1,
	}

	if status.Namespace != "project-test" {
		t.Errorf("Expected namespace 'project-test', got '%s'", status.Namespace)
	}

	if status.VMCount != 3 {
		t.Errorf("Expected VMCount 3, got %d", status.VMCount)
	}

	totalResources := status.VMCount + status.LLMModelCount + status.ServiceCount
	if totalResources != 6 {
		t.Errorf("Expected total resources 6, got %d", totalResources)
	}
}

func TestProjectList(t *testing.T) {
	list := &ProjectList{
		Items: []Project{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "project1"},
				Spec:       ProjectSpec{Description: "First project"},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "project2"},
				Spec:       ProjectSpec{Description: "Second project"},
			},
		},
	}

	if len(list.Items) != 2 {
		t.Errorf("Expected 2 items in list, got %d", len(list.Items))
	}

	if list.Items[0].Name != "project1" {
		t.Errorf("Expected first project name 'project1', got '%s'", list.Items[0].Name)
	}
}

func TestProjectMember(t *testing.T) {
	member := ProjectMember{
		Username: "testuser",
		Role:     "admin",
	}

	if member.Username == "" {
		t.Error("Username should not be empty")
	}

	if member.Role == "" {
		t.Error("Role should not be empty")
	}

	validRoles := map[string]bool{
		"owner": true, "admin": true, "developer": true, "viewer": true,
	}

	if !validRoles[member.Role] {
		t.Errorf("Role '%s' is not valid", member.Role)
	}
}

func TestProjectWithMultipleMembers(t *testing.T) {
	project := &Project{
		ObjectMeta: metav1.ObjectMeta{Name: "team-project"},
		Spec: ProjectSpec{
			Description: "Team project",
			Members: []ProjectMember{
				{Username: "owner", Role: "owner"},
				{Username: "dev1", Role: "developer"},
				{Username: "dev2", Role: "developer"},
				{Username: "viewer", Role: "viewer"},
			},
		},
	}

	if len(project.Spec.Members) != 4 {
		t.Errorf("Expected 4 members, got %d", len(project.Spec.Members))
	}

	// Count by role
	roleCount := make(map[string]int)
	for _, member := range project.Spec.Members {
		roleCount[member.Role]++
	}

	if roleCount["developer"] != 2 {
		t.Errorf("Expected 2 developers, got %d", roleCount["developer"])
	}
}

func TestProjectWithResourceQuotas(t *testing.T) {
	maxVMs := int32(5)
	maxModels := int32(3)
	maxCPU := "10"
	maxMemory := "20Gi"

	project := &Project{
		ObjectMeta: metav1.ObjectMeta{Name: "quota-project"},
		Spec: ProjectSpec{
			ResourceQuotas: &ProjectResourceQuotas{
				MaxVMs:       &maxVMs,
				MaxLLMModels: &maxModels,
				MaxCPU:       &maxCPU,
				MaxMemory:    &maxMemory,
			},
		},
	}

	if project.Spec.ResourceQuotas == nil {
		t.Fatal("ResourceQuotas should not be nil")
	}

	if *project.Spec.ResourceQuotas.MaxVMs != 5 {
		t.Errorf("Expected MaxVMs 5, got %d", *project.Spec.ResourceQuotas.MaxVMs)
	}
}
