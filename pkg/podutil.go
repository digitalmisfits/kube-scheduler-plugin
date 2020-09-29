package pkg

import (
	v1 "k8s.io/api/core/v1"
	v1Meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1Pod "k8s.io/kubernetes/pkg/api/v1/pod"
	"time"
)

func hasStableStatus(pod *v1.Pod, conditionType v1.PodConditionType, minTransitionSeconds int32, now v1Meta.Time) bool {
	exists, condition := v1Pod.GetPodCondition(&pod.Status, conditionType)
	if exists < 0 {
		return false
	}

	minTransitionSecondsDuration := time.Duration(minTransitionSeconds) * time.Second
	if minTransitionSeconds == 0 || !condition.LastTransitionTime.IsZero() &&
		condition.LastTransitionTime.Add(minTransitionSecondsDuration).Before(now.Time) {
		return true
	}
	return false
}
