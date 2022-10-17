package app

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/h44z/wg-portal/internal/authentication"

	"github.com/h44z/wg-portal/internal/adapters"
	"github.com/h44z/wg-portal/internal/common"
	"github.com/h44z/wg-portal/internal/config"
	"github.com/h44z/wg-portal/internal/model"
)

type userRepo interface {
	GetUser(ctx context.Context, id model.UserIdentifier) (*model.User, error)
	GetAllUsers(ctx context.Context) ([]model.User, error)
	FindUsers(ctx context.Context, search string) ([]model.User, error)
	SaveUser(ctx context.Context, id model.UserIdentifier, updateFunc func(u *model.User) (*model.User, error)) error
	DeleteUser(ctx context.Context, id model.UserIdentifier) error
}

type App struct {
	cfg   *config.Config
	users userRepo

	oauthAuthenticators map[string]authentication.OauthAuthenticator
	ldapAuthenticators  map[string]authentication.LdapAuthenticator
}

func New(cfg *config.Config) (*App, error) {
	db, err := common.NewDatabase(cfg.Database)
	if err != nil {
		return nil, err
	}
	dbRepo := adapters.NewSqlRepository(db)

	return &App{
		cfg:   cfg,
		users: dbRepo,
	}, nil
}

func (a *App) setup(ctx context.Context) error {
	if err := a.setupExternalAuthProviders(ctx); err != nil {
		return fmt.Errorf("external authentication provider error: %w", err)
	}

	return nil
}

func (a *App) setupExternalAuthProviders(ctx context.Context) error {
	extUrl, err := url.Parse(a.cfg.Web.ExternalUrl)
	if err != nil {
		return fmt.Errorf("failed to parse external url: %w", err)
	}

	for i := range a.cfg.Auth.OpenIDConnect {
		providerCfg := &a.cfg.Auth.OpenIDConnect[i]
		providerId := strings.ToLower(providerCfg.ProviderName)

		if _, exists := a.oauthAuthenticators[providerId]; exists {
			return fmt.Errorf("auth provider with name %s is already registerd", providerId)
		}

		redirectUrl := *extUrl
		redirectUrl.Path = path.Join(redirectUrl.Path, "/auth/login/", providerId, "/callback")

		authenticator, err := authentication.NewOidcAuthenticator(ctx, redirectUrl.String(), providerCfg)
		if err != nil {
			return fmt.Errorf("failed to setup oidc authentication provider %s: %w", providerCfg.ProviderName, err)
		}
		a.oauthAuthenticators[providerId] = authenticator
	}
	for i := range a.cfg.Auth.OAuth {
		providerCfg := &a.cfg.Auth.OAuth[i]
		providerId := strings.ToLower(providerCfg.ProviderName)

		if _, exists := a.oauthAuthenticators[providerId]; exists {
			return fmt.Errorf("auth provider with name %s is already registerd", providerId)
		}

		redirectUrl := *extUrl
		redirectUrl.Path = path.Join(redirectUrl.Path, "/auth/login/", providerId, "/callback")

		authenticator, err := authentication.NewPlainOauthAuthenticator(ctx, redirectUrl.String(), providerCfg)
		if err != nil {
			return fmt.Errorf("failed to setup oauth authentication provider %s: %w", providerId, err)
		}
		a.oauthAuthenticators[providerId] = authenticator
	}
	for i := range a.cfg.Auth.Ldap {
		providerCfg := &a.cfg.Auth.Ldap[i]
		providerId := strings.ToLower(providerCfg.URL)

		if _, exists := a.ldapAuthenticators[providerId]; exists {
			return fmt.Errorf("auth provider with name %s is already registerd", providerId)
		}

		authenticator, err := authentication.NewLdapAuthenticator(ctx, providerCfg)
		if err != nil {
			return fmt.Errorf("failed to setup ldap authentication provider %s: %w", providerId, err)
		}
		a.ldapAuthenticators[providerId] = authenticator
	}

	return nil
}
