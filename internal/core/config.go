package core

import (
	"fmt"
	"os"
	"time"

	"github.com/h44z/wg-portal/internal/authentication"
	"github.com/h44z/wg-portal/internal/persistence"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Core struct {
		LogLevel       string        `yaml:"log_level"`
		StartupTimeout time.Duration ``

		ListeningAddress string `yaml:"listening_address"`
		SessionSecret    string `yaml:"session_secret"`

		ExternalUrl string `yaml:"external_url"`
		Title       string `yaml:"title"`
		CompanyName string `yaml:"company"`

		// AdminUser defines the default administrator account that will be created
		AdminUser     string `yaml:"admin_user"` // must be an email address
		AdminPassword string `yaml:"admin_password"`

		EditableKeys            bool   `yaml:"editable_keys"`
		CreateDefaultPeer       bool   `yaml:"create_default_peer"`
		SelfProvisioningAllowed bool   `yaml:"self_provisioning_allowed"`
		LdapEnabled             bool   `yaml:"ldap_enabled"`
		LogoUrl                 string `yaml:"logo_url"`
	} `yaml:"core"`

	Auth struct {
		OpenIDConnect []authentication.OpenIDConnectProvider `yaml:"oidc"`
		OAuth         []authentication.OAuthProvider         `yaml:"oauth"`
		Ldap          []authentication.LdapProvider          `yaml:"ldap"`
	} `yaml:"auth"`

	Database persistence.DatabaseConfig `yaml:"database"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}

	// default config

	cfg.Core.CompanyName = "WireGuard Portal"
	cfg.Core.StartupTimeout = 60 * time.Second

	cfg.Database = persistence.DatabaseConfig{
		Type: "sqlite",
		DSN:  "sqlite.db",
	}

	// override config values from YAML file

	cfgFileName := "config.yml"
	if envCfgFileName := os.Getenv("WG_PORTAL_CONFIG"); envCfgFileName != "" {
		cfgFileName = envCfgFileName
	}

	if err := loadConfigFile(cfg, cfgFileName); err != nil {
		return nil, fmt.Errorf("failed to load config from yaml: %w", err)
	}

	return cfg, nil
}

func loadConfigFile(cfg interface{}, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return errors.WithMessage(err, "failed to open file")
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(cfg)
	if err != nil {
		return errors.WithMessage(err, "failed to decode config file")
	}

	return nil
}
