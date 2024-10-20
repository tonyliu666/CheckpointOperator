package handlers

import (
	"context"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	util "tony123.tw/util"
)

func int64Ptr(i int64) *int64 { return &i }

func BuildahPodPushImage(nodeName string, nameSpace string, checkpoint string, registryIp string) error {
	podName := util.ModifyCheckpointToImageName(checkpoint)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "buildah-job-" + time.Now().Format("2006-01-02-15-04-05-000"),
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: func() *int32 { i := int32(30); return &i }(),
			BackoffLimit: func() *int32 { i := int32(1); return &i }(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "buildah"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "buildah",
							Image: "quay.io/buildah/stable",
							SecurityContext: &corev1.SecurityContext{
								Privileged: func() *bool { b := true; return &b }(),
								RunAsUser:  int64Ptr(0),
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"SYS_ADMIN"},
								},
							},
							Command: []string{"/bin/bash"},
							Args: []string{
								"-c",
								"newcontainer=$(buildah from scratch); buildah add $newcontainer " + checkpoint +
									" /; buildah config --annotation=io.kubernetes.cri-o.annotations.checkpoint.name=" + podName +
									" $newcontainer; buildah commit $newcontainer " + podName + ":latest; buildah rm $newcontainer; " +
									"buildah push --creds=myuser:mypasswd --tls-verify=false localhost/" + podName + ":latest " +
									registryIp + ":5000/" + podName + ":latest;",
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "checkpointed-image", MountPath: "/var/lib/kubelet/checkpoints/"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "checkpointed-image",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{Path: "/var/lib/kubelet/checkpoints/"},
							},
						},
					},
					NodeName:      nodeName,
					RestartPolicy: corev1.RestartPolicyNever,
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
