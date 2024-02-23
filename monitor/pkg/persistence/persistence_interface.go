package persistence

import "klustercost/monitor/pkg/model"

type Persistence interface {
	InsertNode(string, *model.NodeMisc) error
	InsertOwner(string, string, *model.AppOwnerReferences) error
	InsertPod(string, string, *model.PodMisc, *model.OwnerReferences, *model.PodConsumption) error
	InsertService(string, string, *model.ServiceMisc) error
}

// InsertNode is a function that inserts the details of a node into the database
func InsertNode(p Persistence, nodeName string, nodeMisc *model.NodeMisc) error {
	return p.InsertNode(nodeName, nodeMisc)
}

func InsertOwner(p Persistence, name string, namespace string, allRef *model.AppOwnerReferences) error {
	return p.InsertOwner(name, namespace, allRef)
}

func InsertPod(p Persistence, pod_name, namespace string, podMisc *model.PodMisc, ownerRef *model.OwnerReferences, podUsage *model.PodConsumption) error {
	return p.InsertPod(pod_name, namespace, podMisc, ownerRef, podUsage)
}

func InsertService(p Persistence, name string, namespace string, svcRef *model.ServiceMisc) error {
	return p.InsertService(name, namespace, svcRef)
}
