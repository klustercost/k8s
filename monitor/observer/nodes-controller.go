package main

import (
	"context"
	"fmt"
	"klustercost/monitor/pkg/postgres"
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
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

type NodeController struct {
	kubeclientset kubernetes.Interface
	nodesLister   corelisters.NodeLister
	nodesSynced   cache.InformerSynced
	nodequeue     workqueue.RateLimitingInterface
}

func NewNodeController(
	ctx context.Context,
	metricsClientset *metricsv.Clientset,
	kubeclientset kubernetes.Interface,
	informer informers.SharedInformerFactory) *NodeController {

	logger := klog.FromContext(ctx)
	nodesInformer := informer.Core().V1().Nodes()

	nc := &NodeController{
		kubeclientset: kubeclientset,
		nodesLister:   nodesInformer.Lister(),
		nodesSynced:   nodesInformer.Informer().HasSynced,
		nodequeue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Nodes")}

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

func (nc *NodeController) Run(ctx context.Context, workers int) error {

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

	//<-ctx.Done()
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
		if err != nil {
			klog.Error(err)
		}

		nodeMisc := nc.getNodeMiscellaneous(nodeName.Name)
		if err != nil {
			klog.Error("Unable to init pod collector ", err)
			return nil
		}
		err = postgres.InsertNode(nodeName.Name, nodeMisc.CreationTime, nodeMisc.Memory, nodeMisc.CPU, nodeMisc.UID)

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

// NodeMisc is a struct that contains the node miscellaneous information
// It is used to insert data into the database
type NodeMisc struct {
	CreationTime time.Time
	Memory       int64
	CPU          int64
	UID          string
}

// getNodeMiscellaneous returns the node creation time, memory, cpu and UID
func (nc *NodeController) getNodeMiscellaneous(name string) *NodeMisc {
	node, err := nc.nodesLister.Get(name)
	if err != nil {
		klog.Error("Error getting node lister ", err)
	}

	nodeMisc := &NodeMisc{}

	nodeMisc.CreationTime = node.CreationTimestamp.Time
	nodeMisc.Memory = node.Status.Capacity.Memory().Value()
	nodeMisc.CPU = node.Status.Capacity.Cpu().Value()
	nodeMisc.UID = string(node.ObjectMeta.UID)

	return nodeMisc
}
