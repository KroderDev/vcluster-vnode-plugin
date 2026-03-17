package hooks

import (
	"context"
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster-sdk/plugin"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	vnodeNodeNameLabel = "vnode.kroderdev.io/node-name"
)

func NewVNodePodHook() plugin.ClientHook {
	return &vnodePodHook{}
}

type vnodePodHook struct{}

func (h *vnodePodHook) Name() string {
	return "vnode-pod-hook"
}

func (h *vnodePodHook) Resource() client.Object {
	return &corev1.Pod{}
}

// MutateCreatePhysical intercepts when the syncer creates a pod on the host.
// If the pod has a nodeName that looks like a vnode, clear it so the host
// scheduler assigns a real node. Store the original nodeName as a label.
var _ plugin.MutateCreatePhysical = &vnodePodHook{}

func (h *vnodePodHook) MutateCreatePhysical(_ context.Context, obj client.Object) (client.Object, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("expected Pod, got %T", obj)
	}

	if pod.Spec.NodeName == "" || !isVNodeName(pod.Spec.NodeName) {
		return pod, nil
	}

	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels[vnodeNodeNameLabel] = pod.Spec.NodeName
	pod.Spec.NodeName = ""

	return pod, nil
}

// MutateGetPhysical intercepts when the syncer reads a host pod back.
// If the pod was originally on a vnode (tracking label present), fake the
// nodeName back to the vnode name so the syncer sees matching nodeNames
// and doesn't delete the virtual pod.
var _ plugin.MutateGetPhysical = &vnodePodHook{}

func (h *vnodePodHook) MutateGetPhysical(_ context.Context, obj client.Object) (client.Object, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("expected Pod, got %T", obj)
	}

	if originalNode, ok := pod.Labels[vnodeNodeNameLabel]; ok && originalNode != "" {
		pod.Spec.NodeName = originalNode
	}

	return pod, nil
}

func isVNodeName(name string) bool {
	return strings.HasPrefix(name, "vnode-")
}
