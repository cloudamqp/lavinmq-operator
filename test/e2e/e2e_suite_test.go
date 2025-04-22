package e2e

import (
	"context"
	"fmt"
	"lavinmq-operator/test/utils"
	"os"
	"os/exec"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/support/kind"
)

var (
	testEnv         env.Environment
	namespace       string
	k8sClient       client.Client
	kindClusterName string
	projectimage    = "example.com/operator-sdk:v0.0.1"
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
			fmt.Println("Creating kind cluster...")
			return envfuncs.CreateClusterWithOpts(
				kindCluster,
				kindClusterName,
				clusterVersion,
			)(ctx, cfg)
		},

		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			fmt.Println("Creating test namespace...")
			_, err := envfuncs.CreateNamespace(namespace)(ctx, cfg)
			if err != nil {
				return ctx, fmt.Errorf("failed to create namespace: %w", err)
			}
			return ctx, nil
		},

		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			fmt.Println("Installing etcd operator...")
			utils.InstallEtcdOperator()
			return ctx, nil
		},
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			fmt.Println("Starting etcd cluster...")
			utils.SetupEtcdCluster(namespace)
			return ctx, nil
		},
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			fmt.Println("Building and installing the operator...")
			utils.BuildingAndInstallingOperator(projectimage, kindClusterName)
			return ctx, nil
		},
	)

	// Cleanup
	testEnv.Finish(
		func(ctx context.Context, c *envconf.Config) (context.Context, error) {
			fmt.Println("Undeploying LavinMQ controller...")
			cmd := exec.Command("make", "undeploy", "ignore-not-found=true")
			if _, err := utils.Run(cmd); err != nil {
				fmt.Printf("Warning: Failed to undeploy controller: %s\n", err)
			}

			fmt.Println("Uninstalling crd...")
			cmd = exec.Command("make", "uninstall", "ignore-not-found=true")
			if _, err := utils.Run(cmd); err != nil {
				fmt.Printf("Warning: Failed to install crd: %s\n", err)
			}
			return ctx, nil
		},
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			fmt.Println("Uninstalling etcd operator...")
			utils.UninstallEtcdOperator()
			return ctx, nil
		},
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			fmt.Println("Removing test namespace...")
			ctx, err := envfuncs.DeleteNamespace(namespace)(ctx, cfg)
			if err != nil {
				fmt.Printf("Failed to delete namespace: %s\n", err)
			}
			return ctx, nil
		},
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			fmt.Println("Destroying kind cluster...")
			return envfuncs.DestroyCluster(kindClusterName)(ctx, cfg)
		},
	)

	os.Exit(testEnv.Run(m))
}
