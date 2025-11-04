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

package uninstall

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
	sshHost      string
	kubeconfig   string
	uninstallK0s bool
)

func NewUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall llmcloud-operator from remote k0s cluster",
		Long:  `Uninstalls llmcloud-operator and optionally k0s from a remote host via SSH`,
		RunE:  runUninstall,
	}

	cmd.Flags().StringVar(&sshHost, "ssh-host", os.Getenv("SSH_HOST"), "SSH host (user@hostname)")
	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", filepath.Join(os.Getenv("HOME"), ".kube", "config-llmcloud"), "Kubeconfig path")
	cmd.Flags().BoolVar(&uninstallK0s, "k0s", false, "Also uninstall k0s from the node")

	return cmd
}

func runUninstall(cmd *cobra.Command, args []string) error {
	if sshHost == "" {
		return fmt.Errorf("SSH_HOST not set - use --ssh-host or set SSH_HOST environment variable")
	}

	fmt.Printf("==> Uninstalling from %s\n", sshHost)

	// Stop operator service
	if err := stopOperator(); err != nil {
		fmt.Printf("⚠ Warning: failed to stop operator: %v\n", err)
	}

	// Remove finalizers and delete resources
	if err := cleanupResources(); err != nil {
		fmt.Printf("⚠ Warning: failed to cleanup resources: %v\n", err)
	}

	// Cleanup operator files
	if err := cleanupOperatorFiles(); err != nil {
		fmt.Printf("⚠ Warning: failed to cleanup operator files: %v\n", err)
	}

	// Uninstall k0s if requested
	if uninstallK0s {
		if err := uninstallK0sCluster(); err != nil {
			return fmt.Errorf("failed to uninstall k0s: %w", err)
		}
	}

	fmt.Println("\n✓ Uninstall completed successfully!")
	return nil
}

func stopOperator() error {
	fmt.Println("Stopping operator service...")

	_ = execCommand("ssh", sshHost, "sudo systemctl stop llmcloud-operator 2>/dev/null || true")
	_ = execCommand("ssh", sshHost, "sudo systemctl disable llmcloud-operator 2>/dev/null || true")
	_ = execCommand("ssh", sshHost, "sudo pkill -f llmcloud-operator || true")

	fmt.Println("✓ Operator stopped")
	return nil
}

func cleanupResources() error {
	fmt.Println("Cleaning up Kubernetes resources...")

	// Check if kubeconfig exists
	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		fmt.Println("⚠ Kubeconfig not found, skipping resource cleanup")
		return nil
	}

	// Load kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Printf("⚠ Failed to load kubeconfig: %v, skipping resource cleanup\n", err)
		return nil
	}

	// Set timeout for API server connection
	config.Timeout = 10 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("⚠ Failed to create kubernetes client: %v, skipping resource cleanup\n", err)
		return nil
	}

	// Use context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Remove finalizers from projects
	fmt.Println("Removing finalizers from projects...")
	if err := removeFinalizers(ctx, "projects"); err != nil {
		fmt.Printf("⚠ Warning: failed to remove project finalizers: %v\n", err)
	}

	// Remove finalizers from users
	fmt.Println("Removing finalizers from users...")
	if err := removeFinalizers(ctx, "users"); err != nil {
		fmt.Printf("⚠ Warning: failed to remove user finalizers: %v\n", err)
	}

	// Delete resources
	resources := []string{"llmmodels", "services.llmcloud.io", "virtualmachines", "projects", "users"}
	for _, resource := range resources {
		fmt.Printf("Deleting %s...\n", resource)
		deleteCmd := fmt.Sprintf("kubectl --kubeconfig=%s delete %s --all --all-namespaces --timeout=10s 2>/dev/null || true", kubeconfig, resource)
		_ = execCommandShell(deleteCmd)
	}

	// Wait for resources to be deleted
	time.Sleep(2 * time.Second)

	// Delete project namespaces with force
	fmt.Println("Deleting project namespaces...")
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, ns := range namespaces.Items {
			if strings.HasPrefix(ns.Name, "project-") {
				fmt.Printf("  Deleting namespace %s...\n", ns.Name)

				// Delete all resources without waiting - use short timeouts
				_ = execCommandShell(fmt.Sprintf("kubectl --kubeconfig=%s delete virtualmachines.llmcloud.llmcloud.io --all -n %s --timeout=3s 2>/dev/null || true", kubeconfig, ns.Name))
				_ = execCommandShell(fmt.Sprintf("kubectl --kubeconfig=%s delete datavolumes --all -n %s --timeout=3s 2>/dev/null || true", kubeconfig, ns.Name))
				_ = execCommandShell(fmt.Sprintf("kubectl --kubeconfig=%s delete pvc --all -n %s --timeout=3s 2>/dev/null || true", kubeconfig, ns.Name))
				_ = execCommandShell(fmt.Sprintf("kubectl --kubeconfig=%s delete pods --all -n %s --grace-period=0 --force --timeout=3s 2>/dev/null || true", kubeconfig, ns.Name))

				// Remove finalizers from namespace
				ns.Finalizers = []string{}
				_, _ = clientset.CoreV1().Namespaces().Update(ctx, &ns, metav1.UpdateOptions{})

				// Delete namespace without waiting
				_ = execCommandShell(fmt.Sprintf("kubectl --kubeconfig=%s delete namespace %s --wait=false --timeout=2s 2>/dev/null || true", kubeconfig, ns.Name))
			}
		}
	}

	// Force cleanup any stuck namespaces after a brief wait - don't wait for this either
	time.Sleep(2 * time.Second)
	_ = execCommandShell(fmt.Sprintf("for ns in $(kubectl --kubeconfig=%s get ns -o name 2>/dev/null | grep project- | cut -d/ -f2); do kubectl --kubeconfig=%s patch namespace $ns -p '{\"metadata\":{\"finalizers\":null}}' --type=merge 2>/dev/null; kubectl --kubeconfig=%s delete namespace $ns --wait=false --grace-period=0 2>/dev/null; done || true", kubeconfig, kubeconfig, kubeconfig))

	fmt.Println("✓ Resources cleaned up")
	return nil
}

func removeFinalizers(ctx context.Context, resource string) error {
	// Get all resources
	listCmd := fmt.Sprintf("kubectl --kubeconfig=%s get %s -o name 2>/dev/null", kubeconfig, resource)
	output, err := exec.Command("sh", "-c", listCmd).Output()
	if err != nil {
		return nil // No resources found
	}

	resources := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, res := range resources {
		if res == "" {
			continue
		}
		patchCmd := fmt.Sprintf("kubectl --kubeconfig=%s patch %s -p '{\"metadata\":{\"finalizers\":[]}}' --type=merge 2>/dev/null", kubeconfig, res)
		_ = execCommandShell(patchCmd)
	}

	return nil
}

func cleanupOperatorFiles() error {
	fmt.Println("Cleaning up operator files...")

	_ = execCommand("ssh", sshHost, "sudo rm -rf /opt/llmcloud-operator 2>/dev/null || true")
	_ = execCommand("ssh", sshHost, "sudo rm -f /etc/systemd/system/llmcloud-operator.service 2>/dev/null || true")
	_ = execCommand("ssh", sshHost, "sudo systemctl daemon-reload || true")

	fmt.Println("✓ Operator files cleaned up")
	return nil
}

func uninstallK0sCluster() error {
	fmt.Println("==> Uninstalling k3s cluster")

	// Check if k3s is installed
	checkCmd := "command -v k3s >/dev/null 2>&1"
	if execCommand("ssh", sshHost, checkCmd) != nil {
		fmt.Println("⚠ k3s not found on remote host, skipping")
		return nil
	}

	// Stop k3s service
	fmt.Println("Stopping k3s service...")
	_ = execCommand("ssh", sshHost, "sudo systemctl stop k3s 2>/dev/null || true")

	// Uninstall k3s using the official uninstall script
	fmt.Println("Uninstalling k3s...")
	_ = execCommand("ssh", sshHost, "sudo /usr/local/bin/k3s-uninstall.sh 2>/dev/null || true")

	// Clean up storage device data
	fmt.Println("Cleaning up storage directories...")
	_ = execCommand("ssh", sshHost, "sudo rm -rf /mnt/k3s /mnt/vm-disks /mnt/llm-models /mnt/services-data 2>/dev/null || true")

	// Unmount /mnt
	fmt.Println("Unmounting /mnt...")
	_ = execCommand("ssh", sshHost, "sudo umount /mnt 2>/dev/null || true")

	// Remove fstab entry
	fmt.Println("Removing fstab entry...")
	_ = execCommand("ssh", sshHost, "sudo sed -i '/\\/mnt.*ext4/d' /etc/fstab 2>/dev/null || true")

	// Remove kubeconfig
	if _, err := os.Stat(kubeconfig); err == nil {
		fmt.Printf("Removing kubeconfig %s...\n", kubeconfig)
		_ = os.Remove(kubeconfig)
	}

	fmt.Println("✓ k0s uninstalled successfully")
	return nil
}

func execCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func execCommandShell(command string) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
