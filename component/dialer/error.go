package dialer

import (
	"errors"
)

var (
	ErrorNoIpAddress           = errors.New("no ip address")
	ErrorInvalidedNetworkStack = errors.New("invalided network stack")

	// ErrUnspecifiedDestination is returned instead of dialing 0.0.0.0 or ::.
	// Such a destination only comes from a sinkholed DNS answer or a
	// misconfiguration, and dialing it would reach whatever listens locally
	// on that port.
	ErrUnspecifiedDestination = errors.New("refusing to dial unspecified destination address")
)
