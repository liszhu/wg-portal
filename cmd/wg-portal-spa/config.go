package main

import "github.com/h44z/wg-portal/internal/core"

type Config struct {
	Backend core.Config `yaml:"backend"`

	Frontend FrontendConfig `yaml:"frontend"`
}

type FrontendConfig struct {
	ListeningAddress string `yaml:"listening_address"`
	SessionSecret    string `yaml:"session_secret"`

	ExternalUrl string `yaml:"external_url"`

	GinDebug bool `yaml:"gin_debug"`
}
