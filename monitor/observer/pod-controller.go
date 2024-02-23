package main

import (
	"context"
	"fmt"
	"klustercost/monitor/pkg/model"
	"klustercost/monitor/pkg/persistence"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

type PodController struct {
	kubeclientset kubernetes.Interface
	podsLister    corelisters.PodLister
	podsSynced    cache.InformerSynced
	podqueue      workqueue.RateLimitingInterface
	metrics       metricsv.Clientset
	logger        klog.Logger
}

func NewController(
	ctx context.Context,
	metricsClientset *metricsv.Clientset,
	kubeclientset kubernetes.Interface,
	informer informers.SharedInformerFactory) *PodController {

	podInformer := informer.Core().V1().Pods()

	controller := &PodController{
		kubeclientset: kubeclientset,
		podsLister:    podInformer.Lister(),
		podsSynced:    podInformer.Informer().HasSynced,
		podqueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Pods"),
		metrics:       *metricsClientset,
		logger:        klog.FromContext(ctx)}

	_, err := podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		//AddFunc: controller.enqueuePod,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueuePod(new)
		},
	})
	if err != nil {
		controller.logger.Error(err, "Klustercost:  unable to fetch pods")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	return controller
}

func (c *PodController) enqueuePod(obj interface{}) {
	pod := obj.(*v1.Pod)
	c.podqueue.Add(pod.ObjectMeta.Namespace + "/" + pod.ObjectMeta.Name)
}

func (c *PodController) Run(ctx context.Context, workers int) error {

	defer runtime.HandleCrash()

	c.logger.Info("Klustercost: Starting observer threads")

	// Wait for the caches to be synced before starting workers
	c.logger.Info("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(ctx.Done(), c.podsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	c.logger.Info("Starting workers for pods", "count", workers)
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	return nil
}

func (c *PodController) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

func (c *PodController) processNextWorkItem(ctx context.Context) bool {
	obj, shutdown := c.podqueue.Get()

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

		if err := c.syncHandler(ctx, key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.podqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.podqueue.Forget(obj)
		c.logger.Info("Successfully synced", "resourceName", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

func (c *PodController) syncHandler(ctx context.Context, key string) error {

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	//Returns the pod and the pod metrics objects
	pod, err := c.initPodCollector(namespace, name)
	if err != nil {
		c.logger.Error(err, "Unable to init pod collector ")
		return nil
	}

	if pod.Status.Phase == v1.PodRunning {
		ownerRef := c.returnOwnerReferences(pod)
		podMisc := c.getPodMiscellaneous(pod)
		if err != nil {
			c.logger.Error(err, "Unable to get pod miscellaneous")
			return nil
		}

		//Returns the memory and CPU usage of the pod
		podUsage := c.getPodConsumption(namespace, name)

		err = persistence.GetPersistInterface().InsertPod(name, namespace, podMisc, ownerRef, podUsage)

		if err != nil {
			c.logger.Error(err, "Error inserting pod data into the database")
		}
	}
	return nil
}

// This function retrieves the pod object from the informer cache.
func (c *PodController) initPodCollector(namespace, name string) (*v1.Pod, error) {
	pod, err := c.podsLister.Pods(namespace).Get(name)
	if err != nil {
		c.logger.Error(err, "Error getting pod lister ")
	}

	return pod, err
}

// Returns owner_version, owner_kind, owner_name, owner_uid of a *v1.Pod
func (c *PodController) returnOwnerReferences(pod *v1.Pod) *model.OwnerReferences {

	ownerRef := &model.OwnerReferences{}

	for _, v := range pod.ObjectMeta.OwnerReferences {
		if v.Name != "" {
			ownerRef.OwnerVersion = v.APIVersion
			ownerRef.OwnerKind = v.Kind
			ownerRef.OwnerName = v.Name
			ownerRef.OwnerUid = string(v.UID)
		}
	}
	return ownerRef
}

// This function retrieves the record_time, owner_uid, own_uid, labels node_name
// It queries the API server
func (c *PodController) getPodMiscellaneous(pod *v1.Pod) *model.PodMisc {
	misc := &model.PodMisc{}
	//record_time is the time when the function is run
	//It is used as a timestamp for the time when data was insterted in the database
	misc.RecordTime = time.Now()
	owner := pod.ObjectMeta.OwnerReferences

	for _, v := range owner {
		if v.Name != "" {
			misc.OwnerName = string(v.UID)
		}
	}
	misc.OwnUid = string(pod.ObjectMeta.UID)
	misc.NodeName = pod.Spec.NodeName
	misc.Labels = MapToString(pod.ObjectMeta.Labels)
	misc.AppLabel = findAppLabel(pod.ObjectMeta.Labels)

	return misc

}

// This function retrieves the pod metrics object from the metrics server.
// And then returns the memory and CPU usage of a pod
// It queries the metrics server
func (c *PodController) getPodConsumption(namespace, name string) *model.PodConsumption {

	usage := &model.PodConsumption{}

	pod, err := c.metrics.MetricsV1beta1().PodMetricses(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		c.logger.Error(err, "Error getting pod metrics ")
	}

	// Calculate total memory usage for the entire pod
	for i := 0; i < len(pod.Containers); i++ {
		usage.Memory += pod.Containers[i].Usage.Memory().Value()
		usage.CPU += pod.Containers[i].Usage.Cpu().MilliValue()
	}

	return usage
}

// Helper function to convert values of a map[string]string to a csv string.
// Map key and value are returned separated by comma key=value,key=value.
func MapToString(labels map[string]string) string {
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

func findAppLabel(m map[string]string) string {
	for key, value := range m {
		if strings.HasPrefix(key, "app") {
			return fmt.Sprintf("%s=%s", key, value)
		}
	}
	return ""
}
