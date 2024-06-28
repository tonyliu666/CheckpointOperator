### this restore folder is for deploying a handler to delete the old pod on the original node if the new pod is successfully deployed on the new node

command: 
1. docker build -t restore .
2. kubectl apply -f restore-role.yaml
3. kubectl apply -f restore-handler.yaml