package config

import (
	"github.com/abyss219/mihomo/component/auth"
	"github.com/abyss219/mihomo/listener/reality"
)

// AuthServer for http/socks/mixed server
type AuthServer struct {
	Enable        bool
	Listen        string
	AuthStore     auth.AuthStore
	Certificate   string
	PrivateKey    string
	EchKey        string
	RealityConfig reality.Config
}
