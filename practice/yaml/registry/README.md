### deploy docker private registry as daemonsets in cluster

1. kubectl apply -f yaml/registry/registry-config/webookconfigmap.yaml
2. kubectl apply -f yaml/registry/daemonsets.yaml