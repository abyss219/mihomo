package constant

import (
	"net/netip"
	"testing"
)

func TestPureStripsGuessedAndMappedHosts(t *testing.T) {
	ip := netip.MustParseAddr("140.82.113.21")

	for _, tc := range []struct {
		name  string
		in    Metadata
		strip bool
	}{
		{"redir-host mapping", Metadata{Host: "m.example", DstIP: ip, DNSMode: DNSMapping}, true},
		{"hosts entry", Metadata{Host: "m.example", DstIP: ip, DNSMode: DNSHosts}, true},
		{"marked guess after a sniff reset the mode", Metadata{Host: "m.example", DstIP: ip, DNSMode: DNSNormal, MappedHost: true}, true},
		{"marked guess without an IP", Metadata{Host: "m.example", MappedHost: true}, false},
		{"client-supplied domain", Metadata{Host: "d.example", DstIP: ip, DNSMode: DNSNormal}, false},
		{"fake-ip", Metadata{Host: "f.example", DNSMode: DNSFakeIP}, false},
		{"fake-ip filtered domain after a sniff", Metadata{Host: "f.example", DstIP: ip, DNSMode: DNSNormal}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := tc.in
			out := in.Pure()
			if tc.strip {
				if out == &in {
					t.Fatal("Pure must return a copy when it strips Host")
				}
				if out.Host != "" || out.DstIP != in.DstIP {
					t.Fatalf("got Host %q DstIP %s, want Host stripped and DstIP kept", out.Host, out.DstIP)
				}
				if in.Host == "" {
					t.Fatal("the original must not be modified")
				}
			} else if out != &in {
				t.Fatal("Pure must return the same pointer when nothing is stripped")
			}
		})
	}
}
