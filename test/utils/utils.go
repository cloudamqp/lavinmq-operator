/*
Copyright 2025.

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

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sigs.k8s.io/e2e-framework/support"
)

const (
	prometheusOperatorVersion = "v0.72.0"
	prometheusOperatorURL     = "https://github.com/prometheus-operator/prometheus-operator/" +
		"releases/download/%s/bundle.yaml"

	certmanagerVersion = "v1.17.2"
	certmanagerURLTmpl = "https://github.com/cert-manager/cert-manager/releases/download/%s/cert-manager.yaml"

	etcdOperatorVersion = "v0.1.0"
	etcdOperatorURL     = "https://github.com/etcd-io/etcd-operator/releases/download/%s/install-%s.yaml"
)

func warnError(err error) {
	_, _ = fmt.Printf("warning: %v\n", err)
}

// InstallPrometheusOperator installs the prometheus Operator to be used to export the enabled metrics.
func InstallPrometheusOperator() error {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "create", "-f", url)
	_, err := Run(cmd)
	return err
}

// Run executes the provided command within this context
func Run(cmd *exec.Cmd) ([]byte, error) {
	dir, _ := GetProjectDir()
	cmd.Dir = dir

	if err := os.Chdir(cmd.Dir); err != nil {
		_, _ = fmt.Printf("chdir dir: %s\n", err)
	}

	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Printf("running: %s\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s failed with error: (%v) %s", command, err, string(output))
	}

	return output, nil
}

// UninstallPrometheusOperator uninstalls the prometheus
func UninstallPrometheusOperator() {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// UninstallCertManager uninstalls the cert manager
func UninstallCertManager() {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// InstallCertManager installs the cert manager bundle.
func InstallCertManager() error {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "apply", "-f", url)
	if _, err := Run(cmd); err != nil {
		return err
	}
	// Wait for cert-manager-webhook to be ready, which can take time if cert-manager
	// was re-installed after uninstalling on a cluster.
	cmd = exec.Command("kubectl", "wait", "deployment.apps/cert-manager-webhook",
		"--for", "condition=Available",
		"--namespace", "cert-manager",
		"--timeout", "5m",
	)

	_, err := Run(cmd)
	return err
}

func InstallEtcdOperator() error {
	url := fmt.Sprintf(etcdOperatorURL, etcdOperatorVersion, etcdOperatorVersion)
	cmd := exec.Command("kubectl", "apply", "-f", url)
	if _, err := Run(cmd); err != nil {
		return err
	}
	return nil
}

func UninstallEtcdOperator() {
	url := fmt.Sprintf(etcdOperatorURL, etcdOperatorVersion, etcdOperatorVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

func SetupEtcdCluster(namespace string) error {
	etcdClusterPath := filepath.Join("config", "samples", "etcd_cluster.yaml")
	cmd := exec.Command("kubectl", "apply", "-f", etcdClusterPath, "--namespace", namespace)
	if _, err := Run(cmd); err != nil {
		return err
	}

	// Wait for etcd-cluster-0 to be created
	cmd = exec.Command("kubectl", "wait", "pod/etcd-cluster-2", "--for=create",
		"--timeout=5m", "--namespace", namespace)

	_, err := Run(cmd)
	if err != nil {
		return err
	}

	// Wait for the etcd cluster to be ready
	// Note: this is a hardcoded name for the etcd cluster, if the name is changed, the test setup will fail
	cmd = exec.Command("kubectl", "wait", "pod/etcd-cluster-2", "--for=condition=Ready",
		"--timeout=5m", "--namespace", namespace)
	_, err = Run(cmd)
	if err != nil {
		return err
	}

	// Expect all etcd pods to be ready
	cmd = exec.Command("kubectl", "exec", "etcd-cluster-0", "--namespace", namespace, "--", "etcdctl", "member", "list")
	result, err := Run(cmd)
	if err != nil {
		return err
	}

	// Verify we have exactly 3 "started" entries
	startedCount := strings.Count(string(result), "started")
	if startedCount != 3 {
		return fmt.Errorf("expected 3 'started' entries in etcd member list, got %d", startedCount)
	}

	return nil
}

func BuildingOperatorImage(projectimage string) error {
	fmt.Println("building the manager(Operator) image")
	cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectimage))
	_, err := Run(cmd)
	if err != nil {
		return err
	}

	return nil
}

func InstallingOperator(projectimage string, kindClusterName string, kindCluster support.E2EClusterProvider) error {

	fmt.Println("installing the Operator CRD")
	cmd := exec.Command("make", "install")
	_, err := Run(cmd)
	if err != nil {
		return err
	}

	fmt.Println("deploying the controller-manager")
	cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectimage))
	_, err = Run(cmd)
	if err != nil {
		return err
	}

	return nil
}

func VerifyControllerUp(namespace string) error {
	fmt.Println("validating that the controller-manager pod is running as expected")
	cmd := exec.Command("kubectl", "get",
		"pods", "-l", "control-plane=controller-manager",
		"-o", "go-template={{ range .items }}"+
			"{{ if not .metadata.deletionTimestamp }}"+
			"{{ .metadata.name }}"+
			"{{ \"\\n\" }}{{ end }}{{ end }}",
		"-n", namespace,
	)

	podOutput, err := Run(cmd)
	if err != nil {
		return err
	}

	podNames := GetNonEmptyLines(string(podOutput))
	if len(podNames) != 1 {
		return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
	}
	controllerPodName := podNames[0]

	// Validate pod status
	cmd = exec.Command("kubectl", "get",
		"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
		"-n", namespace,
	)
	status, err := Run(cmd)
	if err != nil {
		return err
	}

	if string(status) != "Running" {
		return fmt.Errorf("controller pod in %s status", status)
	}
	return nil
}

// GetNonEmptyLines converts given command output string into individual objects
// according to line breakers, and ignores the empty elements in it.
func GetNonEmptyLines(output string) []string {
	var res []string
	elements := strings.Split(output, "\n")
	for _, element := range elements {
		if element != "" {
			res = append(res, element)
		}
	}

	return res
}

// GetProjectDir will return the directory where the project is
func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return wd, err
	}
	wd = strings.Replace(wd, "/test/e2e", "", -1)
	return wd, nil
}
