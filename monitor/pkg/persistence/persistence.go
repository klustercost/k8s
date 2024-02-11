package persistence

import "klustercost/monitor/pkg/model"

type Persistence interface {
	InsertNode(string, *model.NodeMisc) error
}

// PostgresDB is a struct that defines the fact that the medium for storage is a Postgres database
type PostgresDB struct {
	Persistence
}

// InsertNode is a function that inserts the details of a node into the database
func InsertNode(p Persistence, nodeName string, nodeMisc *model.NodeMisc) error {
	return p.InsertNode(nodeName, nodeMisc)
}
