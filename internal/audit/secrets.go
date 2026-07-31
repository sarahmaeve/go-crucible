package audit

import (
	"bytes"
	"context"
	"io"
	"sync"
	"time"

	"github.com/go-crucible/go-crucible/internal/client"
	"github.com/go-crucible/go-crucible/internal/types"
)

const (
	annotationExpiryDate = "patrol.k8s.io/expiry-date"
	expiryDateFormat     = "2006-01-02"
)

var (
	readerOpenHook  func()
	readerCloseHook func()
	readerStats     struct {
		sync.Mutex
		current int
		peak    int
		closes  int
	}
)

// TestHookReaderCounts returns the current, peak, and closed reader counts
// recorded since InstallTestCloseHook was called.
func TestHookReaderCounts() (current, peak, closes int) {
	readerStats.Lock()
	defer readerStats.Unlock()
	return readerStats.current, readerStats.peak, readerStats.closes
}

// InstallTestCloseHook sets up reader lifecycle counters for tests.
// Call this before running AuditSecretExpiry in a test.
func InstallTestCloseHook() {
	readerStats.Lock()
	readerStats.current = 0
	readerStats.peak = 0
	readerStats.closes = 0
	readerStats.Unlock()
	readerOpenHook = func() {
		readerStats.Lock()
		defer readerStats.Unlock()
		readerStats.current++
		if readerStats.current > readerStats.peak {
			readerStats.peak = readerStats.current
		}
	}
	readerCloseHook = func() {
		readerStats.Lock()
		defer readerStats.Unlock()
		readerStats.current--
		readerStats.closes++
	}
}

// UninstallTestCloseHook removes the test hooks.
func UninstallTestCloseHook() {
	readerOpenHook = nil
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
	if readerOpenHook != nil {
		readerOpenHook()
	}
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
