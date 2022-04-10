package config

import (
	"fmt"
	"os"

	"github.com/h44z/wg-portal/internal/domain"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Auth struct {
		OpenIDConnect []domain.OpenIDConnectProvider `yaml:"oidc"`
		OAuth         []domain.OAuthProvider         `yaml:"oauth"`
		Ldap          []domain.LdapProvider          `yaml:"ldap"`
	} `yaml:"auth"`
}

func Load() (*Config, error) {
	cfg := &Config{}

	err := loadYaml(&cfg, "config.yml")
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func loadYaml(cfg interface{}, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(cfg)
	if err != nil {
		return fmt.Errorf("failed to decode config file: %w", err)
	}

	return nil
}
