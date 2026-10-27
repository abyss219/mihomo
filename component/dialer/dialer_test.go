package dialer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strconv"
	"testing"
	"time"

	"github.com/metacubex/mihomo/component/resolver"
)

// spyNetDialer records every address the dialer tried to open a socket to.
type spyNetDialer struct {
	calls []string
}

func (s *spyNetDialer) DialContext(_ context.Context, _ string, address string) (net.Conn, error) {
	s.calls = append(s.calls, address)
	return nil, errors.New("spy: dial attempted")
}

func testCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// listenLoopback starts an accept-and-close listener and returns its address.
func listenLoopback(t *testing.T, network, address string) (netip.Addr, string) {
	t.Helper()
	ln, err := net.Listen(network, address)
	if err != nil {
		t.Skipf("cannot listen on %s %s: %v", network, address, err)
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
	return ap.Addr(), strconv.Itoa(int(ap.Port()))
}

// closedLoopbackPort returns a loopback port that nothing listens on.
func closedLoopbackPort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot listen on loopback: %v", err)
	}
	port := strconv.Itoa(int(netip.MustParseAddrPort(ln.Addr().String()).Port()))
	_ = ln.Close()
	return port
}

func TestDialContextRefusesUnspecifiedDestination(t *testing.T) {
	for _, tc := range []struct {
		name string
		ip   string
	}{
		{"ipv4 zero", "0.0.0.0"},
		{"ipv6 zero", "::"},
		{"ipv4-mapped zero", "::ffff:0.0.0.0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spy := &spyNetDialer{}
			_, err := dialContext(testCtx(t), "tcp", netip.MustParseAddr(tc.ip), "443", applyOptions(WithNetDialer(spy)))
			if !errors.Is(err, ErrUnspecifiedDestination) {
				t.Fatalf("got %v, want ErrUnspecifiedDestination", err)
			}
			if len(spy.calls) != 0 {
				t.Fatalf("net dialer was called with %v; the guard must run before any socket is created", spy.calls)
			}
		})
	}
}

func TestDialContextDoesNotRefuseNonZeroAddresses(t *testing.T) {
	for _, ip := range []string{"127.0.0.1", "::1", "192.168.0.1", "140.82.113.21"} {
		t.Run(ip, func(t *testing.T) {
			spy := &spyNetDialer{}
			_, err := dialContext(testCtx(t), "tcp", netip.MustParseAddr(ip), "443", applyOptions(WithNetDialer(spy)))
			if errors.Is(err, ErrUnspecifiedDestination) {
				t.Fatalf("%s must not be refused", ip)
			}
			if len(spy.calls) != 1 {
				t.Fatalf("expected one dial attempt, got %v", spy.calls)
			}
		})
	}
}

func TestDialContextRefusesZeroProducedByIP4P(t *testing.T) {
	// A 2001:0000::/32 address carries an IPv4 address and port in its last
	// six bytes. This one converts to 0.0.0.0:443 when IP4P is enabled.
	ip4p := netip.MustParseAddr("2001::1bb:0:0")
	old := resolver.GetIP4PEnable()
	t.Cleanup(func() { resolver.SetIP4PEnable(old) })

	resolver.SetIP4PEnable(true)
	spy := &spyNetDialer{}
	_, err := dialContext(testCtx(t), "tcp", ip4p, "1", applyOptions(WithNetDialer(spy)))
	if !errors.Is(err, ErrUnspecifiedDestination) {
		t.Fatalf("got %v, want ErrUnspecifiedDestination after IP4P conversion", err)
	}
	if len(spy.calls) != 0 {
		t.Fatalf("net dialer was called with %v", spy.calls)
	}

	resolver.SetIP4PEnable(false)
	spy = &spyNetDialer{}
	_, err = dialContext(testCtx(t), "tcp", ip4p, "1", applyOptions(WithNetDialer(spy)))
	if errors.Is(err, ErrUnspecifiedDestination) {
		t.Fatalf("an unconverted 2001:: address must not be refused")
	}
	if len(spy.calls) != 1 {
		t.Fatalf("expected one dial attempt, got %v", spy.calls)
	}
}

func TestDialContextRefusesUnspecifiedBeforeTFO(t *testing.T) {
	_, err := dialContext(testCtx(t), "tcp", netip.MustParseAddr("0.0.0.0"), "443", applyOptions(WithTFO(true)))
	if !errors.Is(err, ErrUnspecifiedDestination) {
		t.Fatalf("got %v, want ErrUnspecifiedDestination with TFO enabled", err)
	}
}

func TestMixedCandidatesStillDialTheUsableAddress(t *testing.T) {
	ip, port := listenLoopback(t, "tcp4", "127.0.0.1:0")
	ips := []netip.Addr{netip.MustParseAddr("0.0.0.0"), ip}
	for name, dial := range map[string]dialFunc{"serial": serialDialContext, "parallel": parallelDialContext} {
		t.Run(name, func(t *testing.T) {
			conn, err := dial(testCtx(t), "tcp4", ips, port, applyOptions())
			if err != nil {
				t.Fatalf("usable candidate was not dialed: %v", err)
			}
			_ = conn.Close()
		})
	}
}

func TestMixedCandidatesFailureIsNotTheRefusal(t *testing.T) {
	port := closedLoopbackPort(t)
	ips := []netip.Addr{netip.MustParseAddr("0.0.0.0"), netip.MustParseAddr("127.0.0.1")}
	for name, dial := range map[string]dialFunc{"serial": serialDialContext, "parallel": parallelDialContext} {
		t.Run(name, func(t *testing.T) {
			_, err := dial(testCtx(t), "tcp4", ips, port, applyOptions())
			if err == nil {
				t.Fatal("expected the closed port to fail")
			}
			if errors.Is(err, ErrUnspecifiedDestination) {
				t.Fatalf("a mixed candidate set must not report the refusal, or the tunnel would stop retrying: %v", err)
			}
		})
	}
}

func TestAllZeroCandidatesFailFastWithTheRefusal(t *testing.T) {
	ips := []netip.Addr{netip.MustParseAddr("0.0.0.0"), netip.MustParseAddr("::")}
	for name, dial := range map[string]dialFunc{"serial": serialDialContext, "parallel": parallelDialContext} {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			_, err := dial(testCtx(t), "tcp", ips, "443", applyOptions())
			if !errors.Is(err, ErrUnspecifiedDestination) {
				t.Fatalf("got %v, want ErrUnspecifiedDestination", err)
			}
			if elapsed := time.Since(start); elapsed > time.Second {
				t.Fatalf("refusal took %s; it must not wait on any socket", elapsed)
			}
		})
	}
}

func TestDualStackZeroIPv4StillDialsIPv6(t *testing.T) {
	ip6, port := listenLoopback(t, "tcp6", "[::1]:0")
	ips := []netip.Addr{netip.MustParseAddr("0.0.0.0"), ip6}
	conn, err := dualStackDialContext(testCtx(t), serialDialContext, "tcp", ips, port, applyOptions())
	if err != nil {
		t.Fatalf("IPv6 candidate was not dialed: %v", err)
	}
	_ = conn.Close()
}

func TestDualStackAllZeroReportsTheRefusal(t *testing.T) {
	ips := []netip.Addr{netip.MustParseAddr("0.0.0.0"), netip.MustParseAddr("::")}
	_, err := dualStackDialContext(testCtx(t), serialDialContext, "tcp", ips, "443", applyOptions())
	if !errors.Is(err, ErrUnspecifiedDestination) {
		t.Fatalf("got %v, want ErrUnspecifiedDestination", err)
	}
}

func TestJoinDialErrors(t *testing.T) {
	refusal := fmt.Errorf("%w: 0.0.0.0:443", ErrUnspecifiedDestination)

	mixed := joinDialErrors([]error{refusal, io.EOF})
	if errors.Is(mixed, ErrUnspecifiedDestination) {
		t.Fatalf("refusal must be dropped when another candidate failed: %v", mixed)
	}
	if !errors.Is(mixed, io.EOF) {
		t.Fatalf("the other candidate's error must be kept: %v", mixed)
	}

	only := joinDialErrors([]error{refusal, refusal})
	if !errors.Is(only, ErrUnspecifiedDestination) {
		t.Fatalf("an all-refused set must report the refusal: %v", only)
	}
}
