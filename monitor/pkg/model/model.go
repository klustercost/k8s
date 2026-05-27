package model

type DataExchange map[string]interface{}

// NodeMisc is a struct that contains the node miscellaneous information
// It is used to insert data into the database
// Used by node-controller.go
type NodeMisc struct {
	Memory       float64
	CPU          float64
	UID          string
	Labels       string
	InstanceType string
	Region       string
	Zone         string
	OS           string
}
