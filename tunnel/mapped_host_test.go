package tunnel

import (
	"net/netip"
	"testing"

	"github.com/metacubex/mihomo/component/resolver"
	C "github.com/metacubex/mihomo/constant"
)

// fakeEnhancer stands in for the DNS enhancer: a reverse map plus the mode.
type fakeEnhancer struct {
	redirHost bool
	fakeIP    bool
	names     map[netip.Addr]string
	fakeIPs   map[netip.Addr]bool
}

func (f *fakeEnhancer) FakeIPEnabled() bool               { return f.fakeIP }
func (f *fakeEnhancer) MappingEnabled() bool              { return f.fakeIP || f.redirHost }
func (f *fakeEnhancer) RedirHostEnabled() bool            { return f.redirHost }
func (f *fakeEnhancer) IsFakeIP(ip netip.Addr) bool       { return f.fakeIP && f.fakeIPs[ip] }
func (f *fakeEnhancer) IsFakeBroadcastIP(netip.Addr) bool { return false }
func (f *fakeEnhancer) IsExistFakeIP(ip netip.Addr) bool  { return f.IsFakeIP(ip) }
func (f *fakeEnhancer) FindHostByIP(ip netip.Addr) (string, bool) {
	host, ok := f.names[ip]
	return host, ok
}
func (f *fakeEnhancer) FlushFakeIP() error                        { return nil }
func (f *fakeEnhancer) InsertHostByIP(ip netip.Addr, host string) { f.names[ip] = host }
func (f *fakeEnhancer) StoreFakePoolState()                       {}

func installEnhancer(t *testing.T, e resolver.Enhancer) {
	t.Helper()
	old := resolver.DefaultHostMapper
	resolver.DefaultHostMapper = e
	t.Cleanup(func() { resolver.DefaultHostMapper = old })
}

func TestPreHandleMetadataMarksReverseMapGuessesOnlyInRedirHost(t *testing.T) {
	real := netip.MustParseAddr("140.82.113.21")
	unknown := netip.MustParseAddr("203.0.113.9")
	fake := netip.MustParseAddr("198.18.0.5")
	fakeMissing := netip.MustParseAddr("198.18.0.6")

	redirHost := &fakeEnhancer{
		redirHost: true,
		names:     map[netip.Addr]string{real: "guess.example"},
	}
	fakeIPMode := &fakeEnhancer{
		fakeIP:  true,
		names:   map[netip.Addr]string{real: "filtered.example", fake: "fake.example"},
		fakeIPs: map[netip.Addr]bool{fake: true, fakeMissing: true},
	}

	tests := []struct {
		name       string
		enhancer   *fakeEnhancer
		in         C.Metadata
		wantHost   string
		wantDstIP  netip.Addr
		wantMode   C.DNSMode
		wantMarked bool
		wantErr    bool
	}{
		{"redir-host: reverse-map hit is marked", redirHost, C.Metadata{DstIP: real}, "guess.example", real, C.DNSMapping, true, false},
		{"redir-host: reverse-map miss stays nameless", redirHost, C.Metadata{DstIP: unknown}, "", unknown, C.DNSNormal, false, false},
		{"redir-host: client-supplied domain is untouched", redirHost, C.Metadata{Host: "client.example"}, "client.example", netip.Addr{}, C.DNSNormal, false, false},
		{"fake-ip: filtered real IP is mapped but not marked", fakeIPMode, C.Metadata{DstIP: real}, "filtered.example", real, C.DNSMapping, false, false},
		{"fake-ip: pool hit clears the IP and is not marked", fakeIPMode, C.Metadata{DstIP: fake}, "fake.example", netip.Addr{}, C.DNSFakeIP, false, false},
		{"fake-ip: missing record is still an error", fakeIPMode, C.Metadata{DstIP: fakeMissing}, "", fakeMissing, C.DNSNormal, false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			installEnhancer(t, tc.enhancer)
			m := tc.in
			err := preHandleMetadata(&m)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if m.Host != tc.wantHost {
				t.Errorf("Host = %q, want %q", m.Host, tc.wantHost)
			}
			if m.DstIP != tc.wantDstIP {
				t.Errorf("DstIP = %s, want %s", m.DstIP, tc.wantDstIP)
			}
			if m.DNSMode != tc.wantMode {
				t.Errorf("DNSMode = %v, want %v", m.DNSMode, tc.wantMode)
			}
			if m.MappedHost != tc.wantMarked {
				t.Errorf("MappedHost = %v, want %v", m.MappedHost, tc.wantMarked)
			}
		})
	}
}
