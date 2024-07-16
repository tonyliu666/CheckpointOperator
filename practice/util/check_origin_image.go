package util

import (
	"context"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	checkJobStatusTime = 60
)

func CheckImageIDExistOnNode(imageIDList []string, dstNode string) error {
	clientset, err := CreateClientSet()
	if err != nil {
		return err
	}
	command := []string{}
	// loop over the imageIDMap and check if the image exists on the node
	for _, imageID := range imageIDList {
		command = append(command, "crictl pull "+imageID)
	}
	// Create a pod to run the command on the node
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "check-image-id",
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "check-image-id",
							Image:   "alpine",
							Command: command,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "crio-socket",
									MountPath: "/var/run/crio/crio.sock",
									SubPath:   "crio.sock",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "crio-socket",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/run/crio/crio.sock",
									Type: new(corev1.HostPathType), // Defaults to Socket if not specified
								},
							},
						},
					},
					NodeName:           dstNode,
					ServiceAccountName: "default",
					RestartPolicy:      corev1.RestartPolicyNever,
				},
			},
		},
	}
	// Create the job
	_, err = clientset.BatchV1().Jobs("docker-registry").Create(context.TODO(), job, metav1.CreateOptions{})

	if err != nil {
		return err
	}
	return nil
}

func CheckJobStatus(jobName string, status string) error {
	clientset, err := CreateClientSet()
	if err != nil {
		return err
	}
	// Check the status of the job
	// set every ten seconds to check the status
	times := 0
	for {
		time.Sleep(10 * time.Second)
		job, err := clientset.BatchV1().Jobs("docker-registry").Get(context.TODO(), jobName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if job.Status.Succeeded == 1 {
			break
		}
		if times > checkJobStatusTime {
			return fmt.Errorf("job checktime exceeded")
		} else {
			times++
		}
	}
	return nil
}
