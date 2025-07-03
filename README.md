# LavinMQ-operator

## Description
The LavinMQ Operator provides a way to deploy and manage LavinMQ clusters on Kubernetes. It leverages Kubernetes Custom Resource Definitions (CRDs) to define and manage LavinMQ instances as native Kubernetes resources.

Supported features:

- LavinMQ version upgrades
- Scaling - Horizontal and vertical.
- Increasing disk size
- Setting LavinMQ specific configurations. Rolling restarts automatically applied.

Known issues/limitations/roadmap:

- Scaling disk is only supporting disk size increase
- Updating TLS contents do not apply a restart/reload of LavinMQs state. A manual restart/reapply is needed
- ETCD has to be managed by end-user and is not provided via this operator. It is on the roadmap though
  - There is a example in config/samples/etcd_cluster.yaml to setup an etcd cluster using https://github.com/etcd-io/etcd-operator, the operator has to be pre-installed to use this.
  - A meta operator is being considered to manage the etcd and lavinmq simultaneously, see related issue https://github.com/cloudamqp/lavinmq-operator/issues/39
- Monitoring capability is currently limited to what LavinMQ itself provides.

Following [operator-sdks Capability Levels](https://sdk.operatorframework.io/docs/overview/operator-capabilities/), the operator can be considered as a Level 3 implementation currently.

## Supported Configurations

The LavinMQ Operator supports a wide range of configurations, as defined in the `lavinmq_types.go` file. Here's a detailed description of the supported features and configurations:

1. **Image Configuration:**
   - The operator allows specifying a custom Docker image for LavinMQ using the `image` field. By default, it uses `cloudamqp/lavinmq:2.3.0`.

2. **Replicas:**
   - You can configure the number of replicas for the LavinMQ cluster. The value must be between 1 and 3, with a default of 1.

3. **Resource Management:**
   - `resources` field allows specifying CPU and memory requests/limits for the LavinMQ pods.

4. **Persistent Storage:**
   - `dataVolumeClaim` field is required and defines the PersistentVolumeClaim (PVC) for storing data. It enforces the `ReadWriteOnce` access mode.

5. **Etcd Integration:**
   - `etcdEndpoints` field allows specifying a list of etcd endpoints for clustering. Required if running more than a single node of LavinMQ

6. **TLS Configuration:**
   - `tlsSecret` field references a Kubernetes Secret containing TLS certificates for secure communication.

7. **LavinMQ Configuration:**
   - The `config` field allows detailed customization of LavinMQ behavior through the following sub-configurations, see [LavinMQ Configuration documentation](https://lavinmq.com/documentation/configuration-files) for extended list of configurations
     - **Main Configuration:**
       - Consumer timeout, default prefetch, default user/password, disk space thresholds, logging levels, and more.
     - **Mgmt Configuration:**
       - HTTP/HTTPS management ports and configurations related to the UI.
     - **AMQP Configuration:**
       - Channel limits, frame size, heartbeat intervals, and AMQP/AMQPS ports, etc...
     - **MQTT Configuration:**
       - In-flight message limits and MQTT/MQTTS ports.
     - **Clustering Configuration:**
       - Maximum unsynced actions in the cluster.

## Provided examples
In `config/samples/`

## Getting Started

### Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.
- Permission enough to modify RBAC rules (cluster-admin)

### To Deploy on the cluster

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://github.com/cloudamqp/lavinmq-operator/releases/download/<version>/install.yaml
```

Images are built and stored in GitHub assets upon new releases.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out. For ETCD example to work, you also need to install the ETCD operator

### To Uninstall
**Undeploy the controller from the cluster:**

```sh
kubectl delete -f https://github.com/cloudamqp/lavinmq-operator/releases/download/<version>/install.yaml
```

## Development
### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/lavinmq-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/lavinmq-operator:tag
```

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

## Project Distribution

Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/lavinmq-operator:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies. The installation yaml is for this repo stored in GitHub Release assets.

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

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

