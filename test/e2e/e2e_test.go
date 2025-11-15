//go:build e2e
// +build e2e

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

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rusik69/llmcloud-operator/test/utils"
)

// namespace where the project is deployed in
const namespace = "llmcloud-operator-system"

// serviceAccountName created for the project
const serviceAccountName = "llmcloud-operator-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "llmcloud-operator-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "llmcloud-operator-metrics-binding"

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
	BeforeAll(func() {
		useExistingCluster := os.Getenv("USE_EXISTING_CLUSTER") == "true"

		if !useExistingCluster {
			By("creating manager namespace")
			cmd := exec.Command("kubectl", "create", "ns", namespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

			By("labeling the namespace to enforce the restricted security policy")
			cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
				"pod-security.kubernetes.io/enforce=restricted")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

			By("installing CRDs")
			cmd = exec.Command("make", "install")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

			By("deploying the controller-manager")
			cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")
		} else {
			By("Using existing cluster deployment, skipping setup")
			_, _ = fmt.Fprintf(GinkgoWriter, "Namespace: %s\n", namespace)
			_, _ = fmt.Fprintf(GinkgoWriter, "Image: %s\n", projectImage)
		}
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		useExistingCluster := os.Getenv("USE_EXISTING_CLUSTER") == "true"

		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace, "--ignore-not-found=true")
		_, _ = utils.Run(cmd)

		By("cleaning up metrics ClusterRoleBinding")
		cmd = exec.Command("kubectl", "delete", "clusterrolebinding", metricsRoleBindingName, "--ignore-not-found=true")
		_, _ = utils.Run(cmd)

		if !useExistingCluster {
			By("undeploying the controller-manager")
			cmd = exec.Command("make", "undeploy")
			_, _ = utils.Run(cmd)

			By("uninstalling CRDs")
			cmd = exec.Command("make", "uninstall")
			_, _ = utils.Run(cmd)

			By("removing manager namespace")
			cmd = exec.Command("kubectl", "delete", "ns", namespace)
			_, _ = utils.Run(cmd)
		} else {
			By("Using existing cluster, cleanup will be handled by CI")
		}
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod status")
			cmd = exec.Command("kubectl", "get", "pod", controllerPodName, "-n", namespace, "-o", "jsonpath={.status.phase}")
			podStatus, err := utils.Run(cmd)
			if err == nil {
				fmt.Printf("Controller pod status: %s\n", podStatus)
			} else {
				fmt.Println("Failed to get pod status")
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				// Get the name of the controller-manager pod
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

				// Validate the pod's status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})

		It("should ensure the metrics endpoint is serving metrics", func() {
			By("creating a ClusterRoleBinding for the service account to allow access to metrics")
			// Delete if exists, then create
			cmd := exec.Command("kubectl", "delete", "clusterrolebinding", metricsRoleBindingName, "--ignore-not-found=true")
			_, _ = utils.Run(cmd)

			cmd = exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
				"--clusterrole=llmcloud-operator-metrics-reader",
				fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
			)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

			By("validating that the metrics service is available")
			cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

			By("getting the service account token")
			token, err := serviceAccountToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			By("waiting for the metrics endpoint to be ready")
			verifyMetricsEndpointReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "endpoints", metricsServiceName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("8443"), "Metrics endpoint is not ready")
			}
			Eventually(verifyMetricsEndpointReady).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("controller-runtime.metrics\tServing metrics server"),
					"Metrics server not yet started")
			}
			Eventually(verifyMetricsServerStarted).Should(Succeed())

			By("creating the curl-metrics pod to access the metrics endpoint")
			cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
				"--namespace", namespace,
				"--image=curlimages/curl:latest",
				"--overrides",
				fmt.Sprintf(`{
					"spec": {
						"containers": [{
							"name": "curl",
							"image": "curlimages/curl:latest",
							"command": ["/bin/sh", "-c"],
							"args": ["curl -s -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics | grep -E '^(controller_runtime_reconcile_total|controller_runtime_reconcile_errors_total)'"],
							"securityContext": {
								"readOnlyRootFilesystem": true,
								"allowPrivilegeEscalation": false,
								"capabilities": {
									"drop": ["ALL"]
								},
								"runAsNonRoot": true,
								"runAsUser": 1000,
								"seccompProfile": {
									"type": "RuntimeDefault"
								}
							}
						}],
						"serviceAccountName": "%s"
					}
				}`, token, metricsServiceName, namespace, serviceAccountName))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

			By("waiting for the curl-metrics pod to complete.")
			verifyCurlUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
					"-o", "jsonpath={.status.phase}",
					"-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
			}
			Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			verifyMetricsAvailable := func(g Gomega) {
				metricsOutput, err := getMetricsOutput()
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
				g.Expect(metricsOutput).NotTo(BeEmpty())
				g.Expect(metricsOutput).To(ContainSubstring("controller_runtime_reconcile_total"))
			}
			Eventually(verifyMetricsAvailable, 2*time.Minute).Should(Succeed())
		})

		// +kubebuilder:scaffold:e2e-webhooks-checks
	})

	Context("VirtualMachine Lifecycle", Ordered, func() {
		const testVMName = "test-cirros-vm"
		const testNamespace = "default"

		AfterAll(func() {
			By("cleaning up test VM")
			cmd := exec.Command("kubectl", "delete", "virtualmachine", testVMName, "-n", testNamespace, "--ignore-not-found=true")
			_, _ = utils.Run(cmd)
		})

		It("should create a VirtualMachine successfully", func() {
			By("creating a VirtualMachine CR")
			vmYAML := fmt.Sprintf(`
apiVersion: llmcloud.llmcloud.io/v1alpha1
kind: VirtualMachine
metadata:
  name: %s
  namespace: %s
spec:
  cpus: 1
  memory: "256Mi"
  os: cirros
  runStrategy: Always
`, testVMName, testNamespace)

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(vmYAML)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create VirtualMachine")

			By("verifying VirtualMachine CR was created")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "virtualmachine", testVMName, "-n", testNamespace, "-o", "jsonpath={.metadata.name}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal(testVMName))
			}, 30*time.Second).Should(Succeed())

			By("waiting for VirtualMachine to be Running")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "virtualmachine", testVMName, "-n", testNamespace, "-o", "jsonpath={.status.phase}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Or(Equal("Running"), Equal("Pending")), "VM should be Running or Pending")
			}, 5*time.Minute, 5*time.Second).Should(Succeed())

			By("verifying underlying KubeVirt VirtualMachine was created")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "virtualmachine", testVMName, "-n", testNamespace, "-o", "json")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("kubevirt.io"))
			}, 1*time.Minute).Should(Succeed())
		})

		It("should stop a running VirtualMachine", func() {
			By("updating VirtualMachine runStrategy to Halted")
			patchJSON := `{"spec":{"runStrategy":"Halted"}}`
			cmd := exec.Command("kubectl", "patch", "virtualmachine", testVMName, "-n", testNamespace,
				"--type=merge", "-p", patchJSON)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to patch VirtualMachine")

			By("verifying VirtualMachine runStrategy was updated")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "virtualmachine", testVMName, "-n", testNamespace,
					"-o", "jsonpath={.spec.runStrategy}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Halted"))
			}, 30*time.Second).Should(Succeed())

			By("verifying VirtualMachineInstance is stopped")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "virtualmachineinstance", testVMName, "-n", testNamespace)
				_, err := utils.Run(cmd)
				// Should not exist or be in stopped state
				g.Expect(err).To(HaveOccurred(), "VirtualMachineInstance should not be running")
			}, 2*time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should start a stopped VirtualMachine", func() {
			By("updating VirtualMachine runStrategy to Always")
			patchJSON := `{"spec":{"runStrategy":"Always"}}`
			cmd := exec.Command("kubectl", "patch", "virtualmachine", testVMName, "-n", testNamespace,
				"--type=merge", "-p", patchJSON)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to patch VirtualMachine")

			By("verifying VirtualMachine runStrategy was updated")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "virtualmachine", testVMName, "-n", testNamespace,
					"-o", "jsonpath={.spec.runStrategy}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Always"))
			}, 30*time.Second).Should(Succeed())

			By("verifying VirtualMachine is running again")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "virtualmachine", testVMName, "-n", testNamespace,
					"-o", "jsonpath={.status.phase}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Or(Equal("Running"), Equal("Pending")))
			}, 3*time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should delete a VirtualMachine successfully", func() {
			By("deleting the VirtualMachine CR")
			cmd := exec.Command("kubectl", "delete", "virtualmachine", testVMName, "-n", testNamespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to delete VirtualMachine")

			By("verifying VirtualMachine CR was deleted")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "virtualmachine", testVMName, "-n", testNamespace)
				_, err := utils.Run(cmd)
				g.Expect(err).To(HaveOccurred(), "VirtualMachine should be deleted")
			}, 1*time.Minute, 2*time.Second).Should(Succeed())

			By("verifying underlying KubeVirt resources were cleaned up")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "virtualmachine,virtualmachineinstance,datavolume",
					"-l", fmt.Sprintf("vm=%s", testVMName), "-n", testNamespace)
				output, err := utils.Run(cmd)
				g.Expect(err).To(Or(HaveOccurred(), Not(HaveOccurred())))
				if err == nil {
					g.Expect(output).To(ContainSubstring("No resources found"))
				}
			}, 2*time.Minute, 5*time.Second).Should(Succeed())
		})
	})
})

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	// Temporary file to store the token request
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		// Execute kubectl command to create the token
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		// Parse the JSON output to extract the token
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() (string, error) {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	return utils.Run(cmd)
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}
