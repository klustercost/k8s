package persistence

import "klustercost/monitor/pkg/model"

type Persistence interface {
	InsertNode(string, *model.NodeMisc) error
}

// PostgresDB is a struct that defines the fact that the medium for storage is a Postgres database
type PostgresDB struct{}

func InsertNode(p Persistence, nodeName string, nodeMisc *model.NodeMisc) error {
	return p.InsertNode(nodeName, nodeMisc)
}
