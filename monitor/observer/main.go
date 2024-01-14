package main

import (
	"flag"
	"os"
	"time"

	"kustercost/monitor/pkg/signals"
	"kustercost/monitor/pkg/version"

	_ "github.com/lib/pq"

	"github.com/go-logr/logr"
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

	ctx := signals.SetupSignalHandler()
	logger := klog.FromContext(ctx)
	logger.Info("Klustercost [Observer]", "v", version.Version)

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

	//Initialize the connection to the postgresql database
	postgreSql := &Postgresql{}

	//Add the password for the local db
	//We need to graciously handle the error when the DB is not available
	err = postgreSql.Initialize("postgres", "", "klustercost")
	if err != nil {
		logger.Error(err, "Error initializing the connection to the database")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, time.Second*30)

	controller := NewController(ctx, metricsClientset, kubeClient, kubeInformerFactory.Core().V1().Pods(), postgreSql)

	kubeInformerFactory.Start(ctx.Done())

	if err = controller.Run(ctx, 2); err != nil {
		logger.Error(err, "Error running controller")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

}
