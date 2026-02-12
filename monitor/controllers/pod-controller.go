package controller

import (
	"context"
	"fmt"
	"klustercost/monitor/pkg/env"
	"klustercost/monitor/pkg/model"
	"klustercost/monitor/pkg/persistence"
	"strconv"
	"time"

	prometheusApi "github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	promModel "github.com/prometheus/common/model"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

var e = env.NewConfiguration()

type PodController struct {
	kubeclientset kubernetes.Interface
	podsLister    corelisters.PodLister
	podsSynced    cache.InformerSynced
	podqueue      workqueue.RateLimitingInterface
	prometheusapi prometheusv1.API
	logger        klog.Logger
}

func NewController(
	ctx context.Context,
	kubeclientset kubernetes.Interface,
	prometheusclient prometheusApi.Client,
	informer informers.SharedInformerFactory) *PodController {

	podInformer := informer.Core().V1().Pods()

	controller := &PodController{
		kubeclientset: kubeclientset,
		podsLister:    podInformer.Lister(),
		podsSynced:    podInformer.Informer().HasSynced,
		podqueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Pods"),
		prometheusapi: prometheusv1.NewAPI(prometheusclient),
		logger:        klog.FromContext(ctx)}

	_, err := podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueuePod,
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

		namespace, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
			return nil
		}

		//Returns the pod objects
		pod, err := c.initPodCollector(namespace, name)
		if err != nil {
			c.logger.Error(err, "Unable to init pod collector ")
			return nil
		}

		if pod.Status.Phase == v1.PodRunning {
			appLabels := c.getAppLabels(pod)
			//Returns the memory and CPU usage of the pod
			podUsage, err := c.getPromData(ctx, namespace, name, strconv.Itoa(e.ResyncTime))
			if err != nil {
				return err
			}

			err = persistence.GetPersistInterface().InsertPod(name, namespace, pod.Spec.NodeName, podUsage, appLabels)

			if err != nil {
				c.podqueue.AddRateLimited(obj)
				runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
				return nil
			}
			c.podqueue.Forget(obj)
		}

		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// Returns the friendly name of the controller
func (c *PodController) FriendlyName() string {
	return "PodController"
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

// getAppLabels extracts the Kubernetes recommended app labels from a pod.
// These are passed to the klustercost.register_pod_data stored procedure.
func (c *PodController) getAppLabels(pod *v1.Pod) *model.PodAppLabels {
	labels := pod.ObjectMeta.Labels
	return &model.PodAppLabels{
		Name:      labels["app.kubernetes.io/name"],
		Instance:  labels["app.kubernetes.io/instance"],
		Version:   labels["app.kubernetes.io/version"],
		Component: labels["app.kubernetes.io/component"],
		PartOf:    labels["app.kubernetes.io/part-of"],
		ManagedBy: labels["app.kubernetes.io/managed-by"],
	}
}

func (c *PodController) getPromData(ctx context.Context, namespace string, service string, timeRange string) (*model.PodConsumption, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := &model.PodConsumption{}

	mem_result_ws, _, err := c.prometheusapi.Query(ctx, c.returnMemory(namespace, "container_memory_working_set_bytes", service, timeRange), time.Now(), prometheusv1.WithTimeout(5*time.Second))
	if err != nil {
		return nil, fmt.Errorf("error querying prometheus client %#v", err)
	}
	vector_mem_result_ws := mem_result_ws.(promModel.Vector)

	if vector_mem_result_ws.Len() == 0 {
		return nil, fmt.Errorf("dont have memory working_set data for %v in %v", service, namespace)
	}

	mem_result_rss, _, err := c.prometheusapi.Query(ctx, c.returnMemory(namespace, "container_memory_rss", service, timeRange), time.Now(), prometheusv1.WithTimeout(5*time.Second))
	if err != nil {
		return nil, fmt.Errorf("error querying prometheus client %#v", err)
	}
	vector_mem_result_rss := mem_result_rss.(promModel.Vector)

	if vector_mem_result_rss.Len() == 0 {
		return nil, fmt.Errorf("dont have memory rss data for %v in %v", service, namespace)
	}

	cpu_result, _, err := c.prometheusapi.Query(ctx, c.returnCPU(namespace, service, timeRange), time.Now(), prometheusv1.WithTimeout(5*time.Second))
	if err != nil {
		return nil, fmt.Errorf("error querying prometheus client %#v", err)
	}
	vector_cpu_result := cpu_result.(promModel.Vector)

	if vector_cpu_result.Len() == 0 {
		return nil, fmt.Errorf("dont have CPU data for %v in %v", service, namespace)
	}

	if vector_mem_result_rss[0].Value < vector_mem_result_ws[0].Value {
		result.Memory = vector_mem_result_ws[0]
	} else {
		result.Memory = vector_mem_result_rss[0]
	}

	result.CPU = vector_cpu_result[0]

	return result, nil
}

func (c *PodController) returnMemory(namespace string, metric string, pod string, timeRange string) string {
	return "max(avg_over_time(" + metric + "{namespace=\"" + namespace + "\", pod=~\"" + pod + ".*\", container_name!=\"POD\"}[" + timeRange + "s]))/1024/1024"
}

func (c *PodController) returnCPU(namespace string, pod string, timeRange string) string {
	return "delta(container_cpu_usage_seconds_total{namespace=\"" + namespace + "\", pod=~\"" + pod + ".*\", container_name!=\"POD\"}[" + timeRange + "s] )/" + timeRange + ""
}
