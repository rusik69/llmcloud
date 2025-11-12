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

package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	llmcloudv1alpha1 "github.com/rusik69/llmcloud-operator/api/v1alpha1"
)

// VirtualMachineReconciler reconciles a VirtualMachine object
type VirtualMachineReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=llmcloud.llmcloud.io,resources=virtualmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=llmcloud.llmcloud.io,resources=virtualmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=llmcloud.llmcloud.io,resources=virtualmachines/finalizers,verbs=update
// +kubebuilder:rbac:groups=kubevirt.io,resources=virtualmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubevirt.io,resources=virtualmachineinstances,verbs=get;list;watch

const (
	vmFinalizer = "llmcloud.llmcloud.io/vm-finalizer"
)

func (r *VirtualMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	vm := &llmcloudv1alpha1.VirtualMachine{}
	if err := r.Get(ctx, req.NamespacedName, vm); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !vm.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(vm, vmFinalizer) {
			if err := r.finalizeVM(ctx, vm); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(vm, vmFinalizer)
			return ctrl.Result{}, r.Update(ctx, vm)
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(vm, vmFinalizer) {
		controllerutil.AddFinalizer(vm, vmFinalizer)
		return ctrl.Result{Requeue: true}, r.Update(ctx, vm)
	}

	// Handle reboot annotation
	if vm.Annotations != nil && vm.Annotations["llmcloud.io/reboot"] == "true" {
		if err := r.rebootVM(ctx, vm); err != nil {
			log.Error(err, "Failed to reboot VM")
			return ctrl.Result{}, err
		}
		// Remove the annotation after handling
		delete(vm.Annotations, "llmcloud.io/reboot")
		if err := r.Update(ctx, vm); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("VM reboot initiated", "vm", vm.Name)
	}

	if err := r.reconcileKubeVirtVM(ctx, vm); err != nil {
		log.Error(err, "Failed to reconcile KubeVirt VM")
		r.updateVMStatus(ctx, vm, "Error", err.Error())
		return ctrl.Result{}, err
	}

	if err := r.updateVMStatusFromVMI(ctx, vm); err != nil {
		// Ignore conflict errors - they will be retried on next reconcile
		if !errors.IsConflict(err) {
			log.Error(err, "Failed to update VM status from VMI")
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

func (r *VirtualMachineReconciler) reconcileKubeVirtVM(ctx context.Context, vm *llmcloudv1alpha1.VirtualMachine) error {
	kvVM := r.buildKubeVirtVM(vm)

	// Use Server-Side Apply for idempotent create/update
	// This will create if not exists, or update if exists
	return r.Patch(ctx, kvVM, client.Apply, client.ForceOwnership, client.FieldOwner("llmcloud-operator"))
}

func (r *VirtualMachineReconciler) buildKubeVirtVM(vm *llmcloudv1alpha1.VirtualMachine) *unstructured.Unstructured {
	runStrategy := vm.Spec.RunStrategy
	if runStrategy == "" {
		runStrategy = "Always"
	}

	cloudInitUserData := vm.Spec.CloudInit
	if cloudInitUserData == "" && len(vm.Spec.SSHKeys) > 0 {
		cloudInitUserData = fmt.Sprintf("#cloud-config\nssh_authorized_keys:\n%s",
			strings.Join(vm.Spec.SSHKeys, "\n"))
	}

	// Build disks and volumes based on configuration
	disks := []interface{}{
		map[string]interface{}{
			"name": "containerdisk",
			"disk": map[string]interface{}{
				"bus": "virtio",
			},
		},
		map[string]interface{}{
			"name": "datadisk",
			"disk": map[string]interface{}{
				"bus": "virtio",
			},
		},
	}
	volumes := []interface{}{
		map[string]interface{}{
			"name": "containerdisk",
			"containerDisk": map[string]interface{}{
				"image": llmcloudv1alpha1.GetImageForOS(vm.Spec.OS, vm.Spec.OSVersion),
			},
		},
		map[string]interface{}{
			"name": "datadisk",
			"dataVolume": map[string]interface{}{
				"name": vm.Name + "-disk",
			},
		},
	}

	// Only add cloudInit if we have data
	if cloudInitUserData != "" {
		disks = append(disks, map[string]interface{}{
			"name": "cloudinitdisk",
			"disk": map[string]interface{}{
				"bus": "virtio",
			},
		})
		volumes = append(volumes, map[string]interface{}{
			"name": "cloudinitdisk",
			"cloudInitNoCloud": map[string]interface{}{
				"userData": cloudInitUserData,
			},
		})
	}

	// Get disk size with default
	diskSize := vm.Spec.DiskSize
	if diskSize == "" {
		diskSize = "10Gi"
	}

	// Get storage class (local storage)
	storageClass := vm.Spec.StorageClass
	if storageClass == "" {
		storageClass = "local-path"
	}

	kvVM := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachine",
			"metadata": map[string]interface{}{
				"name":      vm.Name,
				"namespace": vm.Namespace,
				"labels": map[string]interface{}{
					"llmcloud.io/managed": "true",
				},
			},
			"spec": map[string]interface{}{
				"runStrategy": runStrategy,
				"dataVolumeTemplates": []interface{}{
					map[string]interface{}{
						"metadata": map[string]interface{}{
							"name": vm.Name + "-disk",
						},
						"spec": map[string]interface{}{
							"source": map[string]interface{}{
								"blank": map[string]interface{}{},
							},
							"storage": map[string]interface{}{
								"accessModes": []interface{}{"ReadWriteOnce"},
								"resources": map[string]interface{}{
									"requests": map[string]interface{}{
										"storage": diskSize,
									},
								},
								"storageClassName": storageClass,
							},
						},
					},
				},
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"domain": map[string]interface{}{
							"cpu": map[string]interface{}{
								"cores": vm.Spec.CPUs,
							},
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{
									"memory": vm.Spec.Memory,
								},
							},
							"devices": map[string]interface{}{
								"disks": disks,
							},
						},
						"volumes": volumes,
					},
				},
			},
		},
	}

	return kvVM
}

func (r *VirtualMachineReconciler) updateVMStatusFromVMI(ctx context.Context, vm *llmcloudv1alpha1.VirtualMachine) error {
	vmi := &unstructured.Unstructured{}
	vmi.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kubevirt.io",
		Version: "v1",
		Kind:    "VirtualMachineInstance",
	})

	err := r.Get(ctx, client.ObjectKey{Name: vm.Name, Namespace: vm.Namespace}, vmi)
	if err != nil {
		if errors.IsNotFound(err) {
			vm.Status.Phase = llmcloudv1alpha1.PhasePending
			vm.Status.Ready = false
			return r.Status().Update(ctx, vm)
		}
		return err
	}

	// Extract status from VMI
	status, _, _ := unstructured.NestedMap(vmi.Object, "status")
	if phase, ok := status["phase"].(string); ok {
		vm.Status.Phase = phase
		vm.Status.Ready = (phase == "Running")
	}

	if node, ok := status["nodeName"].(string); ok {
		vm.Status.Node = node
	}

	if interfaces, ok := status["interfaces"].([]interface{}); ok && len(interfaces) > 0 {
		if iface, ok := interfaces[0].(map[string]interface{}); ok {
			if ip, ok := iface["ipAddress"].(string); ok {
				vm.Status.IPAddress = ip
			}
		}
	}

	meta.SetStatusCondition(&vm.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "VMRunning",
		Message:            "Virtual machine is running",
		ObservedGeneration: vm.Generation,
	})

	return r.Status().Update(ctx, vm)
}

func (r *VirtualMachineReconciler) finalizeVM(ctx context.Context, vm *llmcloudv1alpha1.VirtualMachine) error {
	kvVM := &unstructured.Unstructured{}
	kvVM.SetGroupVersionKind(schema.GroupVersionKind{Group: "kubevirt.io", Version: "v1", Kind: "VirtualMachine"})
	kvVM.SetName(vm.Name)
	kvVM.SetNamespace(vm.Namespace)
	return client.IgnoreNotFound(r.Delete(ctx, kvVM))
}

func (r *VirtualMachineReconciler) updateVMStatus(ctx context.Context, vm *llmcloudv1alpha1.VirtualMachine, phase, message string) {
	vm.Status.Phase = phase
	vm.Status.Ready = false
	meta.SetStatusCondition(&vm.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             "ReconciliationError",
		Message:            message,
		ObservedGeneration: vm.Generation,
	})
	_ = r.Status().Update(ctx, vm)
}

// rebootVM reboots a KubeVirt VM by stopping and starting it
func (r *VirtualMachineReconciler) rebootVM(ctx context.Context, vm *llmcloudv1alpha1.VirtualMachine) error {
	log := logf.FromContext(ctx)

	// Get the KubeVirt VM
	kvVM := &unstructured.Unstructured{}
	kvVM.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "kubevirt.io",
		Version: "v1",
		Kind:    "VirtualMachine",
	})

	if err := r.Get(ctx, client.ObjectKey{Namespace: vm.Namespace, Name: vm.Name}, kvVM); err != nil {
		return fmt.Errorf("failed to get KubeVirt VM: %w", err)
	}

	// Set runStrategy to Halted
	if err := unstructured.SetNestedField(kvVM.Object, "Halted", "spec", "runStrategy"); err != nil {
		return fmt.Errorf("failed to set runStrategy to Halted: %w", err)
	}

	if err := r.Update(ctx, kvVM); err != nil {
		return fmt.Errorf("failed to stop VM: %w", err)
	}

	log.Info("VM stopped for reboot", "vm", vm.Name)

	// Set runStrategy back to Always
	if err := unstructured.SetNestedField(kvVM.Object, "Always", "spec", "runStrategy"); err != nil {
		return fmt.Errorf("failed to set runStrategy to Always: %w", err)
	}

	if err := r.Update(ctx, kvVM); err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}

	log.Info("VM restarted", "vm", vm.Name)
	return nil
}

func (r *VirtualMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&llmcloudv1alpha1.VirtualMachine{}).Named("virtualmachine").Complete(r)
}
