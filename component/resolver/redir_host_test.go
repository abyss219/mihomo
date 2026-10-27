package resolver

import (
	"net/netip"
	"testing"
)

type modeOnlyEnhancer struct {
	redirHost bool
}

func (m modeOnlyEnhancer) FakeIPEnabled() bool                    { return false }
func (m modeOnlyEnhancer) MappingEnabled() bool                   { return m.redirHost }
func (m modeOnlyEnhancer) RedirHostEnabled() bool                 { return m.redirHost }
func (m modeOnlyEnhancer) IsFakeIP(netip.Addr) bool               { return false }
func (m modeOnlyEnhancer) IsFakeBroadcastIP(netip.Addr) bool      { return false }
func (m modeOnlyEnhancer) IsExistFakeIP(netip.Addr) bool          { return false }
func (m modeOnlyEnhancer) FindHostByIP(netip.Addr) (string, bool) { return "", false }
func (m modeOnlyEnhancer) FlushFakeIP() error                     { return nil }
func (m modeOnlyEnhancer) InsertHostByIP(netip.Addr, string)      {}
func (m modeOnlyEnhancer) StoreFakePoolState()                    {}

func TestRedirHostEnabledFollowsTheDefaultMapper(t *testing.T) {
	old := DefaultHostMapper
	t.Cleanup(func() { DefaultHostMapper = old })

	DefaultHostMapper = nil
	if RedirHostEnabled() {
		t.Fatal("no mapper must mean redir-host is off")
	}

	DefaultHostMapper = modeOnlyEnhancer{redirHost: true}
	if !RedirHostEnabled() {
		t.Fatal("redir-host mapper must report redir-host on")
	}

	DefaultHostMapper = modeOnlyEnhancer{redirHost: false}
	if RedirHostEnabled() {
		t.Fatal("non-redir-host mapper must report redir-host off")
	}
}
