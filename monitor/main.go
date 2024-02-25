package main

import (
	"flag"
	"os"
	"time"

	"klustercost/monitor/pkg/env"
	"klustercost/monitor/pkg/observer"
	"klustercost/monitor/pkg/persistence"
	"klustercost/monitor/pkg/signals"
	"klustercost/monitor/pkg/version"

	controller "klustercost/monitor/observer"

	"github.com/go-logr/logr"
	_ "github.com/lib/pq"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

var controllers []observer.Controller

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

	defer persistence.Close()

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

	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, time.Second*time.Duration(env.ResyncTime))

	// Create the controllers
	// All new controllers to be initialized from here
	controllers = append(controllers,
		controller.NewController(ctx, metricsClientset, kubeClient, kubeInformerFactory),
		controller.NewNodeController(ctx, kubeClient, kubeInformerFactory),
		controller.NewAppController(ctx, kubeClient, kubeInformerFactory),
		controller.NewServiceController(ctx, kubeClient, kubeInformerFactory),
	)

	kubeInformerFactory.Start(ctx.Done())

	for _, controller := range controllers {
		if err = controller.Run(ctx, env.ControllerWorkers); err != nil {
			logger.Error(err, "Error running ", controller.FriendlyName())
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
	}

	<-ctx.Done()
}
