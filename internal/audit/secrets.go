package audit

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/go-crucible/go-crucible/internal/client"
	"github.com/go-crucible/go-crucible/internal/types"
)

const (
	annotationExpiryDate = "patrol.k8s.io/expiry-date"
	expiryDateFormat     = "2006-01-02"
)

// readerCloseHook is called by the tracking wrapper's Close method, if set.
// It is nil in production and set only by tests.
var readerCloseHook func()

// TestHookCloseCount installs a close hook and returns a function that returns
// the number of times Close was called since installation. Calling it again
// resets the counter. It is exported for use in external (_test) packages.
func TestHookCloseCount() int {
	return testCloseCounter
}

var testCloseCounter int

// InstallTestCloseHook sets up the close counter for tests.
// Call this before running AuditSecretExpiry in a test.
func InstallTestCloseHook() {
	testCloseCounter = 0
	readerCloseHook = func() {
		testCloseCounter++
	}
}

// UninstallTestCloseHook removes the test hook.
func UninstallTestCloseHook() {
	readerCloseHook = nil
}

// trackingReadCloser wraps an io.ReadCloser and invokes readerCloseHook on Close.
type trackingReadCloser struct {
	inner io.ReadCloser
}

func (t *trackingReadCloser) Read(p []byte) (int, error) {
	return t.inner.Read(p)
}

func (t *trackingReadCloser) Close() error {
	if readerCloseHook != nil {
		readerCloseHook()
	}
	return t.inner.Close()
}

// newSecretReader returns an io.ReadCloser over the raw bytes of a secret value.
// The returned closer invokes readerCloseHook (if set) when closed.
func newSecretReader(data []byte) io.ReadCloser {
	return &trackingReadCloser{
		inner: io.NopCloser(bytes.NewReader(data)),
	}
}

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

		// Open a reader over the secret data.
		reader := newSecretReader(secret.Data["value"])

		expiry, err := parseExpiryFromReader(reader, expiryStr)
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

// parseExpiryFromReader reads from r and parses the expiry date string.
func parseExpiryFromReader(r io.Reader, expiryStr string) (time.Time, error) {
	buf := &bytes.Buffer{}
	_, err := io.Copy(buf, r)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(expiryDateFormat, expiryStr)
}

