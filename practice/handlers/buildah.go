package handlers

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"tony123.tw/util"
)

func BuildahPodPushImage(nodeName string, nameSpace string, checkpoint string, registryIp string) error {
	num := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "buildah-deployment",
			// set the same namespace as the pod
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &num,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "buildah",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "buildah",
					},
				},
				// specify the pod should be created on the node where the pod is running
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							// privileged container
							Name:  "buildah",
							Image: "quay.io/buildah/stable",
							SecurityContext: &corev1.SecurityContext{
								Privileged: func() *bool { b := true; return &b }(),
							},
							Command: []string{"/bin/bash"},

							Args: []string{
								"-c",
								"newcontainer=$(buildah from scratch); buildah add $newcontainer " + checkpoint + "; buildah config --annotation=io.kubernetes.cri-o.annotations.checkpoint.name=default-counter $newcontainer; buildah commit $newcontainer checkpoint-image:latest; buildah rm $newcontainer; buildah push --creds=myuser:mypasswd --tls-verify=false localhost/checkpoint-image:latest " + registryIp + ":5000/checkpoint-image:latest;",
								// buildah push checkpoint-image to nodeIP:nodePort with username and password, username=myuser and password=mypasswd
								// buildah push --creds=myuser:mypasswd --tls-verify=false localhost/checkpoint-image:latest 10.85.0.8:5000/checkpoint-image:latest
							},

							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "checkpointed-image",
									MountPath: "/var/lib/kubelet/checkpoints/",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "checkpointed-image",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/checkpoints/",
								},
							},
						},
					},
					NodeName: nodeName,
				},
			},
		},
	}
	clientset,err := util.CreateClientSet()
	if err != nil {
		return err
	}
	_, err = clientset.AppsV1().Deployments(nameSpace).Create(context.TODO(), deployment, metav1.CreateOptions{})
	return err
}
