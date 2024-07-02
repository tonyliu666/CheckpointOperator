package handlers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	util "tony123.tw/util"
)

func DeployPodOnNewNode(pod *corev1.Pod, nameSpace string, dstNode string) error {
	// deploy a new pod on the destination node
	migratePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "checkpoint-"+pod.Name,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "checkpoint-"+pod.Name,
					Image: "checkpoint-" + pod.Name + ":latest",
				},
			},
			NodeName: dstNode,
		},
	}

	clientset, err := util.CreateClientSet()
	if err != nil {
		log.Log.Error(err, "unable to create clientset", err)
		return fmt.Errorf("unable to create clientset: %w", err)
	}
	// check the buildah pod in migration namespace whose state is completed
	err = util.CheckPodStatus(pod.Name,"Succeeded","migration", 30)
	if err != nil {
		log.Log.Error(err, "unable to check pod status")
		return fmt.Errorf("unable to check pod status: %w", err)
	}

	// TODO: replace default namespace with the namespace of the pod
	newpod, err := clientset.CoreV1().Pods(nameSpace).Create(context.TODO(), migratePod, metav1.CreateOptions{})
	if err != nil {
		log.Log.Error(err, "unable to deploy new pod")
		return fmt.Errorf("unable to create pod: %w", err)
	}
	log.Log.Info("new pod created", "pod", newpod.Name)
	return nil
}
