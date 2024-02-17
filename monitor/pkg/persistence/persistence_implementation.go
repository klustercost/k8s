package persistence

import (
	"klustercost/monitor/pkg/postgres"
	//TODO: enumerate here any other supported persistence implementation
	"k8s.io/klog/v2"

)

const (
	POSTGRESS = iota
	PROMETHEUS
)

// TODO: read from environment variable and call the correct persistence package: postgres, prometheus, so on
var persistence_type = POSTGRESS

func GetPersistInterface() Persistence {
	switch persistence_type {
	case POSTGRESS:
		return postgres.GetPersistInterface().(Persistence)
	default:
		//nc.logger.Error(err, "Klustercost:  persistence not supported (PROMETHEUS)")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
}

func Close() {
	switch persistence_type {
	case POSTGRESS:
		postgres.ClosePersistInterface()
		break
	default
		break
	}

}
