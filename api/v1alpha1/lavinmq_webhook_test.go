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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LavinMQ Webhook", func() {

	Context("When creating LavinMQ", func() {
		It("Should accept default", func() {
			lavin := &LavinMQ{}
			_, err := lavin.ValidateCreate(nil, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should accept clustering with etcd", func() {
			lavin := &LavinMQ{Spec: LavinMQSpec{
				Replicas:      3,
				EtcdEndpoints: []string{"http://etcd-cluster:2379"},
			},
			}
			_, err := lavin.ValidateCreate(nil, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should deny if a clustering without etcd", func() {
			lavin := &LavinMQ{Spec: LavinMQSpec{
				Replicas: 3,
			},
			}
			_, err := lavin.ValidateCreate(nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("a provided etcd cluster is required for replication"))
		})
	})
	Context("When updating LavinMQ", func() {
		It("Should accept default", func() {
			lavin := &LavinMQ{}
			_, err := lavin.ValidateCreate(nil, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should accept clustering with etcd", func() {
			lavin := &LavinMQ{Spec: LavinMQSpec{
				Replicas:      3,
				EtcdEndpoints: []string{"http://etcd-cluster:2379"},
			},
			}
			_, err := lavin.ValidateCreate(nil, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should deny if a clustering without etcd", func() {
			lavin := &LavinMQ{Spec: LavinMQSpec{
				Replicas: 3,
			},
			}
			_, err := lavin.ValidateCreate(nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("a provided etcd cluster is required for replication"))
		})
	})

})
