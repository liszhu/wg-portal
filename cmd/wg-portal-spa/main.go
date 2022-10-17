package main

import (
	"context"
	"syscall"

	"github.com/h44z/wg-portal/internal/app"
	"github.com/h44z/wg-portal/internal/config"

	"github.com/h44z/wg-portal/internal"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx := internal.SignalAwareContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	logrus.Infof("Starting web portal...")

	cfg, err := LoadConfig()
	internal.AssertNoError(err)

	backend, err := app.New(cfg.Backend)
	internal.AssertNoError(err)

	webService, err := NewServer(cfg, backend)
	if err != nil {
		panic(err)
	}

	webService.Run(ctx)

	// wait until context gets cancelled
	<-ctx.Done()

	logrus.Infof("Stopped web portal")

}

func LoadConfig() (*Config, error) {
	backendCfg, err := config.GetConfig()
	internal.AssertNoError(err)

	cfg := &Config{
		Backend: backendCfg,
	}

	// default config

	cfg.Frontend.ListeningAddress = ":5000"
	cfg.Frontend.SessionSecret = "wgPortalSession"
	cfg.Frontend.GinDebug = true

	cfg.Backend.Web.ExternalUrl = "http://localhost:5000" // enable if running frontend via npm run dev

	/*cfgFileName := "config.yml"
	if envCfgFileName := os.Getenv("WG_PORTAL_CONFIG"); envCfgFileName != "" {
		cfgFileName = envCfgFileName
	}

	if err := loadConfigFile(cfg, cfgFileName); err != nil {
		return nil, fmt.Errorf("failed to load config from yaml: %w", err)
	}*/

	return cfg, nil
}
