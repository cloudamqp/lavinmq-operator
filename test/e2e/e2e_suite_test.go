package e2e

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"

	"github.com/cloudamqp/lavinmq-operator/test/utils"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/support/kind"
)

var (
	testEnv         env.Environment
	namespace       string
	kindClusterName string
	projectimage    = "cloudamqp/lavin-operator:v0.0.1"
	clusterVersion  = "kindest/node:v1.32.2"
)

func TestMain(m *testing.M) {
	cfg, _ := envconf.NewFromFlags()
	// Setup test environment
	testEnv = env.NewWithConfig(cfg)

	kindClusterName = envconf.RandomName("lavinmq", 15)
	namespace = envconf.RandomName("lavinmq-ns", 15)
	kindCluster := kind.NewCluster(kindClusterName)
	clusterVersion := kind.WithImage(clusterVersion)

	// Setup test environment
	testEnv.Setup(
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Creating kind cluster...", kindClusterName)
			return envfuncs.CreateClusterWithOpts(
				kindCluster,
				kindClusterName,
				clusterVersion,
			)(ctx, cfg)
		},

		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Creating test namespace...")
			_, err := envfuncs.CreateNamespace(namespace)(ctx, cfg)
			if err != nil {
				return ctx, fmt.Errorf("failed to create namespace: %w", err)
			}
			return ctx, nil
		},

		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Installing cert manager...")
			err := utils.InstallCertManager()
			if err != nil {
				return ctx, fmt.Errorf("failed to install cert manager: %w", err)
			}
			return ctx, nil
		},

		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Installing etcd operator...")
			err := utils.InstallEtcdOperator()
			if err != nil {
				return ctx, fmt.Errorf("failed to install etcd operator: %w", err)
			}
			return ctx, nil
		},
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Starting etcd cluster...")
			err := utils.SetupEtcdCluster(namespace)
			if err != nil {
				return ctx, fmt.Errorf("failed to setup etcd cluster: %w", err)
			}
			return ctx, nil
		},
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Building the operator image...")
			err := utils.BuildingOperatorImage(projectimage)
			if err != nil {
				return ctx, fmt.Errorf("failed to build operator image: %w", err)
			}
			return ctx, nil
		},
		envfuncs.LoadImageToCluster(kindClusterName, projectimage),
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Installing the operator...")
			err := utils.InstallingOperator()
			if err != nil {
				return ctx, fmt.Errorf("failed to build and install operator: %w", err)
			}
			return ctx, nil
		},
	)

	// Cleanup
	testEnv.Finish(
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Undeploying LavinMQ controller...")
			cmd := exec.Command("kubectl", "delete", "-f", "dist/install.yaml")
			if _, err := utils.Run(cmd); err != nil {
				log.Printf("Warning: Failed to teardown operator: %s\n", err)
			}
			return ctx, nil
		},
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Uninstalling etcd operator...")
			utils.UninstallEtcdOperator()
			return ctx, nil
		},
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Uninstalling cert manager...")
			utils.UninstallCertManager()
			return ctx, nil
		},
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Removing test namespace...")
			ctx, err := envfuncs.DeleteNamespace(namespace)(ctx, cfg)
			if err != nil {
				log.Printf("Failed to delete namespace: %s\n", err)
			}
			return ctx, nil
		},
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Destroying kind cluster...")
			return envfuncs.DestroyCluster(kindClusterName)(ctx, cfg)
		},
	)

	os.Exit(testEnv.Run(m))
}
