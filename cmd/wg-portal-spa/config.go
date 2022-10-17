package main

import (
	"github.com/h44z/wg-portal/internal/config"
)

type Config struct {
	Backend *config.Config `yaml:"backend"`

	Frontend FrontendConfig `yaml:"frontend"`
}

type FrontendConfig struct {
	ListeningAddress string `yaml:"listening_address"`
	SessionSecret    string `yaml:"session_secret"`

	GinDebug bool `yaml:"gin_debug"`
}
