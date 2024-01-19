package main

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
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

type NodeController struct {
	kubeclientset kubernetes.Interface
	nodesLister   corelisters.NodeLister
	nodesSynced   cache.InformerSynced
	nodequeue     workqueue.RateLimitingInterface
	metrics       metricsv.Clientset
	postgresql    *Postgresql
}

func NewNodeController(
	ctx context.Context,
	metricsClientset *metricsv.Clientset,
	kubeclientset kubernetes.Interface,
	nodesInformer coreinformers.NodeInformer,
	postgres *Postgresql) *NodeController {

	logger := klog.FromContext(ctx)

	nc := &NodeController{
		kubeclientset: kubeclientset,
		nodesLister:   nodesInformer.Lister(),
		nodesSynced:   nodesInformer.Informer().HasSynced,
		nodequeue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Nodes"),
		metrics:       *metricsClientset,
		postgresql:    postgres}

	_, err := nodesInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: nc.enqueueNode,
		UpdateFunc: func(old, new interface{}) {
			nc.enqueueNode(new)
		},
	})
	if err != nil {
		logger.Error(err, "Klustercost:  unable to fetch nodes")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	return nc
}

func (nc *NodeController) enqueueNode(obj interface{}) {
	node := obj.(*v1.Node)
	nc.nodequeue.Add(node.ObjectMeta.Name)
}

func (nc *NodeController) RunNode(ctx context.Context, workers int) error {

	defer runtime.HandleCrash()

	logger := klog.FromContext(ctx)
	logger.Info("Klustercost: Starting node observer threads")

	// Wait for the caches to be synced before starting workers
	logger.Info("Waiting for node informer caches to sync")

	if ok := cache.WaitForCacheSync(ctx.Done(), nc.nodesSynced); !ok {
		return fmt.Errorf("failed to wait for node caches to sync")
	}

	logger.Info("Starting workers for nodes", "count", workers)
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, nc.runWorker, time.Second)
	}

	<-ctx.Done()
	logger.Info("Done")

	return nil
}

func (nc *NodeController) runWorker(ctx context.Context) {
	for nc.processNextWorkItem(ctx) {
	}
}

func (nc *NodeController) processNextWorkItem(ctx context.Context) bool {
	obj, shutdown := nc.nodequeue.Get()
	//logger := klog.FromContext(ctx)

	if shutdown {
		return false
	}
	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		defer nc.nodequeue.Done(obj)
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
			nc.nodequeue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		nodeName, err := cache.ParseObjectName(key)
		node, err := nc.initNodeCollector(nodeName.Name)
		if err != nil {
			klog.Error("Unable to init pod collector ", err)
			return nil
		}
		record_time, node_mem, node_cpu := nc.getNodeMiscellaneous(node)
		err = nc.postgresql.insertNode(nodeName.Name, record_time, node_mem, node_cpu)

		if err != nil {
			nc.nodequeue.Forget(obj)
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

func (nc *NodeController) initNodeCollector(name string) (*v1.Node, error) {
	node, err := nc.nodesLister.Get(name)
	if err != nil {
		klog.Error("Error getting node lister ", err)
	}

	return node, err
}

func (nc *NodeController) getNodeMiscellaneous(node *v1.Node) (time.Time, int64, int64) {

	creation_time := node.CreationTimestamp.Time
	node_mem := node.Status.Allocatable.Memory().Value()
	node_cpu := node.Status.Allocatable.Cpu().Value()

	return creation_time, node_mem, node_cpu
}
