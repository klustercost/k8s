package main

import (
	"context"
	"fmt"
	"klustercost/monitor/pkg/postgres"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	applister "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

type AppController struct {
	kubeclientset kubernetes.Interface
	dsLister      applister.DaemonSetLister
	dsSynced      cache.InformerSynced
	deployLister  applister.DeploymentLister
	deploySynced  cache.InformerSynced
	sSetLister    applister.StatefulSetLister
	sSetSynced    cache.InformerSynced
	rSetLister    applister.ReplicaSetLister
	rSetSynced    cache.InformerSynced
	appqueue      workqueue.RateLimitingInterface
	metrics       metricsv.Clientset
	postgresql    *postgres.Postgresql
}

func NewAppController(
	ctx context.Context,
	metricsClientset *metricsv.Clientset,
	kubeclientset kubernetes.Interface,
	informer informers.SharedInformerFactory,
	postgres *postgres.Postgresql) *AppController {

	//Init informers:
	dsInformer := informer.Apps().V1().DaemonSets()
	deployInformer := informer.Apps().V1().Deployments()
	sSetInformer := informer.Apps().V1().StatefulSets()
	rSetInformer := informer.Apps().V1().ReplicaSets()

	logger := klog.FromContext(ctx)

	ac := &AppController{
		kubeclientset: kubeclientset,
		dsLister:      dsInformer.Lister(),
		dsSynced:      dsInformer.Informer().HasSynced,
		deployLister:  deployInformer.Lister(),
		deploySynced:  deployInformer.Informer().HasSynced,
		sSetLister:    sSetInformer.Lister(),
		sSetSynced:    sSetInformer.Informer().HasSynced,
		rSetLister:    rSetInformer.Lister(),
		rSetSynced:    rSetInformer.Informer().HasSynced,
		appqueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Apps"),
		metrics:       *metricsClientset,
		postgresql:    postgres}

	_, err := dsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: ac.enqueueApp,
		UpdateFunc: func(old, new interface{}) {
			ac.enqueueApp(new)
		},
	})
	_, err = deployInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: ac.enqueueApp,
		UpdateFunc: func(old, new interface{}) {
			ac.enqueueApp(new)
		},
	})
	_, err = sSetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: ac.enqueueApp,
		UpdateFunc: func(old, new interface{}) {
			ac.enqueueApp(new)
		},
	})
	_, err = rSetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: ac.enqueueApp,
		UpdateFunc: func(old, new interface{}) {
			ac.enqueueApp(new)
		},
	})
	if err != nil {
		logger.Error(err, "Klustercost:  unable to fetch apps/v1")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	return ac
}

func (ac *AppController) enqueueApp(obj interface{}) {

	switch v := obj.(type) {
	case *appsv1.DaemonSet:
		ac.appqueue.Add(v.ObjectMeta.Namespace + "/" + v.ObjectMeta.Name)
	case *appsv1.Deployment:
		ac.appqueue.Add(v.ObjectMeta.Namespace + "/" + v.ObjectMeta.Name)
	case *appsv1.StatefulSet:
		ac.appqueue.Add(v.ObjectMeta.Namespace + "/" + v.ObjectMeta.Name)
	case *appsv1.ReplicaSet:
		ac.appqueue.Add(v.ObjectMeta.Namespace + "/" + v.ObjectMeta.Name)
	}

}

func (ac *AppController) Run(ctx context.Context, workers int) error {

	defer runtime.HandleCrash()

	logger := klog.FromContext(ctx)
	logger.Info("Klustercost: Starting apps observer threads")

	// Wait for the caches to be synced before starting workers
	logger.Info("Waiting for apps informer caches to sync")

	if ok := cache.WaitForCacheSync(ctx.Done(), ac.dsSynced); !ok {
		return fmt.Errorf("failed to wait for DaemonSet caches to sync")
	}

	if ok := cache.WaitForCacheSync(ctx.Done(), ac.deploySynced); !ok {
		return fmt.Errorf("failed to wait for Deployment caches to sync")
	}

	if ok := cache.WaitForCacheSync(ctx.Done(), ac.sSetSynced); !ok {
		return fmt.Errorf("failed to wait for StatefulSet caches to sync")
	}

	logger.Info("Starting workers for apps", "count", workers)
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, ac.runWorker, time.Second)
	}

	<-ctx.Done()
	logger.Info("Done")

	return nil
}

func (ac *AppController) runWorker(ctx context.Context) {
	for ac.processNextWorkItem(ctx) {
	}
}

func (ac *AppController) processNextWorkItem(ctx context.Context) bool {
	obj, shutdown := ac.appqueue.Get()
	//logger := klog.FromContext(ctx)

	if shutdown {
		return false
	}
	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		defer ac.appqueue.Done(obj)
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
			ac.appqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		namespace, name, err := cache.SplitMetaNamespaceKey(key)
		record_time, own_version, own_kind, own_uid, owner_version, owner_kind, owner_name, owner_uid, labels := ac.returnOwnerReferences(namespace, name)

		err = ac.postgresql.InsertOwners(name, namespace, record_time, own_version, own_kind, own_uid, owner_version, owner_kind, owner_name, owner_uid, labels)
		if err != nil {
			klog.Error(err)
		}

		if err != nil {
			klog.Error(err)
		}

		if err != nil {
			ac.appqueue.Forget(obj)
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

// Returns owner_version, owner_kind, owner_name, owner_uid
func ownerReferences(owner []metav1.OwnerReference) (string, string, string, string) {
	for _, v := range owner {
		if v.Name != "" {
			return v.APIVersion, v.Kind, v.Name, string(v.UID)
		}
	}
	return "", "", "", ""
}

// record_time, own_version, own_kind, own_uid, owner_version, owner_kind, owner_name, owner_uid, labels
func (ac *AppController) returnOwnerReferences(namespace, name string) (time.Time, string, string, string, string, string, string, string, string) {
	//record_time is the time when the function is run
	//It is used as a timestamp for the time when data was insterted in the database
	record_time := time.Now()
	ds, _ := ac.dsLister.DaemonSets(namespace).Get(name)
	deploy, _ := ac.deployLister.Deployments(namespace).Get(name)
	sSet, _ := ac.sSetLister.StatefulSets(namespace).Get(name)
	rSet, _ := ac.rSetLister.ReplicaSets(namespace).Get(name)

	switch {
	case ds != nil:
		owner := ds.ObjectMeta.OwnerReferences
		owner_version, owner_kind, owner_name, owner_uid := ownerReferences(owner)
		return record_time, "apps/v1", "DaemonSet", string(ds.UID), owner_version, owner_kind, owner_name, owner_uid, mapToString(ds.Labels)
	case deploy != nil:
		owner := deploy.ObjectMeta.OwnerReferences
		owner_version, owner_kind, owner_name, owner_uid := ownerReferences(owner)
		return record_time, "apps/v1", "Deployment", string(deploy.UID), owner_version, owner_kind, owner_name, owner_uid, mapToString(deploy.Labels)
	case sSet != nil:
		owner := sSet.ObjectMeta.OwnerReferences
		owner_version, owner_kind, owner_name, owner_uid := ownerReferences(owner)
		return record_time, "apps/v1", "StatefulSet", string(sSet.UID), owner_version, owner_kind, owner_name, owner_uid, mapToString(sSet.Labels)
	case rSet != nil:
		owner := rSet.ObjectMeta.OwnerReferences
		owner_version, owner_kind, owner_name, owner_uid := ownerReferences(owner)
		return record_time, "apps/v1", "ReplicaSet", string(rSet.UID), owner_version, owner_kind, owner_name, owner_uid, mapToString(rSet.Labels)
		/*
			case *batchv1.Job:
				return v.ObjectMeta.OwnerReferences*/
	default:
		return record_time, "", "", "", "", "", "", "", ""
	}
}
