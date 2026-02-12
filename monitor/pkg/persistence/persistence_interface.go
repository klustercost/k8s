package persistence

import "klustercost/monitor/pkg/model"

type Persistence interface {
	InsertNode(string, *model.NodeMisc) error
	InsertOwner(string, string, *model.AppOwnerReferences) error
	InsertPod(string, string, string, *model.PodConsumption, *model.PodAppLabels) error
	InsertService(string, string, *model.ServiceMisc) error
}
