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

var (
	resinc_time        int
	controller_workers int
	pg_db_user         string
	pg_db_pass         string
	pg_db_name         string
)

func init() {
	flag.IntVar(&resinc_time, "resinc_time", 60, "Resinc time for the shared informer factory")
	flag.IntVar(&controller_workers, "controller_workers", 2, "Number of workers for the controller")
	flag.StringVar(&pg_db_user, "pg_db_user", "postgres", "Username for the postgresql server login")
	flag.StringVar(&pg_db_pass, "pg_db_pass", "admin", "Password for the postgresql server login")
	flag.StringVar(&pg_db_name, "pg_db_name", "klustercost", "Name of the postgresql database")
}

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
	if err != nil {
		logger.Error(err, "Error connecting to the metrics server")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	//Initialize the connection to the postgresql database
	postgreSql, err := NewPostgresql(pg_db_user, pg_db_pass, pg_db_name)
	if err != nil {
		logger.Error(err, "Error connecting to the postgresql database")
	}

	defer postgreSql.Close()

	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, time.Second*time.Duration(resinc_time))
	controller := NewController(ctx, metricsClientset, kubeClient, kubeInformerFactory.Core().V1().Pods(), postgreSql)

	kubeInformerFactory.Start(ctx.Done())

	if err = controller.Run(ctx, controller_workers); err != nil {
		logger.Error(err, "Error running controller")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

}
