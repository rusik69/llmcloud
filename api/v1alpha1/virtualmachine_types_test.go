package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVirtualMachineSpec(t *testing.T) {
	tests := []struct {
		name string
		spec VirtualMachineSpec
	}{
		{
			name: "basic vm spec",
			spec: VirtualMachineSpec{
				OS:        "ubuntu",
				OSVersion: "22.04",
				CPUs:      2,
				Memory:    "4Gi",
			},
		},
		{
			name: "vm with ssh keys",
			spec: VirtualMachineSpec{
				OS:      "fedora",
				CPUs:    4,
				Memory:  "8Gi",
				SSHKeys: []string{"ssh-rsa AAAA..."},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.spec.CPUs <= 0 {
				t.Errorf("CPUs should be positive, got %d", tt.spec.CPUs)
			}
			if tt.spec.Memory == "" {
				t.Error("Memory should not be empty")
			}
			if tt.spec.OS == "" {
				t.Error("OS should not be empty")
			}
		})
	}
}

func TestVirtualMachineCreation(t *testing.T) {
	vm := &VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vm",
			Namespace: "default",
		},
		Spec: VirtualMachineSpec{
			OS:          "ubuntu",
			OSVersion:   "22.04",
			CPUs:        2,
			Memory:      "4Gi",
			RunStrategy: "Always",
			SSHKeys:     []string{"ssh-rsa key1", "ssh-rsa key2"},
		},
	}

	if vm.Name != "test-vm" {
		t.Errorf("Expected VM name 'test-vm', got '%s'", vm.Name)
	}

	if vm.Spec.CPUs != 2 {
		t.Errorf("Expected 2 CPUs, got %d", vm.Spec.CPUs)
	}

	if len(vm.Spec.SSHKeys) != 2 {
		t.Errorf("Expected 2 SSH keys, got %d", len(vm.Spec.SSHKeys))
	}
}

func TestVirtualMachineStatus(t *testing.T) {
	status := VirtualMachineStatus{
		Phase:     "Running",
		Ready:     true,
		Node:      "worker-1",
		IPAddress: "10.244.0.5",
	}

	if status.Phase != "Running" {
		t.Errorf("Expected phase 'Running', got '%s'", status.Phase)
	}

	if !status.Ready {
		t.Error("Expected VM to be ready")
	}

	if status.Node == "" {
		t.Error("Node should not be empty for running VM")
	}

	if status.IPAddress == "" {
		t.Error("IPAddress should not be empty for running VM")
	}
}

func TestVirtualMachineList(t *testing.T) {
	list := &VirtualMachineList{
		Items: []VirtualMachine{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "default"},
				Spec:       VirtualMachineSpec{OS: "ubuntu", OSVersion: "22.04", CPUs: 2, Memory: "4Gi"},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "vm2", Namespace: "default"},
				Spec:       VirtualMachineSpec{OS: "fedora", CPUs: 4, Memory: "8Gi"},
			},
		},
	}

	if len(list.Items) != 2 {
		t.Errorf("Expected 2 VMs in list, got %d", len(list.Items))
	}
}

func TestVirtualMachineRunStrategies(t *testing.T) {
	strategies := []string{"Always", "RerunOnFailure", "Manual", "Halted"}

	for _, strategy := range strategies {
		t.Run(strategy, func(t *testing.T) {
			vm := &VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vm"},
				Spec: VirtualMachineSpec{
					OS:          "ubuntu",
					OSVersion:   "22.04",
					CPUs:        2,
					Memory:      "4Gi",
					RunStrategy: strategy,
				},
			}

			if vm.Spec.RunStrategy != strategy {
				t.Errorf("Expected RunStrategy '%s', got '%s'", strategy, vm.Spec.RunStrategy)
			}
		})
	}
}

func TestVirtualMachineWithCloudInit(t *testing.T) {
	cloudInit := `#cloud-config
users:
  - name: admin
    ssh_authorized_keys:
      - ssh-rsa AAAA...
`

	vm := &VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{Name: "cloudinit-vm"},
		Spec: VirtualMachineSpec{
			OS:        "ubuntu",
			OSVersion: "22.04",
			CPUs:      2,
			Memory:    "4Gi",
			CloudInit: cloudInit,
		},
	}

	if vm.Spec.CloudInit == "" {
		t.Error("CloudInit should not be empty")
	}

	if vm.Spec.CloudInit != cloudInit {
		t.Error("CloudInit content mismatch")
	}
}

func TestVirtualMachineWithStorageClass(t *testing.T) {
	vm := &VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{Name: "storage-vm"},
		Spec: VirtualMachineSpec{
			OS:           "ubuntu",
			OSVersion:    "22.04",
			CPUs:         2,
			Memory:       "4Gi",
			DiskSize:     "50Gi",
			StorageClass: "fast-ssd",
		},
	}

	if vm.Spec.StorageClass != "fast-ssd" {
		t.Errorf("Expected StorageClass 'fast-ssd', got '%s'", vm.Spec.StorageClass)
	}

	if vm.Spec.DiskSize != "50Gi" {
		t.Errorf("Expected DiskSize '50Gi', got '%s'", vm.Spec.DiskSize)
	}
}

func TestVirtualMachinePhases(t *testing.T) {
	phases := []string{"Pending", "Running", "Stopped", "Failed"}

	for _, phase := range phases {
		t.Run(phase, func(t *testing.T) {
			status := VirtualMachineStatus{
				Phase: phase,
				Ready: phase == "Running",
			}

			if status.Phase != phase {
				t.Errorf("Expected phase '%s', got '%s'", phase, status.Phase)
			}

			if phase == "Running" && !status.Ready {
				t.Error("Running VM should be ready")
			}
		})
	}
}
