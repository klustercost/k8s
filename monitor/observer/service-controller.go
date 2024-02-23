package main

import (
	"context"
	"fmt"
	"klustercost/monitor/pkg/model"
	"klustercost/monitor/pkg/persistence"
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

type ServiceController struct {
	kubeclientset kubernetes.Interface
	serviceLister corelisters.ServiceLister
	serviceSynced cache.InformerSynced
	servicequeue  workqueue.RateLimitingInterface
	logger        klog.Logger
}

func NewServiceController(
	ctx context.Context,
	kubeclientset kubernetes.Interface,
	informer informers.SharedInformerFactory) *ServiceController {

	serviceInformer := informer.Core().V1().Services()

	sc := &ServiceController{
		kubeclientset: kubeclientset,
		serviceLister: serviceInformer.Lister(),
		serviceSynced: serviceInformer.Informer().HasSynced,
		servicequeue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Services"),
		logger:        klog.FromContext(ctx)}

	_, err := serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		//AddFunc: nc.enqueueService,
		UpdateFunc: func(old, new interface{}) {
			sc.enqueueService(new)
		},
	})
	if err != nil {
		sc.logger.Error(err, "Klustercost:  unable to fetch services")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	return sc
}

func (sc *ServiceController) enqueueService(obj interface{}) {
	service := obj.(*v1.Service)
	sc.servicequeue.Add(service.ObjectMeta.Namespace + "/" + service.ObjectMeta.Name)
}

func (sc *ServiceController) Run(ctx context.Context, workers int) error {

	defer runtime.HandleCrash()

	sc.logger.Info("Klustercost: Starting service observer threads")

	// Wait for the caches to be synced before starting workers
	sc.logger.Info("Waiting for service informer caches to sync")

	if ok := cache.WaitForCacheSync(ctx.Done(), sc.serviceSynced); !ok {
		return fmt.Errorf("failed to wait for service caches to sync")
	}

	sc.logger.Info("Starting workers for services", "count", workers)
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, sc.runWorker, time.Second)
	}

	sc.logger.Info("Done")

	return nil
}

// runWorker runs a worker to process items from the workqueue
func (sc *ServiceController) runWorker(ctx context.Context) {
	for sc.processNextWorkItem(ctx) {
	}
}

// processNextWorkItem processes items from the workqueue
func (sc *ServiceController) processNextWorkItem(ctx context.Context) bool {
	obj, shutdown := sc.servicequeue.Get()

	if shutdown {
		return false
	}
	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		defer sc.servicequeue.Done(obj)
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
			sc.servicequeue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := sc.syncHandler(ctx, key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			sc.servicequeue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		sc.servicequeue.Forget(obj)
		sc.logger.Info("Successfully synced", "resourceName", key)

		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

func (sc *ServiceController) syncHandler(ctx context.Context, key string) error {

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	x := sc.getServiceMiscellaneous(namespace, name)

	err = persistence.GetPersistInterface().InsertService(name, namespace, x)

	if err != nil {
		sc.logger.Error(err, "Error inserting service details into the database")
	}
	return nil
}

func (sc *ServiceController) getServiceMiscellaneous(namespace, name string) *model.ServiceMisc {

	service := sc.serviceLister.Services(namespace)

	serviceMisc := &model.ServiceMisc{}

	x, _ := service.Get(name)

	serviceMisc.UID = string(x.UID)
	serviceMisc.Labels = MapToString(x.Labels)
	serviceMisc.Selector = MapToString(x.Spec.Selector)

	return serviceMisc
}
