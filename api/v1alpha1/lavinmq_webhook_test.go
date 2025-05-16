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

package v1alpha1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateDefault(t *testing.T) {
	t.Parallel()
	lavin := &LavinMQ{}
	_, err := lavin.ValidateCreate(context.TODO(), nil)
	assert.NoErrorf(t, err, "Failed to validate update")
}

func TestCreateClusterWithEtcd(t *testing.T) {
	t.Parallel()
	lavin := &LavinMQ{Spec: LavinMQSpec{
		Replicas:      3,
		EtcdEndpoints: []string{"http://etcd-cluster:2379"},
	},
	}
	_, err := lavin.ValidateCreate(context.TODO(), nil)
	assert.NoErrorf(t, err, "Failed to validate create")

}
func TestCreateClusterWithoutEtcd(t *testing.T) {
	t.Parallel()
	lavin := &LavinMQ{Spec: LavinMQSpec{
		Replicas: 3,
	},
	}
	_, err := lavin.ValidateCreate(context.TODO(), nil)
	assert.Errorf(t, err, "Expected error when creating cluster without etcd")
	assert.Equal(t, err.Error(), "a provided etcd cluster is required for replication")

}
func TestUpdateDefault(t *testing.T) {
	t.Parallel()
	oldLavinMQ := &LavinMQ{}
	newLavinMQ := &LavinMQ{}
	_, err := newLavinMQ.ValidateUpdate(context.TODO(), newLavinMQ, oldLavinMQ)
	assert.NoErrorf(t, err, "Failed to validate update")
}
