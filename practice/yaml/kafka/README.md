### this is for the page about deploying strimzi kafka operator to your cluster
#### because I don't care the message could be lost after commiting the image, I prefer not to use the persistent volume to store messages

> execute ./deploy.sh

**check the kafka operator works or not**: 
> you can follow the following link to debug:

https://strimzi.io/quickstarts/

**If kafka broker comes into some issues like the error mentioned here *https://github.com/strimzi/strimzi-kafka-operator/issues/10040*,a.k.a the dns resolution issue, please try the following command:**

> kubectl rollout restart deployment coredns -n kube-system 
