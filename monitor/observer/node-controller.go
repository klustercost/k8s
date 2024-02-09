package main

import (
	"context"
	"fmt"
	"klustercost/monitor/pkg/model"
	"klustercost/monitor/pkg/postgres"
	"strings"
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
	logger        klog.Logger
}

func NewNodeController(
	ctx context.Context,
	metricsClientset *metricsv.Clientset,
	kubeclientset kubernetes.Interface,
	informer informers.SharedInformerFactory) *NodeController {

	//logger := klog.FromContext(ctx)
	nodesInformer := informer.Core().V1().Nodes()

	nc := &NodeController{
		kubeclientset: kubeclientset,
		nodesLister:   nodesInformer.Lister(),
		nodesSynced:   nodesInformer.Informer().HasSynced,
		nodequeue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Nodes"),
		logger:        klog.FromContext(ctx)}

	_, err := nodesInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: nc.enqueueNode,
		UpdateFunc: func(old, new interface{}) {
			nc.enqueueNode(new)
		},
	})
	if err != nil {
		nc.logger.Error(err, "Klustercost:  unable to fetch nodes")
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

	nc.logger.Info("Klustercost: Starting node observer threads")

	// Wait for the caches to be synced before starting workers
	nc.logger.Info("Waiting for node informer caches to sync")

	if ok := cache.WaitForCacheSync(ctx.Done(), nc.nodesSynced); !ok {
		return fmt.Errorf("failed to wait for node caches to sync")
	}

	nc.logger.Info("Starting workers for nodes", "count", workers)
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, nc.runWorker, time.Second)
	}

	//<-ctx.Done()
	nc.logger.Info("Done")

	return nil
}

// runWorker runs a worker to process items from the workqueue
func (nc *NodeController) runWorker(ctx context.Context) {
	for nc.processNextWorkItem(ctx) {
	}
}

// processNextWorkItem processes items from the workqueue
func (nc *NodeController) processNextWorkItem(ctx context.Context) bool {
	obj, shutdown := nc.nodequeue.Get()

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
			nc.logger.Error(err, "Error parsing object name")
		}

		nodeMisc := nc.getNodeMiscellaneous(nodeName.Name)
		if err != nil {
			nc.logger.Error(err, "Unable to init pod collector ")
			return nil
		}

		err = postgres.InsertNode(nodeName.Name, nodeMisc)

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

// getNodeMiscellaneous returns the node creation time, memory, cpu and UID
func (nc *NodeController) getNodeMiscellaneous(name string) *model.NodeMisc {
	node, err := nc.nodesLister.Get(name)
	if err != nil {
		klog.Error(err, "Error getting node lister ")
	}

	nodeMisc := &model.NodeMisc{}

	nodeMisc.Memory = node.Status.Capacity.Memory().Value()
	nodeMisc.CPU = node.Status.Capacity.Cpu().Value()
	nodeMisc.UID = string(node.ObjectMeta.UID)
	nodeMisc.Labels = NodeLabelSelector(node.Labels)

	return nodeMisc
}

// NodeLabelSelector returns a string with the labels of a node
func NodeLabelSelector(labels map[string]string) string {
	var nodeLabels = []string{"node.kubernetes.io/instance-type", "topology.kubernetes.io/region", "topology.kubernetes.io/zone", "kubernetes.io/os"}
	var sb strings.Builder

	for _, label := range nodeLabels {
		value, exists := labels[label]
		sb.WriteString(label)
		if exists {
			sb.WriteString("=")
			sb.WriteString(value)
		}
		sb.WriteString(",")
	}
	// Remove the trailing comma
	result := sb.String()
	if len(result) > 0 {
		result = result[:len(result)-1]
	}
	return result
}
