package controller

import (
	"context"
	"fmt"
	transform "klustercost/monitor/controllers/templates"
	"klustercost/monitor/pkg/env"
	"klustercost/monitor/pkg/model"
	"klustercost/monitor/pkg/persistence"
	"klustercost/monitor/pkg/signals"

	"time"

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
	transform     *transform.Transform
}

func NewPodController(
	kubeclientset kubernetes.Interface,
	informer informers.SharedInformerFactory) *PodController {

	podInformer := informer.Core().V1().Pods()

	controller := &PodController{
		kubeclientset: kubeclientset,
		podsLister:    podInformer.Lister(),
		podsSynced:    podInformer.Informer().HasSynced,
		podqueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Pods"),
		transform:     transform.NewTransform(signals.Ctx, "./transform/pod/")}

	_, err := podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueuePod,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueuePod(new)
		},
	})
	if err != nil {
		signals.Logger.Error(err, "Klustercost:  unable to fetch pods")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	return controller
}

func (c *PodController) enqueuePod(obj interface{}) {
	pod := obj.(*v1.Pod)
	c.podqueue.Add(pod.ObjectMeta.Namespace + "/" + pod.ObjectMeta.Name)
}

func (c *PodController) Run(workers int) error {

	defer runtime.HandleCrash()

	signals.Logger.Info("Klustercost: Starting observer threads")

	// Wait for the caches to be synced before starting workers
	signals.Logger.Info("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(signals.Ctx.Done(), c.podsSynced); !ok {
		return fmt.Errorf("Failed to wait for caches to sync")
	}

	signals.Logger.Info("Starting workers for pods", "count", workers)
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(signals.Ctx, c.runWorker, time.Second)
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
			runtime.HandleError(fmt.Errorf("Expected string in workqueue but got %#v", obj))
			return nil
		}

		namespace, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			runtime.HandleError(fmt.Errorf("Invalid resource key: %s", key))
			return nil
		}

		pod, err := c.initPodCollector(namespace, name)
		if err != nil {
			signals.Logger.Error(err, "Unable to init pod collector ")
			return nil
		}

		if pod.Status.Phase == v1.PodRunning {
			transformedPodJson, err := c.transform.Transform(ctx, pod)
			if err != nil {
				c.podqueue.AddRateLimited(obj)
				runtime.HandleError(fmt.Errorf("Cannot transform pod JSON for key %s:", key))
				return nil
			}
			signals.Logger.Info("About to register", "pod data", string(transformedPodJson))

			err = persistence.GetPersistInterface().InsertPodJson(string(transformedPodJson))

			if err != nil {
				c.podqueue.AddRateLimited(obj)
				runtime.HandleError(fmt.Errorf("Cannot insert pod JSON: %s", string(transformedPodJson)))
				return nil
			}
		}

		c.podqueue.Forget(obj)

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
		signals.Logger.Error(err, "Error getting pod lister ")
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
