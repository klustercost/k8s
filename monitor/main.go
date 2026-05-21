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

	controller "klustercost/monitor/controllers"

	_ "github.com/lib/pq"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

var controllers []observer.Controller

func get_config() (*rest.Config, error) {
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
	signals.Logger.Info("Klustercost [Observer]", "v", version.Version)

	defer persistence.Close()

	config, err := get_config()
	if err != nil {
		signals.Logger.Error(err, "Cannot get a valid k8s context")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		signals.Logger.Error(err, "Error building kubernetes clientset")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, time.Second*time.Duration(env.EnvironmentVariables.ResyncTime))

	// Create the controllers
	// All new controllers to be initialized from here
	controllers = append(controllers,
		controller.NewPodController(kubeClient, kubeInformerFactory),
		controller.NewNodeController(kubeClient, kubeInformerFactory),
	)

	kubeInformerFactory.Start(signals.Ctx.Done())

	for _, controller := range controllers {
		if err = controller.Run(env.EnvironmentVariables.ControllerWorkers); err != nil {
			signals.Logger.Error(err, "Error running ", controller.FriendlyName())
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
	}

	<-signals.Ctx.Done()
}
