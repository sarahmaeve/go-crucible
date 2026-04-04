package client

import (
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
func (f *FakeClient) ListPods(namespace string) ([]corev1.Pod, error) {
	return (&KubeClient{cs: f.cs}).ListPods(namespace)
}

// ListDeployments returns all deployments in the fake clientset for the given namespace.
func (f *FakeClient) ListDeployments(namespace string) ([]appsv1.Deployment, error) {
	return (&KubeClient{cs: f.cs}).ListDeployments(namespace)
}

// ListSecrets returns all secrets in the fake clientset for the given namespace.
func (f *FakeClient) ListSecrets(namespace string) ([]corev1.Secret, error) {
	return (&KubeClient{cs: f.cs}).ListSecrets(namespace)
}

// ErrorClient is an AuditClient that always returns the configured errors.
// It is used to exercise error-handling paths.
type ErrorClient struct {
	PodError        error
	DeploymentError error
	SecretError     error
}

func (e *ErrorClient) ListPods(_ string) ([]corev1.Pod, error) {
	return nil, e.PodError
}

func (e *ErrorClient) ListDeployments(_ string) ([]appsv1.Deployment, error) {
	return nil, e.DeploymentError
}

func (e *ErrorClient) ListSecrets(_ string) ([]corev1.Secret, error) {
	return nil, e.SecretError
}
