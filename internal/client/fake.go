package client

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

// FakeClient wraps client-go's fake clientset and implements AuditClient.
// It is intended for use in tests.
type FakeClient struct {
	cs *fake.Clientset
}

// NewFakeClient creates a FakeClient pre-populated with the provided objects.
// Accepted object types: *corev1.Pod, *appsv1.Deployment, *corev1.Secret.
func NewFakeClient(objs ...runtime.Object) *FakeClient {
	cs := fake.NewClientset(objs...)
	return &FakeClient{cs: cs}
}

// ListPods returns all pods in the fake clientset for the given namespace.
func (f *FakeClient) ListPods(ctx context.Context, namespace string) ([]corev1.Pod, error) {
	return (&KubeClient{cs: f.cs}).ListPods(ctx, namespace)
}

// ListDeployments returns all deployments in the fake clientset for the given namespace.
func (f *FakeClient) ListDeployments(ctx context.Context, namespace string) ([]appsv1.Deployment, error) {
	return (&KubeClient{cs: f.cs}).ListDeployments(ctx, namespace)
}

// ListSecrets returns all secrets in the fake clientset for the given namespace.
func (f *FakeClient) ListSecrets(ctx context.Context, namespace string) ([]corev1.Secret, error) {
	return (&KubeClient{cs: f.cs}).ListSecrets(ctx, namespace)
}

// ErrorClient is an AuditClient that always returns the configured errors.
// It is used to exercise error-handling paths.
type ErrorClient struct {
	PodError        error
	DeploymentError error
	SecretError     error
}

// ListPods always returns (nil, e.PodError).
func (e *ErrorClient) ListPods(_ context.Context, _ string) ([]corev1.Pod, error) {
	return nil, e.PodError
}

// ListDeployments always returns (nil, e.DeploymentError).
func (e *ErrorClient) ListDeployments(_ context.Context, _ string) ([]appsv1.Deployment, error) {
	return nil, e.DeploymentError
}

// ListSecrets always returns (nil, e.SecretError).
func (e *ErrorClient) ListSecrets(_ context.Context, _ string) ([]corev1.Secret, error) {
	return nil, e.SecretError
}
