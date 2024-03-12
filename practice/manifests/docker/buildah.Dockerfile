FROM quay.io/buildah/stable AS buildah 

# run following commands in buildah container
# newcontainer=$(buildah from scratch)
# buildah add $newcontainer /var/lib/kubelet/checkpoints/checkpoint-<pod-name>_<namespace-name>-<container-name>-<timestamp>.tar /
# buildah config --annotation=io.kubernetes.cri-o.annotations.checkpoint.name=<container-name> $newcontainer
# buildah commit $newcontainer checkpoint-image:latest
# buildah rm $newcontainer

# Path: manifests/docker/Dockerfile
RUN newcontainer=$(buildah from scratch) && \
    buildah add $newcontainer /var/lib/kubelet/checkpoints/checkpoint-counters_default-counter-2024-03-04T12:20:30Z.tar / && \
    buildah config --annotation=io.kubernetes.cri-o.annotations.checkpoint.name=<container-name> $newcontainer && \
    buildah commit $newcontainer checkpoint-image:latest && \
    buildah rm $newcontainer

    
