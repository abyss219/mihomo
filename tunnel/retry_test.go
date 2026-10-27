package tunnel

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/metacubex/mihomo/component/dialer"
)

func TestShouldStopRetryOnUnspecifiedDestination(t *testing.T) {
	if !shouldStopRetry(dialer.ErrUnspecifiedDestination) {
		t.Fatal("a refused zero destination must not be retried")
	}
	wrapped := fmt.Errorf("DIRECT connect error: %w", dialer.ErrUnspecifiedDestination)
	if !shouldStopRetry(wrapped) {
		t.Fatal("a wrapped refusal must not be retried")
	}
	if shouldStopRetry(io.EOF) {
		t.Fatal("an ordinary dial error must still be retried")
	}
	// The dialer never joins the refusal with another candidate's error, so
	// this combination cannot reach the tunnel; the assertion documents why
	// that filtering matters.
	if !shouldStopRetry(errors.Join(dialer.ErrUnspecifiedDestination, io.EOF)) {
		t.Fatal("errors.Is sees through errors.Join; the dialer must filter the refusal out of mixed sets")
	}
}
