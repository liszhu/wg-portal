package app

import (
	"context"
	"fmt"

	"github.com/h44z/wg-portal/internal/authentication"

	"github.com/h44z/wg-portal/internal/adapters"
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
	db, err := adapters.NewDatabase(cfg.Database)
	if err != nil {
		return nil, err
	}
	dbRepo := adapters.NewSqlRepository(db)

	a := &App{
		cfg:   cfg,
		users: dbRepo,
	}

	err = a.setup(context.Background())
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (a *App) setup(ctx context.Context) error {
	if err := a.setupExternalAuthProviders(ctx); err != nil {
		return fmt.Errorf("external authentication provider error: %w", err)
	}

	return nil
}
