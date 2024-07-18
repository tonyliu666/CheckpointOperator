## this folder is for the docker-registry webhook handler deployed as a pod in docker-registry namespace

* you could create the imagePullSecret if you would like to pull image from dockerHub.
* The sample is listed in create-pull-fake-secret.sh

1. kubectl apply -f testwebhook/registry-role.yaml
2. docker build -t testwebhook .
3. kubectl apply -f testwebhook/webhooktest.yaml