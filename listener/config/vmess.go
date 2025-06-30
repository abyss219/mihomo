package config

import (
	"encoding/json"

	"github.com/abyss219/mihomo/listener/reality"
	"github.com/abyss219/mihomo/listener/sing"
)

type VmessUser struct {
	Username string
	UUID     string
	AlterID  int
}

type VmessServer struct {
	Enable          bool
	Listen          string
	Users           []VmessUser
	WsPath          string
	GrpcServiceName string
	Certificate     string
	PrivateKey      string
	EchKey          string
	RealityConfig   reality.Config
	MuxOption       sing.MuxOption `yaml:"mux-option" json:"mux-option,omitempty"`
}

func (t VmessServer) String() string {
	b, _ := json.Marshal(t)
	return string(b)
}
