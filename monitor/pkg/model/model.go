package model

import (
	"time"

	promModel "github.com/prometheus/common/model"
)

// NodeMisc is a struct that contains the node miscellaneous information
// It is used to insert data into the database
// Used by node-controller.go
type NodeMisc struct {
	Memory int64
	CPU    int64
	UID    string
	Labels string
}

// This struct is used to store the memory and CPU usage of a pod
// It is used to insert data into the database
// Used by pod-controller.go
type PodConsumption struct {
	Memory *promModel.Sample
	CPU    *promModel.Sample
}

// This struct is used to store the owner_version, owner_kind, owner_name, owner_uid of a *v1.Pod
// It is used to insert data into the database
// Used by pod-controller.go and app-controller.go
type OwnerReferences struct {
	OwnerVersion string
	OwnerKind    string
	OwnerName    string
	OwnerUid     string
}

// This struct is used to store the record_time, owner_uid, own_uid, labels node_name
// It is used to insert data into the database
// Used by pod-controller.go
type PodMisc struct {
	RecordTime time.Time
	OwnerName  string
	OwnUid     string
	Labels     string
	NodeName   string
	AppLabel   string
	Shard      int
}

// record_time, own_version, own_kind, own_uid, owner_version, owner_kind, owner_name, owner_uid, labels
// Used by app-controller.go
type AppOwnerReferences struct {
	RecordTime   time.Time
	OwnVersion   string
	OwnKind      string
	OwnUid       string
	OwnerVersion string
	OwnerKind    string
	OwnerName    string
	OwnerUid     string
	Labels       string
}

// ServiceMisc is a struct that contains the node miscellaneous information
// It is used to insert data into the database
// Used by service-controller.go
type ServiceMisc struct {
	RecordTime time.Time
	UID        string
	AppLabel   string
	Labels     string
	Selector   string
}
