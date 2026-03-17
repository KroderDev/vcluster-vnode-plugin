package hooks

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewVNodePodHookMetadata(t *testing.T) {
	hook := NewVNodePodHook()

	if got := hook.Name(); got != "vnode-pod-hook" {
		t.Fatalf("expected hook name %q, got %q", "vnode-pod-hook", got)
	}

	if _, ok := hook.Resource().(*corev1.Pod); !ok {
		t.Fatalf("expected hook resource to be *corev1.Pod, got %T", hook.Resource())
	}
}

func TestMutateCreatePhysical(t *testing.T) {
	hook := &vnodePodHook{}

	tests := []struct {
		name         string
		nodeName     string
		wantClear    bool
		wantLabel    string
	}{
		{"no nodeName", "", false, ""},
		{"non-vnode nodeName", "worker-1", false, ""},
		{"vnode nodeName", "vnode-abc-1", true, "vnode-abc-1"},
		{"vnode prefix only", "vnode-x", true, "vnode-x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
				Spec:       corev1.PodSpec{NodeName: tt.nodeName},
			}

			result, err := hook.MutateCreatePhysical(context.Background(), pod)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			resultPod := result.(*corev1.Pod)
			if tt.wantClear {
				if resultPod.Spec.NodeName != "" {
					t.Errorf("expected empty nodeName, got %q", resultPod.Spec.NodeName)
				}
				if resultPod.Labels[vnodeNodeNameLabel] != tt.wantLabel {
					t.Errorf("expected label %q, got %q", tt.wantLabel, resultPod.Labels[vnodeNodeNameLabel])
				}
			} else {
				if resultPod.Spec.NodeName != tt.nodeName {
					t.Errorf("expected nodeName %q unchanged, got %q", tt.nodeName, resultPod.Spec.NodeName)
				}
			}
		})
	}
}

func TestMutateCreatePhysicalInitializesAndPreservesLabels(t *testing.T) {
	hook := &vnodePodHook{}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			Labels: map[string]string{
				"app": "demo",
			},
		},
		Spec: corev1.PodSpec{NodeName: "vnode-abc-1"},
	}

	result, err := hook.MutateCreatePhysical(context.Background(), pod)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultPod := result.(*corev1.Pod)
	if resultPod != pod {
		t.Fatalf("expected pod to be mutated in place")
	}
	if got := resultPod.Labels["app"]; got != "demo" {
		t.Fatalf("expected existing label to be preserved, got %q", got)
	}
	if got := resultPod.Labels[vnodeNodeNameLabel]; got != "vnode-abc-1" {
		t.Fatalf("expected vnode label to be set, got %q", got)
	}
	if resultPod.Spec.NodeName != "" {
		t.Fatalf("expected nodeName to be cleared, got %q", resultPod.Spec.NodeName)
	}
}

func TestMutateCreatePhysicalRejectsNonPod(t *testing.T) {
	hook := &vnodePodHook{}

	result, err := hook.MutateCreatePhysical(context.Background(), &corev1.ConfigMap{})
	if err == nil {
		t.Fatal("expected an error for non-pod object")
	}
	if result != nil {
		t.Fatalf("expected nil result for non-pod object, got %T", result)
	}
}

func TestMutateGetPhysical(t *testing.T) {
	hook := &vnodePodHook{}

	tests := []struct {
		name         string
		nodeName     string
		label        string
		wantNodeName string
	}{
		{"no label", "worker-1", "", "worker-1"},
		{"with label", "worker-1", "vnode-abc-1", "vnode-abc-1"},
		{"empty label", "worker-1", "", "worker-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			labels := map[string]string{}
			if tt.label != "" {
				labels[vnodeNodeNameLabel] = tt.label
			}

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Labels: labels},
				Spec:       corev1.PodSpec{NodeName: tt.nodeName},
			}

			result, err := hook.MutateGetPhysical(context.Background(), pod)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			resultPod := result.(*corev1.Pod)
			if resultPod.Spec.NodeName != tt.wantNodeName {
				t.Errorf("expected nodeName %q, got %q", tt.wantNodeName, resultPod.Spec.NodeName)
			}
		})
	}
}

func TestMutateGetPhysicalRejectsNonPod(t *testing.T) {
	hook := &vnodePodHook{}

	result, err := hook.MutateGetPhysical(context.Background(), &corev1.ConfigMap{})
	if err == nil {
		t.Fatal("expected an error for non-pod object")
	}
	if result != nil {
		t.Fatalf("expected nil result for non-pod object, got %T", result)
	}
}

func TestIsVNodeName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "empty", in: "", want: false},
		{name: "plain node", in: "worker-1", want: false},
		{name: "exact prefix", in: "vnode-", want: true},
		{name: "vnode name", in: "vnode-abc-1", want: true},
		{name: "embedded vnode", in: "worker-vnode-1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isVNodeName(tt.in); got != tt.want {
				t.Fatalf("expected %v for %q, got %v", tt.want, tt.in, got)
			}
		})
	}
}
