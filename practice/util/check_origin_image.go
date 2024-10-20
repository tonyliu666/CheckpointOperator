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
	checkJobStatusTime = 30 // the checktime of the job status, you could adjust here
)

func intUtility(x int64) *int64 {
	return &x
}
func boolUtility(x bool) *bool {
	return &x
}

func CheckImageIDExistOnNode(imageIDList []string, dstNode string)  (*batchv1.Job,error) {
	clientset, err := CreateClientSet()
	if err != nil {
		return nil,err
	}
	args := []string{"-c"}
	// Initialize an empty command string
	commandString := ""

	// Loop over the imageIDList and construct the command string
	for _, imageID := range imageIDList {
		commandString += "buildah pull " + imageID + " && "
	}

	// Remove the last ' && ' from the command string
	commandString = commandString[:len(commandString)-4]
	args = append(args, commandString)
	// Create a pod to run the command on the node
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			// show the time unit in milliseconds
			Name: "check-image-id" + time.Now().Format("2006-01-02-15-04-05-000"),
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: func() *int32 { i := int32(10); return &i }(),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "check-image-id",
							Image:   "quay.io/buildah/stable",
							Command: []string{"/bin/sh"},
							Args:    args,
							// command is like crictl pull docker.io/library/mongo@sha256:24ecfe95bbb98cd49e1d968c204515d4033ef86621e68ce148cf1d0a5a0f8a82
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/var/lib/containers/storage",
									Name:      "container-storage-graphroot",
								},
								{
									MountPath: "/run/containers/storage",
									Name:      "container-storage-runroot",
								},
								{
									MountPath: "/etc/containers/registries.conf.d",
									Name:      "container-storage-conf",
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: boolUtility(true),
								RunAsUser:  intUtility(0),
								RunAsGroup: intUtility(0),
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "container-storage-graphroot",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/containers/storage",
								},
							},
						},
						{
							Name: "container-storage-runroot",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/run/containers/storage",
								},
							},
						},
						{
							Name: "container-storage-conf",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/containers/registries.conf.d",
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
	newjob, err := clientset.BatchV1().Jobs("docker-registry").Create(context.TODO(), job, metav1.CreateOptions{})

	if err != nil {
		return nil,err
	}
	return newjob,nil
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
			// if the error is like "jobs.batch \"check-image-id-1729355541374\ then ignore it
			if err.Error() == "jobs.batch \"" + jobName + "\" not found" {
				if times > checkJobStatusTime {
					return fmt.Errorf("job checktime exceeded")
				} else {
					times++
					continue
				}
			}else{
				return err
			}
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
