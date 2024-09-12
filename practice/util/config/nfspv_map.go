package config

import (
	corev1 "k8s.io/api/core/v1"
)

var (
	// create pv_maps whose key is ip address of each node and value is pv itself
	// create pvc_maps whose key is ip address of each node and value is pvc itself
	PvSourceMap  map[string]*corev1.PersistentVolume
	PvcSourceMap map[string]*corev1.PersistentVolumeClaim
)
