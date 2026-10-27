package sniffer

import (
	"net/netip"
	"testing"

	C "github.com/metacubex/mihomo/constant"
)

func TestReplaceDomainKeepsMarkerWhenOverrideIsOff(t *testing.T) {
	ip := netip.MustParseAddr("140.82.113.21")
	m := &C.Metadata{Host: "guess.example", DstIP: ip, DNSMode: C.DNSMapping, MappedHost: true}

	replaceDomain(m, "real.example", false)

	if m.SniffHost != "real.example" {
		t.Fatalf("SniffHost = %q, want the observed name", m.SniffHost)
	}
	if m.Host != "guess.example" || m.DstIP != ip {
		t.Fatalf("override off must leave Host and DstIP alone, got %q %s", m.Host, m.DstIP)
	}
	if !m.MappedHost {
		t.Fatal("Host is still the reverse-map guess, so the marker must stay")
	}
	if m.DNSMode != C.DNSNormal {
		t.Fatalf("DNSMode = %v, want the unchanged upstream reset to normal", m.DNSMode)
	}
}

func TestReplaceDomainClearsMarkerWhenOverrideIsOn(t *testing.T) {
	ip := netip.MustParseAddr("140.82.113.21")
	m := &C.Metadata{Host: "guess.example", DstIP: ip, DNSMode: C.DNSMapping, MappedHost: true}

	replaceDomain(m, "real.example", true)

	if m.Host != "real.example" || m.DstIP.IsValid() {
		t.Fatalf("override on must replace Host and clear DstIP, got %q %s", m.Host, m.DstIP)
	}
	if m.MappedHost {
		t.Fatal("Host is now the observed name, so the marker must be cleared")
	}
}
