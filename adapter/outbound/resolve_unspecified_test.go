package outbound

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"github.com/metacubex/mihomo/component/dialer"
	"github.com/metacubex/mihomo/component/resolver"
	C "github.com/metacubex/mihomo/constant"
)

// sinkholeResolver answers every name with the unspecified address, the way
// an ad-blocking DNS does for a blocked name.
type sinkholeResolver struct {
	literalOnlyResolver
}

func (sinkholeResolver) LookupIP(context.Context, string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("0.0.0.0")}, nil
}

func (sinkholeResolver) LookupIPv4(context.Context, string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("0.0.0.0")}, nil
}

func (sinkholeResolver) LookupIPv6(context.Context, string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("::")}, nil
}

func TestResolveIPWithResolverRefusesSinkholedAnswers(t *testing.T) {
	// IPv6 lookups are refused outright while IPv6 is disabled, which is the
	// default in a bare process; enable it so the IPv6 cases reach the check.
	oldDisableIPv6 := resolver.DisableIPv6
	resolver.DisableIPv6 = false
	t.Cleanup(func() { resolver.DisableIPv6 = oldDisableIPv6 })

	ctx := context.Background()
	for _, prefer := range []C.DNSPrefer{C.DualStack, C.IPv4Only, C.IPv6Only, C.IPv6Prefer} {
		_, err := resolveIPWithResolver(ctx, "blocked.example", prefer, sinkholeResolver{})
		if !errors.Is(err, dialer.ErrUnspecifiedDestination) {
			t.Fatalf("prefer %v: got %v, want ErrUnspecifiedDestination", prefer, err)
		}
	}

	ip, err := resolveIPWithResolver(ctx, "127.0.0.1", C.DualStack, literalOnlyResolver{})
	if err != nil || ip != netip.MustParseAddr("127.0.0.1") {
		t.Fatalf("a real address must still resolve, got %s %v", ip, err)
	}
}

func TestDirectResolveUDPRefusesSinkholedHost(t *testing.T) {
	oldResolver := resolver.DirectHostResolver
	resolver.DirectHostResolver = sinkholeResolver{}
	t.Cleanup(func() { resolver.DirectHostResolver = oldResolver })

	d := NewDirect()
	m := &C.Metadata{NetWork: C.UDP, Host: "blocked.example", DstPort: 443}

	err := d.ResolveUDP(context.Background(), m)
	if !errors.Is(err, dialer.ErrUnspecifiedDestination) {
		t.Fatalf("got %v, want ErrUnspecifiedDestination", err)
	}
	if m.DstIP.IsValid() {
		t.Fatalf("DstIP must stay unset, got %s", m.DstIP)
	}
}
