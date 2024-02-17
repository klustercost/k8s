package persistence

import "klustercost/monitor/pkg/model"

type Persistence interface {
	InsertNode(string, *model.NodeMisc) error
}

// InsertNode is a function that inserts the details of a node into the database
func InsertNode(p Persistence, nodeName string, nodeMisc *model.NodeMisc) error {
	return p.InsertNode(nodeName, nodeMisc)
}
