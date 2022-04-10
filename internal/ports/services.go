package ports

import (
	"context"

	"github.com/h44z/wg-portal/internal/domain"
)

// Authenticator service handles authentication use-cases.
type Authenticator interface {
	IsAuthenticated(ctx context.Context) bool
	AuthenticationInfo(ctx context.Context) *domain.ContextAuthenticationInfo
	DestroyLogin(ctx context.Context) context.Context

	GetAuthenticators() []domain.AuthenticatorConfig
	GetAuthenticator(authenticator domain.AuthenticatorId) (domain.AuthenticatorConfig, error)

	// AuthenticateContext can be used to perform a password authentication flow
	AuthenticateContext(ctx context.Context, authenticator domain.AuthenticatorId, username, password string) (context.Context, error)

	// GetOauthUrl can be used to start the oauth authentication flow
	GetOauthUrl(authenticator domain.AuthenticatorId) (url, state, nonce string, err error)
	AuthenticateContextWithCode(ctx context.Context, authenticator domain.AuthenticatorId, code, state, nonce string) (context.Context, error)

	GetUserInfo(ctx context.Context) (*domain.AuthenticatorUserInfo, error)
	GetAllUserInfos(ctx context.Context) ([]*domain.AuthenticatorUserInfo, error)
}

// UserManager service handles user related use-cases.
type UserManager interface {
}
