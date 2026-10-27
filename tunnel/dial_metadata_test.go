package tunnel

import (
	"net/netip"
	"testing"

	"github.com/metacubex/mihomo/component/resolver"
	"github.com/metacubex/mihomo/component/trie"
	C "github.com/metacubex/mihomo/constant"
)

func installHosts(t *testing.T, name string, ip netip.Addr) {
	t.Helper()
	tr := trie.New[resolver.HostValue]()
	hv, err := resolver.NewHostValueByIPs([]netip.Addr{ip})
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.Insert(name, hv); err != nil {
		t.Fatal(err)
	}
	old := resolver.DefaultHosts
	resolver.DefaultHosts = resolver.NewHosts(tr)
	t.Cleanup(func() { resolver.DefaultHosts = old })
}

func TestTCPDialMetadataHandsProxiesTheObservedNameForReverseMappedHosts(t *testing.T) {
	ip := netip.MustParseAddr("140.82.113.21")
	installEnhancer(t, &fakeEnhancer{redirHost: true})

	t.Run("sniffed name differs from the guess", func(t *testing.T) {
		m := &C.Metadata{Host: "guess.example", SniffHost: "real.example", DstIP: ip, DstPort: 443, MappedHost: true}
		out := tcpDialMetadata(m)
		if out == m {
			t.Fatal("the dial copy must not be the original, which logs and the API still read")
		}
		if out.Host != "real.example" || out.DstIP != ip || !out.MappedHost {
			t.Fatalf("got Host %q DstIP %s marked %v; want the sniffed name, the client's IP and the marker kept", out.Host, out.DstIP, out.MappedHost)
		}
		if m.Host != "guess.example" || m.SniffHost != "real.example" {
			t.Fatal("the original metadata must keep both names")
		}
	})

	t.Run("no sniffed name keeps the guess for proxies", func(t *testing.T) {
		m := &C.Metadata{Host: "guess.example", DstIP: ip, DstPort: 443, MappedHost: true}
		if out := tcpDialMetadata(m); out != m || out.Host != "guess.example" {
			t.Fatalf("expected the original metadata unchanged, got %+v", out)
		}
	})

	t.Run("sniffed name equal to the guess changes nothing", func(t *testing.T) {
		m := &C.Metadata{Host: "same.example", SniffHost: "same.example", DstIP: ip, DstPort: 443, MappedHost: true}
		if out := tcpDialMetadata(m); out != m {
			t.Fatal("expected the original metadata unchanged")
		}
	})

	t.Run("client-supplied domain is never rewritten", func(t *testing.T) {
		m := &C.Metadata{Host: "client.example", SniffHost: "real.example", DstPort: 443}
		if out := tcpDialMetadata(m); out != m || out.Host != "client.example" {
			t.Fatalf("expected the client's domain kept, got %+v", out)
		}
	})

	t.Run("hosts entry for the guess still wins", func(t *testing.T) {
		hostsIP := netip.MustParseAddr("192.0.2.7")
		installHosts(t, "guess.example", hostsIP)
		m := &C.Metadata{Host: "guess.example", SniffHost: "real.example", DstIP: ip, DstPort: 443, MappedHost: true}
		out := tcpDialMetadata(m)
		if out.Host != "" || out.DstIP != hostsIP || out.DNSMode != C.DNSHosts {
			t.Fatalf("got Host %q DstIP %s mode %v; want the hosts address with Host stripped", out.Host, out.DstIP, out.DNSMode)
		}
	})
}

func TestLogDestinationShowsTheDialedIPForReverseMappedHosts(t *testing.T) {
	ip := netip.MustParseAddr("140.82.113.21")
	plain := &C.Metadata{Host: "client.example", DstIP: ip, DstPort: 443}
	if got := logDestination(plain); got != "client.example:443" {
		t.Fatalf("unmarked connection: got %q", got)
	}
	marked := &C.Metadata{Host: "guess.example", SniffHost: "real.example", DstIP: ip, DstPort: 443, MappedHost: true}
	if got := logDestination(marked); got != "140.82.113.21:443 (sniffed real.example, mapped guess.example)" {
		t.Fatalf("marked connection: got %q", got)
	}
	marked.SniffHost = ""
	if got := logDestination(marked); got != "140.82.113.21:443 (mapped guess.example)" {
		t.Fatalf("marked connection without a sniff: got %q", got)
	}
}
