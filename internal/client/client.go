// Package client provides a Kubernetes audit client interface and implementation.
package client

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// AuditClient abstracts the Kubernetes API calls needed for auditing.
// All methods accept a [context.Context] so callers can enforce deadlines and
// cancellation. Callers should propagate their own context rather than
// passing [context.Background].
type AuditClient interface {
	ListPods(ctx context.Context, namespace string) ([]corev1.Pod, error)
	ListDeployments(ctx context.Context, namespace string) ([]appsv1.Deployment, error)
	ListSecrets(ctx context.Context, namespace string) ([]corev1.Secret, error)
}

// KubeClient is the real implementation of AuditClient backed by a live cluster.
type KubeClient struct {
	cs kubernetes.Interface
}

// ListPods returns all pods in the given namespace.
func (k *KubeClient) ListPods(ctx context.Context, namespace string) ([]corev1.Pod, error) {
	list, err := k.cs.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListDeployments returns all deployments in the given namespace.
func (k *KubeClient) ListDeployments(ctx context.Context, namespace string) ([]appsv1.Deployment, error) {
	list, err := k.cs.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListSecrets returns all secrets in the given namespace.
func (k *KubeClient) ListSecrets(ctx context.Context, namespace string) ([]corev1.Secret, error) {
	list, err := k.cs.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// NewAuditClient builds an AuditClient from the given kubeconfig path.
// If kubeconfig is empty, in-cluster config is attempted.
func NewAuditClient(kubeconfig string) (AuditClient, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return (*KubeClient)(nil), nil
	}

	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return (*KubeClient)(nil), nil
	}

	return &KubeClient{cs: cs}, nil
}
