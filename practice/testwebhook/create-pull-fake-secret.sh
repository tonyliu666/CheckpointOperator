#! /bin/bash
# create the following secret template of your own: 
kubectl create secret docker-registry regcred `
  --docker-server="https://index.docker.io/v1/" `
  --docker-username="<Username>" `
  --docker-password="<AccessToken>" `
  --docker-email="<Email>"