package main

import (
	"flag"
	"os"
	"sync"
	"time"

	"kustercost/monitor/pkg/signals"
	"kustercost/monitor/pkg/version"

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
	//Sync workload go routines via wait groups
	ch := make(chan int)
	var wg sync.WaitGroup

	klog.InitFlags(nil)
	env := initEnvVars()
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
	if err != nil {
		logger.Error(err, "Error connecting to the metrics server")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	//Initialize the connection to the postgresql database
	postgreSql, err := NewPostgresql(env.pg_db_user, env.pg_db_pass, env.pg_db_name)
	if err != nil {
		logger.Error(err, "Error connecting to the postgresql database")
	}

	defer postgreSql.Close()

	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, time.Second*time.Duration(env.resinc_time))
	controller := NewController(ctx, metricsClientset, kubeClient, kubeInformerFactory.Core().V1().Pods(), postgreSql)
	nodecontroller := NewNodeController(ctx, metricsClientset, kubeClient, kubeInformerFactory.Core().V1().Nodes(), postgreSql)

	kubeInformerFactory.Start(ctx.Done())
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err = controller.Run(ctx, env.controller_workers); err != nil {
			logger.Error(err, "Error running controller")
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err = nodecontroller.RunNode(ctx, env.controller_workers); err != nil {
			logger.Error(err, "Error running node controller")
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
	}()
	wg.Wait()
	close(ch)
}
