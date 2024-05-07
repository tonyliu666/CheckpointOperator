package handlers

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"tony123.tw/util"
)

func BuildahPodPushImage(nodeName string, nameSpace string, checkpoint string, registryIp string) error {
	podName := util.ModifyCheckpointToImageName(checkpoint)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "buildah-job",
			// set the same namespace as the pod
		},
		// set ttlSecondsAfterFinished to 30 seconds
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: func() *int32 { i := int32(20); return &i }(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "buildah",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "buildah",
							Image: "quay.io/buildah/stable",
							SecurityContext: &corev1.SecurityContext{
								Privileged: func() *bool { b := true; return &b }(),
							},
							Command: []string{"/bin/bash"},
							// builah add the file under checkpointed-image to the new container
							Args: []string{
								"-c",
								"newcontainer=$(buildah from scratch); buildah add $newcontainer " + checkpoint + "  /" + "; buildah commit $newcontainer " + podName + ":latest; buildah rm $newcontainer; buildah push --creds=myuser:mypasswd --tls-verify=false localhost/" + podName + ":latest " + registryIp + ":5000/" + podName + ":latest;",
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
					NodeName:      nodeName,
					RestartPolicy: corev1.RestartPolicyNever, // Ensure the job doesn't restart
				},
			},
		},
	}

	clientset, err := util.CreateClientSet()
	if err != nil {
		return err
	}
	// _, err = clientset.AppsV1().Deployments(nameSpace).Create(context.TODO(), deployment, metav1.CreateOptions{})
	_, err = clientset.BatchV1().Jobs(nameSpace).Create(context.TODO(), job, metav1.CreateOptions{})
	return err
}

func DeleteBuildahDeployment(clientset *kubernetes.Clientset) error {
	return clientset.AppsV1().Deployments("docker-registry").Delete(context.TODO(), "buildah-deployment", metav1.DeleteOptions{})
}
