package hooks

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
