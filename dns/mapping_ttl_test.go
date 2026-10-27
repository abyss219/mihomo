package dns

import (
	"net/netip"
	"testing"
	"time"

	"github.com/metacubex/mihomo/common/lru"
	C "github.com/metacubex/mihomo/constant"
)

func newMappingEnhancer(mode C.DNSMode) *ResolverEnhancer {
	return &ResolverEnhancer{mode: mode, mapping: lru.New(lru.WithSize[netip.Addr, string](16))}
}

func TestFindHostByIPHonorsTTLOnlyInRedirHost(t *testing.T) {
	ip := netip.MustParseAddr("140.82.113.21")
	fresh := time.Now().Add(time.Hour)
	stale := time.Now().Add(-time.Minute)

	redirHost := newMappingEnhancer(C.DNSMapping)
	redirHost.mapping.SetWithExpire(ip, "live.example", fresh)
	if host, ok := redirHost.FindHostByIP(ip); !ok || host != "live.example" {
		t.Fatalf("a live entry must be found, got %q %v", host, ok)
	}
	redirHost.mapping.SetWithExpire(ip, "stale.example", stale)
	if host, ok := redirHost.FindHostByIP(ip); ok {
		t.Fatalf("an expired entry must be treated as missing in redir-host, got %q", host)
	}
	if _, _, ok := redirHost.mapping.GetWithExpire(ip); ok {
		t.Fatal("an expired entry must be dropped from the map")
	}
	if _, ok := redirHost.FindHostByIP(netip.MustParseAddr("203.0.113.9")); ok {
		t.Fatal("an unknown IP must stay unknown")
	}

	fakeIP := newMappingEnhancer(C.DNSFakeIP)
	fakeIP.mapping.SetWithExpire(ip, "stale.example", stale)
	if host, ok := fakeIP.FindHostByIP(ip); !ok || host != "stale.example" {
		t.Fatalf("fake-ip must keep the upstream behavior of never expiring entries, got %q %v", host, ok)
	}
}
