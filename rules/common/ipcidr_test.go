package common

import (
	"net/netip"
	"testing"

	C "github.com/metacubex/mihomo/constant"
)

func TestIPCIDRMatchSniffedSinkhole(t *testing.T) {
	sinkhole := netip.MustParseAddr("0.0.0.0")
	publicIP := netip.MustParseAddr("93.184.216.34")

	// a sniffed host sinkholed by an adblocking DNS should match
	rule, err := NewIPCIDR("0.0.0.0/32", "REJECT")
	if err != nil {
		t.Fatal(err)
	}
	metadata := &C.Metadata{DstIP: publicIP, SniffHost: "ads.example.com", SniffDstIP: sinkhole}
	if matched, _ := rule.Match(metadata, C.RuleMatchHelper{}); !matched {
		t.Error("sniffed sinkhole address should match")
	}

	// an ipv6 sinkhole should match its ipv6 rule
	rule6, err := NewIPCIDR("::/128", "REJECT")
	if err != nil {
		t.Fatal(err)
	}
	metadata = &C.Metadata{DstIP: publicIP, SniffHost: "ads.example.com", SniffDstIP: netip.MustParseAddr("::")}
	if matched, _ := rule6.Match(metadata, C.RuleMatchHelper{}); !matched {
		t.Error("sniffed ipv6 sinkhole address should match")
	}

	// a source rule must never match against the sniffed destination address
	srcRule, err := NewIPCIDR("0.0.0.0/32", "REJECT", WithIPCIDRSourceIP(true))
	if err != nil {
		t.Fatal(err)
	}
	metadata = &C.Metadata{SrcIP: netip.MustParseAddr("192.168.1.50"), DstIP: publicIP, SniffDstIP: sinkhole}
	if matched, _ := srcRule.Match(metadata, C.RuleMatchHelper{}); matched {
		t.Error("source rule should ignore SniffDstIP")
	}

	// the real destination address must still match as before
	rule, err = NewIPCIDR("93.184.216.0/24", "DIRECT")
	if err != nil {
		t.Fatal(err)
	}
	metadata = &C.Metadata{DstIP: publicIP, SniffDstIP: sinkhole}
	if matched, _ := rule.Match(metadata, C.RuleMatchHelper{}); !matched {
		t.Error("destination address should still match")
	}
}
