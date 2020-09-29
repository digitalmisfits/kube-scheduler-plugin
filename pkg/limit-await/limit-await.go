package limit_await

import (
	"context"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	v12 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	v1Pod "k8s.io/kubernetes/pkg/api/v1/pod"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
	"k8s.io/kubernetes/pkg/scheduler/util"
	"math"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
)

const (
	Name         = "LimitAwait"
	PollInterval = 15000
)

type LimitAwaitScheduling struct {
	frameworkHandle  framework.FrameworkHandle
	podLister        v12.PodLister
	nodeLister       v12.NodeLister
	waitingPods      map[string]bool // default false
	waitingPodsMutex sync.RWMutex
	clock            util.Clock
}

var _ framework.PermitPlugin = &LimitAwaitScheduling{}

func New(configuration runtime.Object, handle framework.FrameworkHandle) (framework.Plugin, error) {
	klog.Infof("Creating LimitAwaitScheduling plugin with configuration %s", configuration)

	podLister := handle.SharedInformerFactory().Core().V1().Pods().Lister()
	nodeLister := handle.SharedInformerFactory().Core().V1().Nodes().Lister()

	podInformer := handle.SharedInformerFactory().Core().V1().Pods().Informer()
	podInformer.AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch t := obj.(type) {
				case *v1.Pod:
					return true
				case cache.DeletedFinalStateUnknown:
					if _, ok := t.Obj.(*v1.Pod); ok {
						return true
					}
					return false
				default:
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					//klog.Infof("Adding pod(%s) state", obj.(*v1.Pod).Name, obj.(*v1.Pod).Status)
				},
				DeleteFunc: func(obj interface{}) {
					//klog.Infof("Deleting pod(%s)", obj.(*v1.Pod).Name)
				},
				UpdateFunc: func(old, new interface{}) {
					//klog.Infof("Update pod(%s) state ", new.(*v1.Pod).Name, new.(*v1.Pod).Status)

					if old.(*v1.Pod).ResourceVersion == new.(*v1.Pod).ResourceVersion {
						klog.Infof("No state changed for pod(%s). ", old.(*v1.Pod).Name)
						return
					}

					//if !reflect.DeepEqual(old.(*v1.Pod).Status, new.(*v1.Pod).Status) {
					//klog.Infof("Status changed from %s to %s. ", old.(*v1.Pod).Status, new.(*v1.Pod).Status)
					//}
				},
			},
		})

	plugin := &LimitAwaitScheduling{
		frameworkHandle: handle,
		podLister:       podLister,
		nodeLister:      nodeLister,
		clock:           util.RealClock{},
	}

	go func(s *LimitAwaitScheduling, parallelism int) {
		timerCh := time.Tick(time.Duration(PollInterval) * time.Millisecond)
		for range timerCh {
			availableSlots := make(map[string]int) // default zero for all types

			// add plugin to determine pending/scheduled=false based on last transition time
			notReadyPodsPerNode, _ := s.GetPodsForNodes(plugin.podNotReady /* and */, plugin.podNotWaiting)
			for nodeName, nodeReadyNodeInfo := range notReadyPodsPerNode {
				notReadyPodNames := make([]string, len(nodeReadyNodeInfo.Pods))
				for _, podInfo := range nodeReadyNodeInfo.Pods {
					notReadyPodNames = append(notReadyPodNames, podInfo.Pod.Name)
				}
				availableSlots[nodeName] = parallelism - int(math.Min(float64(parallelism), float64(len(nodeReadyNodeInfo.Pods))))

				klog.Infof("Node(%s, available-slots=%d), not ready pods: %s", nodeName, availableSlots[nodeName], notReadyPodNames)
			}

			s.frameworkHandle.IterateOverWaitingPods(func(waitingPod framework.WaitingPod) {
				assignedNode := waitingPod.GetPod().Spec.NodeName

				if availableSlots[assignedNode] > 0 {
					availableSlots[assignedNode]--

					klog.Infof("Allowing WaitingPod(%s) for node(%s, available-slots=%d)", waitingPod.GetPod().Name, assignedNode, availableSlots[assignedNode])
					waitingPod.Allow(Name)
				}
			})
		}
	}(plugin, 4)

	return plugin, nil
}

func (s *LimitAwaitScheduling) Name() string {
	return Name
}

func (s *LimitAwaitScheduling) podNotReady(pod *v1.Pod) bool {
	switch pod.Status.Phase {
	case v1.PodPending:
		//klog.Infof("PodPending(%s)", pod.Name)
		return true
	case v1.PodRunning:
		//klog.Infof("PodRunning(%s)", pod.Name)
		return v1Pod.IsPodReady(pod) == false
	case v1.PodSucceeded:
		//klog.Infof("PodSucceeded(%s)", pod.Name)
		return false
	case v1.PodFailed:
		//klog.Infof("PodFailed(%s)", pod.Name)
		return false
	case v1.PodUnknown:
		//klog.Infof("PodUnknown(%s)", pod.Name)
		return false
	default:
		return false
	}
}

func (s *LimitAwaitScheduling) Permit(_ context.Context, _ *framework.CycleState, pod *v1.Pod, nodeName string) (*framework.Status, time.Duration) {
	if pod.Namespace == meta.NamespaceSystem {
		return framework.NewStatus(framework.Success, ""), 0
	}

	klog.Infof("Awaiting permit for pod(%s) on node(%s)", pod.Name, pod.Spec.NodeName)
	return framework.NewStatus(framework.Wait, "queued"), 10 * time.Minute
}

func (s *LimitAwaitScheduling) podNotWaiting(pod *v1.Pod) bool {
	return !s.isAwaitingPermit(pod)
}

func (s *LimitAwaitScheduling) isAwaitingPermit(pod *v1.Pod) bool {
	waitingPod := s.frameworkHandle.GetWaitingPod(pod.UID)
	if waitingPod == nil {
		return false
	}
	return true
}

func (s *LimitAwaitScheduling) GetPodsForNodes(filters ...func(pod *v1.Pod) bool) (map[string]*framework.NodeInfo, error) {

	pods, err := s.podLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to retrieve pods using the podLister")
		return nil, err
	}

	nodes, err := s.nodeLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to retrieve nodes using the nodeLister")
		return nil, err
	}

	return createNodeInfoMap(nodes, pods, filters...), nil
}

func createNodeInfoMap(nodes []*v1.Node, pods []*v1.Pod, filters ...func(pod *v1.Pod) bool) map[string]*framework.NodeInfo {
	nodeNameToInfo := make(map[string]*framework.NodeInfo)
	for _, pod := range pods {
		var shouldApply = true
		for _, filter := range filters {
			shouldApply = shouldApply && filter(pod)
			if shouldApply == false {
				break // short-circuit
			}
		}
		if shouldApply {
			nodeName := pod.Spec.NodeName
			if _, ok := nodeNameToInfo[nodeName]; !ok {
				nodeNameToInfo[nodeName] = framework.NewNodeInfo()
			}
			nodeNameToInfo[nodeName].AddPod(pod)
		}
	}

	for _, node := range nodes {
		if _, ok := nodeNameToInfo[node.Name]; !ok {
			nodeNameToInfo[node.Name] = framework.NewNodeInfo()
		}
		nodeInfo := nodeNameToInfo[node.Name]
		err := nodeInfo.SetNode(node)
		if err != nil {
			panic("Failed to set node on nodeInfo")
		}
	}
	return nodeNameToInfo
}
