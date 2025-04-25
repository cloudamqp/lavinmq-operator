package e2e

import (
	"context"
	"fmt"
	"lavinmq-operator/test/utils"
	"log"
	"os"
	"os/exec"
	"testing"

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
			log.Println("Creating kind cluster...")
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
			log.Println("Building and installing the operator...")
			err := utils.BuildingAndInstallingOperator(projectimage, kindClusterName)
			if err != nil {
				return ctx, fmt.Errorf("failed to build and install operator: %w", err)
			}
			return ctx, nil
		},
	)

	// Cleanup
	testEnv.Finish(
		func(ctx context.Context, c *envconf.Config) (context.Context, error) {
			log.Println("Undeploying LavinMQ controller...")
			cmd := exec.Command("make", "undeploy", "ignore-not-found=true")
			if _, err := utils.Run(cmd); err != nil {
				log.Printf("Warning: Failed to undeploy controller: %s\n", err)
			}

			log.Println("Uninstalling crd...")
			cmd = exec.Command("make", "uninstall", "ignore-not-found=true")
			if _, err := utils.Run(cmd); err != nil {
				log.Printf("Warning: Failed to install crd: %s\n", err)
			}
			return ctx, nil
		},
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			log.Println("Uninstalling etcd operator...")
			utils.UninstallEtcdOperator()
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
