package handlers

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	util "tony123.tw/util"
	"tony123.tw/util/config"
)

func intUtility(x int64) *int64 {
	return &x
}
func boolUtility(x bool) *bool {
	return &x
}

func BuildahPodPushImage(originPodName string, nameSpace string, checkpoint string, srcNodeIP string, dstNode string) error {
	// please follow the given yaml contents to create a specific job
	fileName := util.ModifyCheckpointToFileName(checkpoint)
	podName := util.ModifyCheckpointToImageName(checkpoint)
	
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("buildah-job-%s", originPodName),
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: func() *int32 { i := int32(3); return &i }(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("buildah-pod-%s", originPodName),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "buildah",
							Image:   "quay.io/buildah/stable",
							Command: []string{"/bin/sh"},
							Args: []string{
								"-c",
								fmt.Sprintf("newcontainer=$(buildah from scratch); if [ -f /mnt/checkpoints/%s ]; then buildah add $newcontainer /mnt/checkpoints/%s /; buildah config --annotation=io.kubernetes.cri-o.annotations.checkpoint.name=%s $newcontainer; buildah commit --log-level=debug $newcontainer  %s:latest; buildah rm $newcontainer;  else echo 'File not found'; exit 1; fi", fileName, fileName, podName, podName),
								// sleep infinity
								// "sleep infinity",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/mnt/checkpoints",
									Name:      "checkpoint-storage",
								},
								{
									MountPath: "/var/lib/containers/storage",
									Name:      "container-storage-graphroot",
								},
								{
									MountPath: "/run/containers/storage",
									Name:      "container-storage-runroot",
								},
								// {
								// 	MountPath: "/etc/containers/registries.conf",
								// 	Name:      "container-storage-conf",
								// },
								// {
								// 	MountPath: "/etc/containers/policy.json",
								// 	Name:      "container-policy",
								// },
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
							Name: "checkpoint-storage",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: config.PvcSourceMap[srcNodeIP].Name,
								},
							},
						},
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
						// TODO: remove the following settings since kubernetes 1.28.0, uncomment them if you are using kubernetes version < 1.28.0
						// {
						// 	Name: "container-storage-conf",
						// 	VolumeSource: corev1.VolumeSource{
						// 		HostPath: &corev1.HostPathVolumeSource{
						// 			Path: "/etc/containers/registries.conf",
						// 		},
						// 	},
						// },
						// {
						// 	Name: "container-policy",
						// 	VolumeSource: corev1.VolumeSource{
						// 		HostPath: &corev1.HostPathVolumeSource{
						// 			Path: "/etc/containers/policy.json",
						// 		},
						// 	},
						// },
					},
					NodeName:      dstNode,
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}
	// create the job in the given namespace
	clientset, err := util.CreateClientSet()
	if err != nil {
		return err
	}
	_, err = clientset.BatchV1().Jobs(nameSpace).Create(context.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Log.Info("Job created successfully", "jobName", job.Name)
	return nil
}
