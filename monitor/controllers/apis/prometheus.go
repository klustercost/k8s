package controller

import (
	env "klustercost/monitor/pkg/env"

	prometheusApi "github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"k8s.io/klog/v2"
)

var prometheusapi prometheusv1.API

func init() {
	prometheusclient, err := prometheusApi.NewClient(prometheusApi.Config{Address: env.EnvironmentVariables.PrometheusServer})
	if err != nil {
		//TODO: ctx and logger should be passed in here instead of using klog directly
		klog.Error(err, "Unable to create Prometheus client")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	prometheusapi = prometheusv1.NewAPI(prometheusclient)
}

func GetPrometheusAPI() prometheusv1.API {
	return prometheusapi
}
