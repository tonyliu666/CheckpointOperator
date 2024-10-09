# CheckpointRestore Operator
* This project is used for pod migration between different nodes. It utilize the CRIU(checpoint-restore in userspace) linux kernel module to keep the state of the running container, and then recreate a new pod with wrapping the checkpoint state on the new node. Therefore, the new pod could be successfully restored.

## Description
1. MigrationController: it's used for monitoring the change of the state of CR(custom resoure). When the state of CR changes, then the controller will do the ongoing processes, like creating the corresponding client with given the certificate in /etc/kubernetes/pki folder in master node to enable the controller to access the kubelet checkpoint api which is located at **https://"workernode-ip"+10250**. The full command to access the kubelet api endpoint is **curl -X POST "https://localhost:10250/checkpoint/namespace/"podId"/"ContainerName"**
2. DaemonSetController: It is for monitoring the private docker registry pod deployed on each node to examine whether the checkpointed image has been pushed to the registry pod which is located at the destination node. I deploy a **Strimzi**, kafka broker service deployed on Kubernetes on the cluster. When the checkpointed image pushed to the registry pod, the webhook of the registry will be sent to the event handler deployed as a pod on the same node as docker private registry pod. Then that handler will create a kafka message then send it to the kafka service broker to let the controller to receive. Finally the controller will create a new pod with the checkpointed image on the registry pod on the destination node. 

## Getting Started

### the order of deployments: 
- make docker-build docker-push
- make manifests
- make install 
- make deploy

### Environments
- go version 1.22.4
- docker version 24.0.5
- kubectl version v1.28.9+.
- Access to a Kubernetes v1.28.9+ cluster.

> **Notes** the checkpoint restore funcationality only supports the kubernetes v1.25.0+ version

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/practice:tag
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
make deploy IMG=<some-registry>/practice:tag
```

> **NOTE**: After using make deploy command, then type this command on your master node:
> kubectl -n practice-system  create secret generic kubelet-client-certs --from-file=client.crt=/etc/kubernetes/pki/apiserver-kubelet-client.crt --from-file=client.key=/etc/kubernetes/pki/apiserver-kubelet-client.key --kubeconfig=/home/ubuntu/.kube/config

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin 
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
kubectl apply -f config/samples/api_v1alpha1_migration.yaml
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

#### If you encounter the too many request when pulling the image from dockerHub, then create the secret token: 

### before using helm to deploy your application, please execute this command: 

> kubectl create secret docker-registry regcred \
  --docker-server="https://index.docker.io/v1/" \
  --docker-username="<Username>" \
  --docker-password="<AccessToken>" \
  --docker-email="<Email>"

> AccessToken: you can create this from the docker hub in the security tab

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/practice:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/practice/<tag or branch>/dist/install.yaml
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.



![structue](https://github.com/tonyliu666/CheckpointOperator/assets/48583047/726138ab-f8d7-4c06-8b16-63b539d77381)
