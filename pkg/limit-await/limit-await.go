package limit_await

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
	"k8s.io/kubernetes/pkg/scheduler/util"
	"time"

	v1 "k8s.io/api/core/v1"
)

const (
	Name           = "LimitAwaitScheduling"
	LimitAWaitName = "limit-await.scheduling.k8s.io/name"
)

type LimitAwaitScheduling struct {
	frameworkHandle framework.FrameworkHandle
	clock           util.Clock
}

func New(obj runtime.Object, handle framework.FrameworkHandle) (framework.Plugin, error) {
	return &LimitAwaitScheduling{
		frameworkHandle: handle,
		clock:           util.RealClock{},
	}, nil
}

func (cs *LimitAwaitScheduling) Name() string {
	return Name
}

func (cs *LimitAwaitScheduling) Permit(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (*framework.Status, time.Duration) {
	// note: snapshot is taking at the beginning of the schedule cycle therefore it does not perfectly reflect the current state
	nodeInfo, err := cs.frameworkHandle.SnapshotSharedLister().NodeInfos().Get(nodeName)

	if err != nil {
		return framework.NewStatus(framework.Error, "Failed to get node information from the snapshot shared lister"), 0
	}

	for _, podInfo := range nodeInfo.Pods {
		klog.Infof("Pod(%s) status: %s", podInfo.Pod.Name, podInfo.Pod.Status)
	}

	klog.Infof("Permit pod(%s) for node(%s)", pod.Name, nodeInfo.Node().Name)
	return framework.NewStatus(framework.Success, ""), 0
}

var _ framework.PermitPlugin = &LimitAwaitScheduling{}
