package audit

import (
	"context"
	"time"

	"github.com/go-crucible/go-crucible/internal/client"
	"github.com/go-crucible/go-crucible/internal/types"
)

const (
	annotationExpiryDate = "patrol.k8s.io/expiry-date"
	expiryDateFormat     = "2006-01-02"
)

// AuditSecretExpiry inspects every secret in namespace and returns a Finding
// for each secret whose "patrol.k8s.io/expiry-date" annotation is in the past.
func AuditSecretExpiry(ctx context.Context, c client.AuditClient, namespace string) ([]types.Finding, error) {
	secrets, err := c.ListSecrets(ctx, namespace)
	if err != nil {
		return nil, err
	}

	var findings []types.Finding
	now := time.Now()

	for _, secret := range secrets {
		expiryStr, ok := secret.Annotations[annotationExpiryDate]
		if !ok {
			continue
		}

		expiry, err := time.Parse(expiryDateFormat, expiryStr)
		if err != nil {
			continue
		}

		if expiry.Before(now) {
			findings = append(findings, types.Finding{
				Resource:    "Secret",
				Namespace:   secret.Namespace,
				Name:        secret.Name,
				Severity:    types.SeverityCritical,
				Message:     "secret has expired: expiry date was " + expiryStr,
				Annotations: secret.Annotations,
			})
		}
	}

	return findings, nil
}
