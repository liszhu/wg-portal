package app

import (
	"context"

	"github.com/h44z/wg-portal/internal/model"
)

// TODO: remove file
type appInterface interface {
	GetExternalLoginProviders(_ context.Context) []model.LoginProviderInfo
	PlainLogin(ctx context.Context, username, password string) (*model.User, error)
	OauthLoginStep1(_ context.Context, providerId string) (authCodeUrl, state, nonce string, err error)
	OauthLoginStep2(ctx context.Context, providerId, nonce, code string) (*model.User, error)
}
