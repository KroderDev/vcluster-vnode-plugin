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
		name            string
		nodeName        string
		initialLabels   map[string]string
		wantClear       bool
		wantLabel       string
		wantLabelsUnchg bool
	}{
		{name: "no nodeName", nodeName: "", wantClear: false, wantLabel: "", wantLabelsUnchg: true},
		{name: "non-vnode nodeName", nodeName: "worker-1", initialLabels: map[string]string{"app": "demo"}, wantClear: false, wantLabel: "", wantLabelsUnchg: true},
		{name: "vnode nodeName", nodeName: "vnode-abc-1", wantClear: true, wantLabel: "vnode-abc-1"},
		{name: "vnode prefix only", nodeName: "vnode-x", wantClear: true, wantLabel: "vnode-x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default", Labels: tt.initialLabels},
				Spec:       corev1.PodSpec{NodeName: tt.nodeName},
			}

			result, err := hook.MutateCreatePhysical(context.Background(), pod)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			resultPod := result.(*corev1.Pod)
			if resultPod != pod {
				t.Fatalf("expected pod to be mutated in place")
			}
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
				if tt.wantLabelsUnchg {
					if len(resultPod.Labels) != len(tt.initialLabels) {
						t.Fatalf("expected labels to remain unchanged, got %#v", resultPod.Labels)
					}
					for k, v := range tt.initialLabels {
						if got := resultPod.Labels[k]; got != v {
							t.Fatalf("expected label %q=%q to remain unchanged, got %q", k, v, got)
						}
					}
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
	if got := err.Error(); got != "expected Pod, got *v1.ConfigMap" {
		t.Fatalf("unexpected error message: %q", got)
	}
	if result != nil {
		t.Fatalf("expected nil result for non-pod object, got %T", result)
	}
}

func TestMutateCreatePhysicalOverwritesExistingVNodeLabel(t *testing.T) {
	hook := &vnodePodHook{}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			Labels: map[string]string{
				vnodeNodeNameLabel: "stale-value",
				"app":              "demo",
			},
		},
		Spec: corev1.PodSpec{NodeName: "vnode-fresh"},
	}

	result, err := hook.MutateCreatePhysical(context.Background(), pod)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultPod := result.(*corev1.Pod)
	if resultPod != pod {
		t.Fatalf("expected pod to be mutated in place")
	}
	if got := resultPod.Labels[vnodeNodeNameLabel]; got != "vnode-fresh" {
		t.Fatalf("expected vnode label to be overwritten, got %q", got)
	}
	if got := resultPod.Labels["app"]; got != "demo" {
		t.Fatalf("expected unrelated labels to remain intact, got %q", got)
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
			if resultPod != pod {
				t.Fatalf("expected pod to be mutated in place")
			}
			if resultPod.Spec.NodeName != tt.wantNodeName {
				t.Errorf("expected nodeName %q, got %q", tt.wantNodeName, resultPod.Spec.NodeName)
			}
		})
	}
}

func TestMutateVirtualReadPathsRestoreVNodeNodeName(t *testing.T) {
	hook := &vnodePodHook{}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
			Labels: map[string]string{
				vnodeNodeNameLabel: "vnode-abc-1",
			},
		},
		Spec: corev1.PodSpec{NodeName: "k8s-master-01"},
	}

	t.Run("get virtual", func(t *testing.T) {
		result, err := hook.MutateGetVirtual(context.Background(), pod.DeepCopy())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := result.(*corev1.Pod).Spec.NodeName; got != "vnode-abc-1" {
			t.Fatalf("expected vnode nodeName, got %q", got)
		}
	})

	t.Run("update virtual", func(t *testing.T) {
		result, err := hook.MutateUpdateVirtual(context.Background(), pod.DeepCopy())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := result.(*corev1.Pod).Spec.NodeName; got != "vnode-abc-1" {
			t.Fatalf("expected vnode nodeName, got %q", got)
		}
	})
}

func TestMutateGetPhysicalWithNilLabels(t *testing.T) {
	hook := &vnodePodHook{}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec:       corev1.PodSpec{NodeName: "worker-1"},
	}

	result, err := hook.MutateGetPhysical(context.Background(), pod)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultPod := result.(*corev1.Pod)
	if resultPod != pod {
		t.Fatalf("expected pod to be mutated in place")
	}
	if got := resultPod.Spec.NodeName; got != "worker-1" {
		t.Fatalf("expected nodeName to remain unchanged, got %q", got)
	}
	if resultPod.Labels != nil {
		t.Fatalf("expected labels to remain nil, got %#v", resultPod.Labels)
	}
}

func TestMutateGetPhysicalRejectsNonPod(t *testing.T) {
	hook := &vnodePodHook{}

	result, err := hook.MutateGetPhysical(context.Background(), &corev1.ConfigMap{})
	if err == nil {
		t.Fatal("expected an error for non-pod object")
	}
	if got := err.Error(); got != "expected Pod, got *v1.ConfigMap" {
		t.Fatalf("unexpected error message: %q", got)
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
