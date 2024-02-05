package main

import (
	"flag"
	"os"
	"time"

	"klustercost/monitor/pkg/env"
	"klustercost/monitor/pkg/postgres"
	"klustercost/monitor/pkg/signals"
	"klustercost/monitor/pkg/version"

	"github.com/go-logr/logr"
	_ "github.com/lib/pq"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

func get_config(loger logr.Logger) (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if nil == err {
		return config, err
	}

	dirname, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	kubeconfig := flag.String("kubeconfig", dirname+"\\.kube\\config", "kubeconfig file")
	flag.Parse()
	return clientcmd.BuildConfigFromFlags("", *kubeconfig)
}

func main() {
	klog.InitFlags(nil)
	env := env.NewConfiguration()

	ctx := signals.SetupSignalHandler()
	logger := klog.FromContext(ctx)
	logger.Info("Klustercost [Observer]", "v", version.Version)

	defer postgres.Close()

	config, err := get_config(logger)
	if err != nil {
		logger.Error(err, "Cannot get a valid k8s context")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Error building kubernetes clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	//Get resources from the metrics server
	metricsClientset, err := metricsv.NewForConfig(config)
	if err != nil {
		logger.Error(err, "Error connecting to the metrics server")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, time.Second*time.Duration(env.ResincTime))
	podcontroller := NewController(ctx, metricsClientset, kubeClient, kubeInformerFactory)
	nodecontroller := NewNodeController(ctx, metricsClientset, kubeClient, kubeInformerFactory)
	appcontroller := NewAppController(ctx, metricsClientset, kubeClient, kubeInformerFactory)

	kubeInformerFactory.Start(ctx.Done())

	if err = podcontroller.Run(ctx, env.ControllerWorkers); err != nil {
		logger.Error(err, "Error running controller")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	if err = nodecontroller.Run(ctx, env.ControllerWorkers); err != nil {
		logger.Error(err, "Error running node controller")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	if err = appcontroller.Run(ctx, env.ControllerWorkers); err != nil {
		logger.Error(err, "Error running node controller")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	<-ctx.Done()
}
