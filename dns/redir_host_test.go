package dns

import (
	"testing"

	C "github.com/metacubex/mihomo/constant"
)

func TestRedirHostEnabledIsFalseOutsideRedirHostMode(t *testing.T) {
	for _, tc := range []struct {
		mode      C.DNSMode
		redirHost bool
		mapping   bool
		fakeIP    bool
	}{
		{C.DNSNormal, false, false, false},
		{C.DNSFakeIP, false, true, true},
		{C.DNSMapping, true, true, false},
	} {
		h := &ResolverEnhancer{mode: tc.mode}
		if got := h.RedirHostEnabled(); got != tc.redirHost {
			t.Errorf("mode %v: RedirHostEnabled() = %v, want %v", tc.mode, got, tc.redirHost)
		}
		if got := h.MappingEnabled(); got != tc.mapping {
			t.Errorf("mode %v: MappingEnabled() = %v, want %v", tc.mode, got, tc.mapping)
		}
		if got := h.FakeIPEnabled(); got != tc.fakeIP {
			t.Errorf("mode %v: FakeIPEnabled() = %v, want %v", tc.mode, got, tc.fakeIP)
		}
	}
}
