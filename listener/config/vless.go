package config

import (
	"encoding/json"

	"github.com/abyss219/mihomo/listener/reality"
	"github.com/abyss219/mihomo/listener/sing"
)

type VlessUser struct {
	Username string
	UUID     string
	Flow     string
}

type VlessServer struct {
	Enable          bool
	Listen          string
	Users           []VlessUser
	WsPath          string
	GrpcServiceName string
	Certificate     string
	PrivateKey      string
	EchKey          string
	RealityConfig   reality.Config
	MuxOption       sing.MuxOption `yaml:"mux-option" json:"mux-option,omitempty"`
}

func (t VlessServer) String() string {
	b, _ := json.Marshal(t)
	return string(b)
}
