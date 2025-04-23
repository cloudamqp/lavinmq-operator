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

package controller

import (
	"os"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	testutils "lavinmq-operator/internal/test_utils"
	// +kubebuilder:scaffold:imports
)

var testEnv *envtest.Environment
var k8sClient client.Client

func TestMain(m *testing.M) {
	testEnv, k8sClient = testutils.StartKubeTestEnv()

	code := m.Run()

	logf.Log.Info("Tearing down test suite")

	err := testEnv.Stop()
	if err != nil {
		logf.Log.Error(err, "Failed to stop test environment")
		os.Exit(1)
	}

	os.Exit(code)
}
