#! /bin/bash
# create kafka namespace if not exists
kubectl create ns kafka
kubectl create -f 'https://strimzi.io/install/latest?namespace=kafka' -n kafka
kubectl create -f yaml/kafka/kafka-ephemeral-single.yaml