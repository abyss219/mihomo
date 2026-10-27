package outbound

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/metacubex/mihomo/component/resolver"
	C "github.com/metacubex/mihomo/constant"

	"github.com/miekg/dns"
)

var errNoResolve = errors.New("test resolver: names must not be resolved")

// literalOnlyResolver stands in for the DIRECT resolver. Like the real one it
// answers an IP literal with itself, but it refuses every name, so a dial can
// only succeed if no name was resolved.
type literalOnlyResolver struct{}

func (literalOnlyResolver) lookup(host string) ([]netip.Addr, error) {
	if ip, err := netip.ParseAddr(host); err == nil {
		return []netip.Addr{ip}, nil
	}
	return nil, errNoResolve
}

func (r literalOnlyResolver) LookupIP(_ context.Context, host string) ([]netip.Addr, error) {
	return r.lookup(host)
}

func (r literalOnlyResolver) LookupIPv4(_ context.Context, host string) ([]netip.Addr, error) {
	return r.lookup(host)
}

func (r literalOnlyResolver) LookupIPv6(_ context.Context, host string) ([]netip.Addr, error) {
	return r.lookup(host)
}

func (literalOnlyResolver) ResolveECH(context.Context, string) ([]byte, error) {
	return nil, errNoResolve
}

func (literalOnlyResolver) ExchangeContext(context.Context, *dns.Msg) (*dns.Msg, error) {
	return nil, errNoResolve
}

func (literalOnlyResolver) Invalid() bool    { return true }
func (literalOnlyResolver) ClearCache()      {}
func (literalOnlyResolver) ResetConnection() {}

func TestDirectDialsTheClientIPForReverseMappedHosts(t *testing.T) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot listen on loopback: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	ap := netip.MustParseAddrPort(ln.Addr().String())

	oldResolver := resolver.DirectHostResolver
	resolver.DirectHostResolver = literalOnlyResolver{}
	t.Cleanup(func() { resolver.DirectHostResolver = oldResolver })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	d := NewDirect()

	marked := &C.Metadata{NetWork: C.TCP, Host: "guess.example", DstIP: ap.Addr(), DstPort: ap.Port(), MappedHost: true}
	conn, err := d.DialContext(ctx, marked)
	if err != nil {
		t.Fatalf("a reverse-mapped connection must dial the client's IP without resolving the guess: %v", err)
	}
	_ = conn.Close()

	unmarked := &C.Metadata{NetWork: C.TCP, Host: "guess.example", DstIP: ap.Addr(), DstPort: ap.Port()}
	if conn, err := d.DialContext(ctx, unmarked); err == nil {
		_ = conn.Close()
		t.Fatal("without the marker DIRECT must keep dialing by Host, as upstream does")
	} else if !errors.Is(err, errNoResolve) {
		t.Fatalf("expected the Host to be resolved, got: %v", err)
	}
}
