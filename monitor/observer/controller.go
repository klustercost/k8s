package main

import (
	"context"
	"fmt"
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
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

type Controller struct {
	kubeclientset kubernetes.Interface
	podsLister    corelisters.PodLister
	podsSynced    cache.InformerSynced
	podqueue      workqueue.RateLimitingInterface
	metrics       metricsv.Clientset
	postgresql    *Postgresql
}

func NewController(
	ctx context.Context,
	metricsClientset *metricsv.Clientset,
	kubeclientset kubernetes.Interface,
	podInformer coreinformers.PodInformer,
	postgres *Postgresql) *Controller {

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

	<-ctx.Done()
	logger.Info("Done")

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
		pod, pod_metrics, err := c.initCollectors(namespace, name)
		if err != nil {
			klog.Error("Unable to init collectors ", err)
			return nil
		}

		//fmt.Println(namespace, name, podMem/1024/1024, podCpu)
		record_time, owner, node_name := c.getPodMiscellaneous(pod)
		if err != nil {
			klog.Error("Get pod miscellaneous:", err)
			return nil
		}

		//Returns the memory and CPU usage of the pod
		podMem, podCpu := c.getPodConsumption(pod_metrics)
		//fmt.Println(namespace, name, record_time, owner, node_name)

		err = c.postgresql.InsertPod(name, namespace, record_time, podMem, podCpu, owner, node_name)
		if err != nil {
			klog.Error(err)
		}

		//TODO:
		//case 1: starts
		//1. get resources from metrics server DONE
		//2. get time start TBD
		//3. get metadata (name, namespace, owener, labels...)
		//4. insert in db DONE
		//case 2: stops
		//1. mark in db the time stop TBD
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

// This function retrieves the memory and CPU usage of a pod
// It queries the metrics server
func (c *Controller) getPodConsumption(pod *v1beta1.PodMetrics) (int64, int64) {
	// Calculate total memory usage for the entire pod
	var totalMemUsageBytes int64
	var totalCPUUsageMili int64

	for i := 0; i < len(pod.Containers); i++ {
		totalMemUsageBytes += pod.Containers[i].Usage.Memory().Value()
		totalCPUUsageMili += pod.Containers[i].Usage.Cpu().MilliValue()
	}

	return totalMemUsageBytes, totalCPUUsageMili
}

// This function retrieves the record_time, owner, node_name
// It queries the API server
func (c *Controller) getPodMiscellaneous(pod *v1.Pod) (time.Time, string, string) {
	record_time := time.Now()
	owner := pod.ObjectMeta.OwnerReferences
	var owner_name string

	for _, v := range owner {
		if v.Name != "" {
			owner_name = string(v.UID)

		} else {
			owner_name = "No owner"
		}

	}
	node_name := pod.Spec.NodeName

	return record_time, owner_name, node_name

}

func (c *Controller) initCollectors(namespace, name string) (*v1.Pod, *v1beta1.PodMetrics, error) {
	pod, err := c.podsLister.Pods(namespace).Get(name)
	if err != nil {
		klog.Error("Error getting pod lister ", err)
		return nil, nil, err
	}

	pod_metrics, err := c.metrics.MetricsV1beta1().PodMetricses(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		klog.Error("Error getting pod metrics ", err)
	}

	return pod, pod_metrics, err
}
