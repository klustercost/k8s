package main

import (
	"context"
	"fmt"
	"klustercost/monitor/pkg/postgres"
	"os"
	"strconv"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

type Controller struct {
	kubeclientset kubernetes.Interface
	podsLister    corelisters.PodLister
	podsSynced    cache.InformerSynced
	podqueue      workqueue.RateLimitingInterface
	metrics       metricsv.Clientset
	postgresql    *postgres.Postgresql
}

func NewController(
	ctx context.Context,
	metricsClientset *metricsv.Clientset,
	kubeclientset kubernetes.Interface,
	podInformer coreinformers.PodInformer,
	postgres *postgres.Postgresql) *Controller {

	logger := klog.FromContext(ctx)

	controller := &Controller{
		kubeclientset: kubeclientset,
		podsLister:    podInformer.Lister(),
		podsSynced:    podInformer.Informer().HasSynced,
		podqueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Pods"),
		metrics:       *metricsClientset,
		postgresql:    postgres}

	_, err := podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueuePod,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueuePod(new)
		},
	})
	if err != nil {
		logger.Error(err, "Klustercost:  unable to fetch pods")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	return controller
}

func (c *Controller) enqueuePod(obj interface{}) {
	pod := obj.(*v1.Pod)
	c.podqueue.Add(pod.ObjectMeta.Namespace + "/" + pod.ObjectMeta.Name)
}

func (c *Controller) Run(ctx context.Context, workers int) error {

	defer runtime.HandleCrash()

	logger := klog.FromContext(ctx)
	logger.Info("Klustercost: Starting observer threads")

	// Wait for the caches to be synced before starting workers
	logger.Info("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(ctx.Done(), c.podsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	logger.Info("Starting workers for pods", "count", workers)
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	return nil
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	obj, shutdown := c.podqueue.Get()
	//logger := klog.FromContext(ctx)

	if shutdown {
		return false
	}
	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		defer c.podqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.podqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		namespace, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			klog.Error(err)
		}
		//Returns the pod and the pod metrics objects
		pod, err := c.initPodCollector(namespace, name)
		if err != nil {
			klog.Error("Unable to init pod collector ", err)
			return nil
		}

		if pod.Status.Phase == v1.PodRunning {
			owner_version, owner_kind, owner_name, owner_uid := c.returnOwnerReferences(pod)
			record_time, owner, own_uid, labels, node_name := c.getPodMiscellaneous(pod)
			if err != nil {
				klog.Error("Get pod miscellaneous:", err)
				return nil
			}

			//Returns the memory and CPU usage of the pod
			podMem, podCpu := c.getPodConsumption(namespace, name)
			fmt.Println("INSERTED:", name, namespace, record_time, podMem, podCpu, owner, node_name)

			err = c.postgresql.InsertPod(name, namespace, record_time, podMem, podCpu, owner_version, owner_kind, owner_name, owner_uid, own_uid, labels, node_name)
			if err != nil {
				klog.Error(err)
			}
		}

		if err != nil {
			c.podqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
			return nil
		}

		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

type EnvVars struct {
	resinc_time        int
	controller_workers int
	pg_db_user         string
	pg_db_pass         string
	pg_db_name         string
}

func initEnvVars() *EnvVars {

	resinc_time, err := strconv.Atoi(os.Getenv("RESINC_TIME"))
	if err != nil {
		resinc_time = 60
		klog.Info("RESINC_TIME not set, using default value of 60")
	}

	controller_workers, err := strconv.Atoi(os.Getenv("CONTROLLER_WORKERS"))
	if err != nil {
		controller_workers = 2
		klog.Info("CONTROLLER_WORKERS not set, using default value of 2")
	}

	pg_db_user := os.Getenv("PG_DB_USER")
	if pg_db_user == "" {
		pg_db_user = "postgres"
		klog.Info("PG_DB_USER not set, using default value of postgres")
	}

	pg_db_pass := os.Getenv("PG_DB_PASS")
	if pg_db_pass == "" {
		pg_db_pass = "admin"
		klog.Info("PG_DB_PASS not set, using default value of admin")
	}

	pg_db_name := os.Getenv("PG_DB_NAME")
	if pg_db_name == "" {
		pg_db_name = "klustercost"
		klog.Info("PG_DB_NAME not set, using default value of klustercost")
	}

	e := &EnvVars{
		resinc_time:        resinc_time,
		controller_workers: controller_workers,
		pg_db_user:         pg_db_user,
		pg_db_pass:         pg_db_pass,
		pg_db_name:         pg_db_name,
	}

	return e
}

// This function retrieves the pod object from the informer cache.
func (c *Controller) initPodCollector(namespace, name string) (*v1.Pod, error) {
	pod, err := c.podsLister.Pods(namespace).Get(name)
	if err != nil {
		klog.Error("Error getting pod lister ", err)
	}

	return pod, err
}

// Returns owner_version, owner_kind, owner_name, owner_uid of a *v1.Pod
func (c *Controller) returnOwnerReferences(pod *v1.Pod) (string, string, string, string) {
	owner := pod.ObjectMeta.OwnerReferences
	for _, v := range owner {
		if v.Name != "" {
			return v.APIVersion, v.Kind, v.Name, string(v.UID)
		}
	}
	return "", "", "", ""
}

// This function retrieves the record_time, owner_uid, own_uid, labels node_name
// It queries the API server
func (c *Controller) getPodMiscellaneous(pod *v1.Pod) (time.Time, string, string, string, string) {
	//record_time is the time when the function is run
	//It is used as a timestamp for the time when data was insterted in the database
	record_time := time.Now()
	owner := pod.ObjectMeta.OwnerReferences
	var owner_name string

	for _, v := range owner {
		if v.Name != "" {
			owner_name = string(v.UID)
		}
	}
	own_uid := pod.ObjectMeta.UID
	node_name := pod.Spec.NodeName
	labels := mapToString(pod.ObjectMeta.Labels)

	return record_time, owner_name, string(own_uid), labels, node_name

}

// This function retrieves the pod metrics object from the metrics server.
// And then returns the memory and CPU usage of a pod
// It queries the metrics server
func (c *Controller) getPodConsumption(namespace, name string) (int64, int64) {

	pod, err := c.metrics.MetricsV1beta1().PodMetricses(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		klog.Error("Error getting pod metrics ", err)
	}

	// Calculate total memory usage for the entire pod
	var totalMemUsageBytes int64
	var totalCPUUsageMili int64

	for i := 0; i < len(pod.Containers); i++ {
		totalMemUsageBytes += pod.Containers[i].Usage.Memory().Value()
		totalCPUUsageMili += pod.Containers[i].Usage.Cpu().MilliValue()
	}

	return totalMemUsageBytes, totalCPUUsageMili
}

// Helper function to convert values of a map[string]string to a csv string.
// Map key and value are returned separated by comma key=value,key=value.
func mapToString(labels map[string]string) string {
	var sb strings.Builder

	i := 0
	for key, value := range labels {
		sb.WriteString(key)
		sb.WriteString("=")
		sb.WriteString(value)
		if i < len(labels)-1 {
			sb.WriteString(",")
		}
		i++
	}
	return sb.String()
}
