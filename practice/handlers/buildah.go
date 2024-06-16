package handlers

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	util "tony123.tw/util"
)

func BuildahPodPushImage(index int,nodeName string, nameSpace string, checkpoint string, registryIp string) error {
	podName := util.ModifyCheckpointToImageName(checkpoint)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "buildah-job-"+fmt.Sprintf("%d", index),
			// TODO: change the name if I want to sent all the images of deployment to the registry
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
								"newcontainer=$(buildah from scratch); buildah add $newcontainer " + checkpoint + "  /" + ";buildah config --annotation=io.kubernetes.cri-o.annotations.checkpoint.name="+podName+" $newcontainer; buildah commit --format oci $newcontainer /var/lib/kubelet/oci-archive/"+podName+"; "+ "buildah rm $newcontainer;" ,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "checkpointed-image",
									MountPath: "/var/lib/kubelet/checkpoints/",
								},
								{
									Name: "oci-archive",
									MountPath: "/var/lib/kubelet/oci-archive/",
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
						{
							Name: "oci-archive",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/oci-archive/",
									Type: func() *corev1.HostPathType { t := corev1.HostPathDirectoryOrCreate; return &t }(),
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
	_, err = clientset.BatchV1().Jobs(nameSpace).Create(context.TODO(), job, metav1.CreateOptions{})
	return err
}


func DeleteBuildahJobs(clientset *kubernetes.Clientset) error {
	//check the job existed in docker-registry namespace
	jobs, err := clientset.BatchV1().Jobs("docker-registry").List(context.TODO(), metav1.ListOptions{})
	// delete the all the completed jobs in jobs
	if err != nil || len(jobs.Items) == 0 {
		return nil
	}
	if err != nil {
		return err
	}
	backgroundDeletion := metav1.DeletePropagationBackground
	for _, job := range jobs.Items {
		if job.Status.Succeeded == 1 {
			clientset.BatchV1().Jobs("docker-registry").Delete(context.TODO(), job.Name, metav1.DeleteOptions{
				PropagationPolicy: &backgroundDeletion, 
			})
		}
	}
	return nil
}
