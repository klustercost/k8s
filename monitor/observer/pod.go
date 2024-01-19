package main

import (
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

func (c *Controller) initPodCollector(namespace, name string) (*v1.Pod, error) {
	pod, err := c.podsLister.Pods(namespace).Get(name)
	if err != nil {
		klog.Error("Error getting pod lister ", err)
	}

	return pod, err
}

// Returns owner_version, owner_kind, owner_name, owner_uid
func (c *Controller) returnOwnerReferences(pod *v1.Pod) (string, string, string, string) {
	owner := pod.ObjectMeta.OwnerReferences
	for _, v := range owner {
		if v.Name != "" {
			return v.APIVersion, v.Kind, v.Name, string(v.UID)
		}
	}
	return "", "", "", ""
}

// This function retrieves the record_time, owner, node_name
// It queries the API server
func (c *Controller) getPodMiscellaneous(pod *v1.Pod) (time.Time, string, string) {
	record_time := time.Now()
	owner := pod.ObjectMeta.OwnerReferences
	var owner_name string

	for _, v := range owner {
		if v.Name != "" {
			owner_name = string(v.UID)

		} else {
			owner_name = "No owner"
		}

	}
	node_name := pod.Spec.NodeName

	return record_time, owner_name, node_name

}

// This function retrieves the memory and CPU usage of a pod
// It queries the metrics server
func (c *Controller) getPodConsumption(pod *v1beta1.PodMetrics) (int64, int64) {
	// Calculate total memory usage for the entire pod
	var totalMemUsageBytes int64
	var totalCPUUsageMili int64

	for i := 0; i < len(pod.Containers); i++ {
		totalMemUsageBytes += pod.Containers[i].Usage.Memory().Value()
		totalCPUUsageMili += pod.Containers[i].Usage.Cpu().MilliValue()
	}

	return totalMemUsageBytes, totalCPUUsageMili
}
