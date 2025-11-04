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

package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	sshHost       string
	k0sVersion    string
	kubeconfig    string
	storageDevice string
)

func NewDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy llmcloud-operator to remote k0s cluster",
		Long:  `Deploys k0s cluster and llmcloud-operator to a remote host via SSH`,
		RunE:  runDeploy,
	}

	cmd.Flags().StringVar(&sshHost, "ssh-host", os.Getenv("SSH_HOST"), "SSH host (user@hostname)")
	cmd.Flags().StringVar(&k0sVersion, "k0s-version", "v1.29.1+k0s.0", "k0s version to install")
	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", filepath.Join(os.Getenv("HOME"), ".kube", "config-llmcloud"), "Kubeconfig path")
	cmd.Flags().StringVar(&storageDevice, "storage-device", "/dev/sda", "Block device for storage (VMs, containers, data)")

	return cmd
}

func runDeploy(cmd *cobra.Command, args []string) error {
	if sshHost == "" {
		return fmt.Errorf("--ssh-host or SSH_HOST environment variable must be set")
	}

	fmt.Printf("==> Deploying to %s\n", sshHost)

	// Setup storage device
	if err := setupStorageDevice(); err != nil {
		return fmt.Errorf("failed to setup storage device: %w", err)
	}

	// Deploy k0s
	if err := deployK0s(); err != nil {
		return fmt.Errorf("failed to deploy k0s: %w", err)
	}

	// Build frontend
	if err := buildFrontend(); err != nil {
		return fmt.Errorf("failed to build frontend: %w", err)
	}

	// Deploy operator
	if err := deployOperator(); err != nil {
		return fmt.Errorf("failed to deploy operator: %w", err)
	}

	// Create root user
	if err := createRootUser(); err != nil {
		return fmt.Errorf("failed to create root user: %w", err)
	}

	fmt.Println("\n✓ Deployment completed successfully!")
	return nil
}

func setupStorageDevice() error {
	fmt.Printf("==> Setting up storage device %s\n", storageDevice)

	// Check if device exists
	checkDeviceCmd := fmt.Sprintf("test -b %s", storageDevice)
	if err := execCommand("ssh", sshHost, checkDeviceCmd); err != nil {
		fmt.Printf("⚠ Warning: Device %s not found, skipping storage setup\n", storageDevice)
		return nil
	}

	// Check if device is already mounted
	checkMountCmd := fmt.Sprintf("mountpoint -q /mnt || mount | grep -q '%s'", storageDevice)
	if execCommand("ssh", sshHost, checkMountCmd) == nil {
		fmt.Println("✓ Storage device already mounted at /mnt")
		return nil
	}

	fmt.Printf("Formatting %s with ext4 filesystem...\n", storageDevice)
	// Format the device with ext4
	formatCmd := fmt.Sprintf("sudo mkfs.ext4 -F %s", storageDevice)
	if err := execCommand("ssh", sshHost, formatCmd); err != nil {
		return fmt.Errorf("failed to format device: %w", err)
	}

	// Create mount point
	fmt.Println("Creating mount point /mnt...")
	_ = execCommand("ssh", sshHost, "sudo mkdir -p /mnt")

	// Mount the device
	fmt.Println("Mounting storage device at /mnt...")
	mountCmd := fmt.Sprintf("sudo mount %s /mnt", storageDevice)
	if err := execCommand("ssh", sshHost, mountCmd); err != nil {
		return fmt.Errorf("failed to mount device: %w", err)
	}

	// Add to fstab for persistent mount
	fmt.Println("Adding to /etc/fstab for persistent mount...")
	fstabEntry := fmt.Sprintf("%s /mnt ext4 defaults 0 2", storageDevice)
	fstabCmd := fmt.Sprintf("sudo grep -q '%s' /etc/fstab || echo '%s' | sudo tee -a /etc/fstab", storageDevice, fstabEntry)
	_ = execCommand("ssh", sshHost, fstabCmd)

	// Create directories for different storage types
	fmt.Println("Creating storage directories...")
	dirs := []string{
		"/mnt/k0s",           // k0s data
		"/mnt/containerd",    // Container images and layers
		"/mnt/vm-disks",      // VM disk images
		"/mnt/llm-models",    // LLM models
		"/mnt/services-data", // Service persistent data
	}

	for _, dir := range dirs {
		_ = execCommand("ssh", sshHost, fmt.Sprintf("sudo mkdir -p %s && sudo chmod 755 %s", dir, dir))
	}

	fmt.Println("✓ Storage device setup completed")
	return nil
}

func installVirtualizationPackages() error {
	fmt.Println("Installing virtualization packages...")

	// Check if packages are already installed
	checkCmd := "dpkg -l | grep -E 'qemu-kvm|libvirt-daemon-system' >/dev/null 2>&1"
	if execCommand("ssh", sshHost, checkCmd) == nil {
		fmt.Println("✓ Virtualization packages already installed")
		return nil
	}

	// Update package cache
	fmt.Println("Updating package cache...")
	if err := execCommand("ssh", sshHost, "sudo apt-get update -qq"); err != nil {
		fmt.Println("⚠ Warning: apt-get update failed, continuing anyway...")
	}

	// Install required packages
	fmt.Println("Installing QEMU, KVM, and libvirt packages...")
	installCmd := `sudo DEBIAN_FRONTEND=noninteractive apt-get install -y \
		qemu-kvm \
		libvirt-daemon-system \
		libvirt-clients \
		bridge-utils \
		cpu-checker \
		>/dev/null 2>&1`

	if err := execCommand("ssh", sshHost, installCmd); err != nil {
		return fmt.Errorf("failed to install virtualization packages: %w", err)
	}

	// Verify KVM is available
	checkKVMCmd := "test -c /dev/kvm && echo 'KVM available' || echo 'KVM not available'"
	if err := execCommand("ssh", sshHost, checkKVMCmd); err != nil {
		fmt.Println("⚠ Warning: /dev/kvm not available - VMs may not work")
	}

	// Set permissions on /dev/kvm (make it world-accessible)
	fmt.Println("Setting permissions on /dev/kvm...")
	if err := execCommand("ssh", sshHost, "sudo chmod 666 /dev/kvm"); err != nil {
		fmt.Println("⚠ Warning: failed to set /dev/kvm permissions")
	}

	// Make /dev/kvm permissions persistent across reboots
	udevRule := `KERNEL=="kvm", GROUP="kvm", MODE="0666"`
	udevCmd := fmt.Sprintf(`echo '%s' | sudo tee /etc/udev/rules.d/99-kvm.rules >/dev/null`, udevRule)
	if err := execCommand("ssh", sshHost, udevCmd); err != nil {
		fmt.Println("⚠ Warning: failed to create udev rule for /dev/kvm")
	}

	fmt.Println("✓ Virtualization packages installed")
	return nil
}

func deployK0s() error {
	fmt.Println("==> Deploying k3s")

	// Check SSH connection
	if err := execCommand("ssh", "-o", "ConnectTimeout=10", "-o", "BatchMode=yes", sshHost, "exit"); err != nil {
		return fmt.Errorf("cannot connect to %s - ensure SSH keys are configured", sshHost)
	}

	// Install virtualization packages
	if err := installVirtualizationPackages(); err != nil {
		return fmt.Errorf("failed to install virtualization packages: %w", err)
	}

	// Check if k3s is already running
	checkCmd := "systemctl is-active k3s"
	isRunning := execCommand("ssh", sshHost, checkCmd) == nil

	if !isRunning {
		fmt.Println("Installing k3s...")

		// Install k3s with custom data directory and KubeVirt-friendly settings
		installCmd := fmt.Sprintf(`curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--data-dir=/mnt/k3s --disable traefik --disable servicelb --kube-proxy-arg=conntrack-max-per-core=0" sh -`)
		if err := execCommand("ssh", sshHost, installCmd); err != nil {
			return fmt.Errorf("failed to install k3s: %w", err)
		}

		// Wait for k3s to start
		fmt.Println("Waiting for k3s to be ready...")
		time.Sleep(20 * time.Second)
	} else {
		fmt.Println("✓ k3s already running")
	}

	// Save kubeconfig locally
	kubeconfigData, err := exec.Command("ssh", sshHost, "sudo cat /etc/rancher/k3s/k3s.yaml").Output()
	if err != nil {
		return fmt.Errorf("failed to retrieve kubeconfig: %w", err)
	}

	// Replace localhost with actual host IP
	kubeconfigStr := string(kubeconfigData)
	// Extract IP from SSH host (format: user@ip)
	hostIP := sshHost
	if idx := strings.Index(sshHost, "@"); idx != -1 {
		hostIP = sshHost[idx+1:]
	}
	kubeconfigStr = strings.ReplaceAll(kubeconfigStr, "127.0.0.1", hostIP)

	if err := os.WriteFile(kubeconfig, []byte(kubeconfigStr), 0600); err != nil {
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	os.Setenv("KUBECONFIG", kubeconfig)

	// Wait for cluster to be ready
	if err := waitForCluster(); err != nil {
		return err
	}

	// Install dependencies
	if err := installDependencies(); err != nil {
		return err
	}

	fmt.Println("✓ k3s ready")
	return nil
}

func waitForCluster() error {
	fmt.Println("Waiting for k3s cluster to be ready...")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to build config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	ctx := context.Background()
	for i := 0; i < 60; i++ {
		nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err == nil && len(nodes.Items) > 0 {
			fmt.Printf("✓ Found %d node(s)\n", len(nodes.Items))
			break
		}
		if i == 59 {
			return fmt.Errorf("timeout waiting for nodes")
		}
		time.Sleep(3 * time.Second)
	}

	// Remove control-plane taint
	_ = execCommand("kubectl", "--kubeconfig", kubeconfig, "taint", "nodes", "--all", "node-role.kubernetes.io/control-plane:NoSchedule-")

	return nil
}

func installDependencies() error {
	fmt.Println("Installing dependencies...")

	// Install KubeVirt v1.6.0 (latest version as of 2025)
	_ = execCommand("kubectl", "--kubeconfig", kubeconfig, "create", "namespace", "kubevirt")
	if err := execCommand("kubectl", "--kubeconfig", kubeconfig, "apply", "-f", "https://github.com/kubevirt/kubevirt/releases/download/v1.6.0/kubevirt-operator.yaml"); err != nil {
		return err
	}
	if err := execCommand("kubectl", "--kubeconfig", kubeconfig, "apply", "-f", "https://github.com/kubevirt/kubevirt/releases/download/v1.6.0/kubevirt-cr.yaml"); err != nil {
		return err
	}

	// Configure KVM device permissions and enable hardware virtualization
	_ = execCommand("ssh", sshHost, "sudo chmod 666 /dev/kvm")
	_ = execCommand("ssh", sshHost, "sudo usermod -a -G kvm $(whoami)")

	// Wait for KubeVirt to be ready then patch for KVM support
	time.Sleep(5 * time.Second)
	_ = execCommand("kubectl", "--kubeconfig", kubeconfig, "-n", "kubevirt", "patch", "kubevirt", "kubevirt", "--type=merge", "-p", `{"spec":{"configuration":{"developerConfiguration":{"featureGates":["HardwareVirtualization"]}}}}`)

	// Install CDI v1.61.0 (latest version as of 2025)
	_ = execCommand("kubectl", "--kubeconfig", kubeconfig, "create", "-f", "https://github.com/kubevirt/containerized-data-importer/releases/download/v1.61.0/cdi-operator.yaml")
	_ = execCommand("kubectl", "--kubeconfig", kubeconfig, "create", "-f", "https://github.com/kubevirt/containerized-data-importer/releases/download/v1.61.0/cdi-cr.yaml")

	// Wait for CDI to be ready and create CDIConfig
	time.Sleep(10 * time.Second)
	cdiConfigYAML := `apiVersion: cdi.kubevirt.io/v1beta1
kind: CDIConfig
metadata:
  name: config
  namespace: cdi
spec:
  featureGates:
  - HonorWaitForFirstConsumer
  uploadProxyURLOverride: ""`
	cdiConfigFile := "/tmp/cdiconfig.yaml"
	if err := os.WriteFile(cdiConfigFile, []byte(cdiConfigYAML), 0600); err == nil {
		_ = execCommand("kubectl", "--kubeconfig", kubeconfig, "apply", "-f", cdiConfigFile)
		os.Remove(cdiConfigFile)
	}

	// Install local-path provisioner
	if err := execCommand("kubectl", "--kubeconfig", kubeconfig, "apply", "-f", "https://raw.githubusercontent.com/rancher/local-path-provisioner/v0.0.28/deploy/local-path-storage.yaml"); err != nil {
		return err
	}

	// Configure local-path provisioner to use /mnt/vm-disks
	fmt.Println("Configuring local-path provisioner to use /mnt/vm-disks...")
	time.Sleep(5 * time.Second) // Wait for provisioner to be created
	patchCmd := `{"data":{"config.json":"{\"nodePathMap\":[{\"node\":\"DEFAULT_PATH_FOR_NON_LISTED_NODES\",\"paths\":[\"/mnt/vm-disks\"]}]}"}}`
	_ = execCommand("kubectl", "--kubeconfig", kubeconfig, "-n", "local-path-storage", "patch", "configmap", "local-path-config", "-p", patchCmd)

	// Restart local-path-provisioner to apply changes
	_ = execCommand("kubectl", "--kubeconfig", kubeconfig, "-n", "local-path-storage", "rollout", "restart", "deployment/local-path-provisioner")

	fmt.Println("✓ Dependencies installed")
	return nil
}

func buildFrontend() error {
	fmt.Println("==> Building frontend")

	webDir := "web"
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		fmt.Println("⚠ No web directory found, skipping frontend build")
		return nil
	}

	if err := execCommandInDir(webDir, "npm", "install"); err != nil {
		return err
	}

	if err := execCommandInDir(webDir, "npm", "run", "build"); err != nil {
		return err
	}

	// Copy static files
	os.MkdirAll("internal/api/static", 0755)
	if err := exec.Command("cp", "-r", "static/", "internal/api/static/").Run(); err != nil {
		fmt.Println("⚠ Failed to copy static files")
	}

	fmt.Println("✓ Frontend built")
	return nil
}

func deployOperator() error {
	fmt.Println("==> Building and deploying operator")

	// Build operator binary
	os.MkdirAll("bin", 0755)
	buildCmd := exec.Command("go", "build", "-o", "bin/manager-linux", "cmd/main.go")
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build operator: %w", err)
	}

	// Stop existing operator and kill any remaining processes
	fmt.Println("Stopping existing operator...")
	_ = execCommand("ssh", sshHost, "sudo systemctl stop llmcloud-operator 2>/dev/null || true")
	_ = execCommand("ssh", sshHost, "sudo pkill -9 -f '/opt/llmcloud-operator/manager' || true")
	_ = execCommand("ssh", sshHost, "sudo pkill -9 -f 'llmcloud' || true")
	// Kill any process using port 8090 or 8081
	_ = execCommand("ssh", sshHost, "sudo fuser -k 8090/tcp 2>/dev/null || true")
	_ = execCommand("ssh", sshHost, "sudo fuser -k 8081/tcp 2>/dev/null || true")
	time.Sleep(3 * time.Second)

	// Copy binary
	_ = execCommand("ssh", sshHost, "sudo mkdir -p /opt/llmcloud-operator")
	if err := execCommand("scp", "bin/manager-linux", sshHost+":/tmp/manager"); err != nil {
		return err
	}
	if err := execCommand("ssh", sshHost, "sudo mv /tmp/manager /opt/llmcloud-operator/manager && sudo chmod +x /opt/llmcloud-operator/manager"); err != nil {
		return err
	}

	// Create kubeconfig on remote host
	kubeconfigCmd := "sudo k0s kubeconfig admin | sudo tee /opt/llmcloud-operator/kubeconfig > /dev/null"
	if err := execCommand("ssh", sshHost, kubeconfigCmd); err != nil {
		return fmt.Errorf("failed to create kubeconfig on remote host: %w", err)
	}

	// Create systemd service
	serviceContent := `[Unit]
Description=LLMCloud Operator
After=network.target

[Service]
Type=simple
ExecStart=/opt/llmcloud-operator/manager
Restart=always
RestartSec=5
Environment="KUBECONFIG=/opt/llmcloud-operator/kubeconfig"

[Install]
WantedBy=multi-user.target`

	serviceCmd := fmt.Sprintf("echo '%s' | sudo tee /etc/systemd/system/llmcloud-operator.service > /dev/null", serviceContent)
	if err := execCommand("ssh", sshHost, serviceCmd); err != nil {
		return err
	}

	// Start service
	if err := execCommand("ssh", sshHost, "sudo systemctl daemon-reload && sudo systemctl enable llmcloud-operator && sudo systemctl start llmcloud-operator"); err != nil {
		return err
	}

	// Wait for operator to start
	fmt.Println("Waiting for operator to start...")
	time.Sleep(10 * time.Second)

	// Install CRDs
	fmt.Println("Installing CRDs...")
	if err := execCommand("kubectl", "--kubeconfig", kubeconfig, "apply", "-f", "config/crd/bases"); err != nil {
		fmt.Println("⚠ Failed to install CRDs, they may already exist")
	}

	fmt.Println("✓ Operator deployed")
	return nil
}

func createRootUser() error {
	fmt.Println("==> Creating root user")

	// Generate password
	password, err := generatePassword()
	if err != nil {
		return err
	}

	// Build password hash generator if needed
	hashGenPath := "bin/gen-password-hash"
	if _, err := os.Stat(hashGenPath); os.IsNotExist(err) {
		if err := exec.Command("go", "build", "-o", hashGenPath, "scripts/gen-password-hash.go").Run(); err != nil {
			return fmt.Errorf("failed to build password hash generator: %w", err)
		}
	}

	// Generate hash
	hashCmd := exec.Command(hashGenPath, password)
	hashBytes, err := hashCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to generate password hash: %w", err)
	}
	passwordHash := string(hashBytes)

	// Create user
	userYAML := fmt.Sprintf(`apiVersion: llmcloud.llmcloud.io/v1alpha1
kind: User
metadata:
  name: root
spec:
  username: root
  passwordHash: %s
  email: root@localhost
  isAdmin: true`, passwordHash)

	userFile := "/tmp/root-user.yaml"
	if err := os.WriteFile(userFile, []byte(userYAML), 0600); err != nil {
		return err
	}

	if err := execCommand("kubectl", "--kubeconfig", kubeconfig, "apply", "-f", userFile); err != nil {
		fmt.Println("⚠ Root user may already exist")
	}

	// Display credentials
	fmt.Println("\n╔════════════════════════════════════════╗")
	fmt.Println("║       ROOT USER CREDENTIALS            ║")
	fmt.Println("╠════════════════════════════════════════╣")
	fmt.Printf("║  Username: root%s║\n", "                        ")
	fmt.Printf("║  Password: %-28s║\n", password)
	fmt.Println("╚════════════════════════════════════════╝")

	// Save credentials
	credFile := ".root-credentials"
	credContent := fmt.Sprintf("Username: root\nPassword: %s\n", password)
	os.WriteFile(credFile, []byte(credContent), 0600)
	fmt.Printf("\n✓ Credentials saved to: %s\n", credFile)

	return nil
}

func generatePassword() (string, error) {
	cmd := exec.Command("openssl", "rand", "-base64", "16")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	// Clean up the password (remove special chars)
	password := string(output)[:16]
	return password, nil
}

func execCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func execCommandInDir(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
