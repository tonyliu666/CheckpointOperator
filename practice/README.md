# CheckpointRestore Operator
* This project is used for pod migration between different nodes. It utilize the CRIU(checpoint-restore in userspace) linux kernel module to keep the state of the running container, and then recreate a new pod with wrapping the checkpoint state on the new node. Therefore, the new pod could be successfully restored.

## Description
* In this branch, I only use this controller to handle the migration process
1. MigrationController: it's used for monitoring the change of the state of CR(custom resoure). When the state of CR changes, then the controller will do the ongoing processes, like creating the corresponding client with given the certificate in /etc/kubernetes/pki folder in master node to enable the controller to access the kubelet checkpoint api which is located at **https://"workernode-ip"+10250**. The full command to access the kubelet api endpoint is **curl -X POST "https://localhost:10250/checkpoint/namespace/"podId"/"ContainerName"**

## Design Principle: 
* Before the MigrationController starts to work, I first create each pair of PersistentVolume(PV) and PersistentVolumeClaim(PVC) on each node in "Migration" namespace
* And each PV define the volume source which is NFS server mounting on the /var/lib/kubelet/checkpoints where the node place the checkpoint file of each container

## Getting Started

### Environments
- go version 1.22.4
- kubectl Client Version: v1.28.9, Server Version: v1.30.0
- Kubernetes v1.30.0 cluster

> **Notes** the checkpoint restore funcationality only supports the kubernetes v1.25.0+ version
> The link: https://kubernetes.io/blog/2022/12/05/forensic-container-checkpointing-alpha/

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

**Deploy the Controller(Manager) to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/practice:tag
```

> **NOTE**: After using make deploy command, then type this command on your master node:
> kubectl -n practice-system  create secret generic kubelet-client-certs --from-file=client.crt=/etc/kubernetes/pki/apiserver-kubelet-client.crt --from-file=client.key=/etc/kubernetes/pki/apiserver-kubelet-client.key --kubeconfig=/home/ubuntu/.kube/config

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin 
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the CustomResource (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
or
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

> kubectl create secret docker-registry regcred \
  --docker-server="https://index.docker.io/v1/" \
  --docker-username="<Username>" \
  --docker-password="<AccessToken>" \
  --docker-email="<Email>"

> AccessToken: you can create this from the docker hub in the security tab

* Optionally, you could deploy your own docker registry for storing your controller image by using the command: 
> make upregistry
> And change the insecure registry setting of crio in each node(under the section[crio.image] in /etc/crio/crio.conf file)
> finally, restart crio by using
>> systemctl restart crio

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

Copyright 2024. TonyLIU

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
