package dialer

import (
	"context"
	"net"
	"syscall"

	"github.com/abyss219/mihomo/common/sockopt"
)

func addrReuseToListenConfig(lc *net.ListenConfig) {
	addControlToListenConfig(lc, func(ctx context.Context, network, address string, c syscall.RawConn) error {
		return sockopt.RawConnReuseaddr(c)
	})
}
