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

package main

import (
	"crypto/tls"
	"flag"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	llmcloudv1alpha1 "github.com/rusik69/llmcloud-operator/api/v1alpha1"
	"github.com/rusik69/llmcloud-operator/cmd/deploy"
	"github.com/rusik69/llmcloud-operator/cmd/uninstall"
	"github.com/rusik69/llmcloud-operator/internal/api"
	"github.com/rusik69/llmcloud-operator/internal/auth"
	"github.com/rusik69/llmcloud-operator/internal/controller"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(llmcloudv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	// Check for subcommands
	if len(os.Args) > 1 {
		if os.Args[1] == "deploy" || os.Args[1] == "uninstall" {
			rootCmd := &cobra.Command{Use: "manager"}
			rootCmd.AddCommand(deploy.NewDeployCmd())
			rootCmd.AddCommand(uninstall.NewUninstallCmd())
			if err := rootCmd.Execute(); err != nil {
				os.Exit(1)
			}
			return
		}
	}

	var metricsAddr, probeAddr string
	var metricsCertPath, metricsCertName, metricsCertKey string
	var webhookCertPath, webhookCertName, webhookCertKey string
	var enableLeaderElection, secureMetrics, enableHTTP2 bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "Metrics endpoint address")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "Health probe address")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election")
	flag.BoolVar(&secureMetrics, "metrics-secure", true, "Serve metrics via HTTPS")
	flag.StringVar(&webhookCertPath, "webhook-cert-path", "", "Webhook certificate directory")
	flag.StringVar(&webhookCertName, "webhook-cert-name", "tls.crt", "Webhook cert filename")
	flag.StringVar(&webhookCertKey, "webhook-cert-key", "tls.key", "Webhook key filename")
	flag.StringVar(&metricsCertPath, "metrics-cert-path", "", "Metrics certificate directory")
	flag.StringVar(&metricsCertName, "metrics-cert-name", "tls.crt", "Metrics cert filename")
	flag.StringVar(&metricsCertKey, "metrics-cert-key", "tls.key", "Metrics key filename")
	flag.BoolVar(&enableHTTP2, "enable-http2", false, "Enable HTTP/2")

	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	var tlsOpts []func(*tls.Config)
	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, func(c *tls.Config) {
			c.NextProtos = []string{"http/1.1"}
		})
	}

	webhookOpts := webhook.Options{TLSOpts: tlsOpts}
	if webhookCertPath != "" {
		webhookOpts.CertDir = webhookCertPath
		webhookOpts.CertName = webhookCertName
		webhookOpts.KeyName = webhookCertKey
	}

	metricsOpts := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}
	if secureMetrics {
		metricsOpts.FilterProvider = filters.WithAuthenticationAndAuthorization
	}
	if metricsCertPath != "" {
		metricsOpts.CertDir = metricsCertPath
		metricsOpts.CertName = metricsCertName
		metricsOpts.KeyName = metricsCertKey
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsOpts,
		WebhookServer:          webhook.NewServer(webhookOpts),
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "fe560ec5.llmcloud.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	controllers := []interface {
		SetupWithManager(ctrl.Manager) error
	}{
		&controller.ProjectReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()},
		&controller.VirtualMachineReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()},
		&controller.LLMModelReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()},
		&controller.ServiceReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()},
		&controller.UserReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()},
		// +kubebuilder:scaffold:builder
	}

	for _, c := range controllers {
		if err := c.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller")
			os.Exit(1)
		}
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Initialize JWT secret for authentication
	if err := auth.InitJWTSecret(); err != nil {
		setupLog.Error(err, "unable to initialize JWT secret")
		os.Exit(1)
	}

	go func() {
		if err := api.NewServer(mgr.GetClient()).Start(":8090"); err != nil {
			setupLog.Error(err, "API server failed")
		}
	}()

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
