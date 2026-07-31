package audit_test

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-crucible/go-crucible/internal/audit"
)

// directSecretsClient is a minimal AuditClient that returns a fixed set of
// corev1.Secrets. Used to test AuditSecretExpiry without a real cluster.
type directSecretsClient struct {
	secrets []corev1.Secret
}

func (d *directSecretsClient) ListPods(_ context.Context, _ string) ([]corev1.Pod, error) {
	return nil, nil
}
func (d *directSecretsClient) ListDeployments(_ context.Context, _ string) ([]appsv1.Deployment, error) {
	return nil, nil
}
func (d *directSecretsClient) ListSecrets(_ context.Context, _ string) ([]corev1.Secret, error) {
	result := make([]corev1.Secret, len(d.secrets))
	copy(result, d.secrets)
	return result, nil
}

// TestAuditSecretExpiry verifies the secret-expiry audit independently from
// the remote rule-loading exercise.
func TestAuditSecretExpiry(t *testing.T) {
	past := time.Now().Add(-48 * time.Hour).Format("2006-01-02")
	future := time.Now().Add(48 * time.Hour).Format("2006-01-02")

	secrets := []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "db-password",
				Namespace: "default",
				Annotations: map[string]string{
					"patrol.k8s.io/expiry-date": past,
				},
			},
			Data: map[string][]byte{"value": []byte("hunter2")},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "api-token",
				Namespace: "default",
				Annotations: map[string]string{
					"patrol.k8s.io/expiry-date": future,
				},
			},
			Data: map[string][]byte{"value": []byte("t0k3n")},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				// No expiry annotation — no reader should be opened.
				Name:      "no-expiry",
				Namespace: "default",
			},
			Data: map[string][]byte{"value": []byte("ignored")},
		},
	}

	fc := &directSecretsClient{secrets: secrets}

	findings, err := audit.AuditSecretExpiry(t.Context(), fc, "default")
	if err != nil {
		t.Fatalf("unexpected error from AuditSecretExpiry: %v", err)
	}

	// Sanity-check: only db-password is expired; api-token is in the future.
	if len(findings) != 1 {
		t.Errorf("expected 1 finding (expired db-password), got %d", len(findings))
	}

}
