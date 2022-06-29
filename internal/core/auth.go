package core

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/h44z/wg-portal/internal/authentication"

	"github.com/h44z/wg-portal/internal/model"
	"github.com/sirupsen/logrus"
)

func (w *wgPortal) GetExternalLoginProviders(ctx context.Context) []model.LoginProviderInfo {
	authProviders := make([]model.LoginProviderInfo, 0, len(w.cfg.Auth.OAuth)+len(w.cfg.Auth.OpenIDConnect))
	for _, provider := range w.cfg.Auth.OpenIDConnect {
		providerId := strings.ToLower(provider.ProviderName)
		providerName := provider.DisplayName
		if providerName == "" {
			providerName = provider.ProviderName
		}
		authProviders = append(authProviders, model.LoginProviderInfo{
			ID:          providerId,
			Name:        providerName,
			ProviderUrl: fmt.Sprintf("/auth/login/%s/init", providerId),
			CallbackUrl: fmt.Sprintf("/auth/login/%s/callback", providerId),
		})
	}
	for _, provider := range w.cfg.Auth.OAuth {
		providerId := strings.ToLower(provider.ProviderName)
		providerName := provider.DisplayName
		if providerName == "" {
			providerName = provider.ProviderName
		}
		authProviders = append(authProviders, model.LoginProviderInfo{
			ID:   providerId,
			Name: providerName,
		})
	}

	return authProviders
}

func (w *wgPortal) PlainLogin(ctx context.Context, username, password string) (*model.User, error) {
	// Validate form input
	if strings.Trim(username, " ") == "" || strings.Trim(password, " ") == "" {
		return nil, fmt.Errorf("missing username or password")
	}

	user, err := w.passwordAuthentication(ctx, model.UserIdentifier(username), password)
	if err != nil {
		logrus.Tracef("invalid login attempt for username %s: %v", username, err)
		return nil, fmt.Errorf("login failed")
	}

	return user, nil
}

func (w *wgPortal) passwordAuthentication(ctx context.Context, identifier model.UserIdentifier, password string) (*model.User, error) {

	var ldapUserInfo *authentication.AuthenticatorUserInfo
	var ldapProvider authentication.RegistrationAuthenticator

	var userInDatabase = false
	var userSource model.UserSource
	existingUser, err := w.users.GetUser(identifier)
	if err == nil {
		userInDatabase = true
		userSource = model.UserSourceDatabase
	} else {
		// search user in ldap if registration is enabled
		for _, authenticator := range w.ldapAuthenticators {
			if !authenticator.RegistrationEnabled() {
				continue
			}
			rawUserInfo, err := authenticator.GetUserInfo(context.Background(), identifier)
			if err != nil {
				continue
			}
			ldapUserInfo, err = authenticator.ParseUserInfo(rawUserInfo)
			if err != nil {
				continue
			}

			// ldap user found
			userSource = model.UserSourceLdap
			ldapProvider = authenticator

			break
		}
	}

	if userSource == "" {
		return nil, errors.New("user not found")
	}

	switch userSource {
	case model.UserSourceDatabase:
		err = w.users.PlaintextAuthentication(identifier, password)
	case model.UserSourceLdap:
		for _, authenticator := range w.ldapAuthenticators {
			err = authenticator.PlaintextAuthentication(identifier, password)
			if err == nil {
				break // auth succeeded
			}
		}
	default:
		err = errors.New("no authentication backend available")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	if !userInDatabase {
		user, err := w.processUserInfo(ctx, ldapUserInfo, model.UserSourceLdap, ldapProvider)
		if err != nil {
			return nil, fmt.Errorf("unable to process user information: %w", err)
		}
		return user, nil
	} else {
		return existingUser, nil
	}
}

func (w *wgPortal) OauthLoginStep1(ctx context.Context, providerId string) (authCodeUrl, state, nonce string, err error) {
	if _, ok := w.oauthAuthenticators[providerId]; !ok {
		return "", "", "", fmt.Errorf("no oauth provider for id %s found", providerId)
	}

	// Prepare authentication flow, set state cookies
	state, err = w.randString(16)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate state: %w", err)
	}

	authenticator := w.oauthAuthenticators[providerId]

	switch authenticator.GetType() {
	case authentication.AuthenticatorTypeOAuth:
		authCodeUrl = authenticator.AuthCodeURL(state)
	case authentication.AuthenticatorTypeOidc:
		nonce, err = w.randString(16)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to generate nonce: %w", err)
		}

		authCodeUrl = authenticator.AuthCodeURL(state, oidc.Nonce(nonce))
	}

	return
}

func (w *wgPortal) randString(nByte int) (string, error) {
	b := make([]byte, nByte)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (w *wgPortal) OauthLoginStep2(ctx context.Context, providerId, nonce, code string) (*model.User, error) {
	if _, ok := w.oauthAuthenticators[providerId]; !ok {
		return nil, fmt.Errorf("no oauth provider for id %s found", providerId)
	}

	authenticator := w.oauthAuthenticators[providerId]
	oauth2Token, err := authenticator.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("unable to exchange code: %w", err)
	}

	rawUserInfo, err := authenticator.GetUserInfo(ctx, oauth2Token, nonce)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch user information: %w", err)
	}

	userInfo, err := authenticator.ParseUserInfo(rawUserInfo)
	if err != nil {
		return nil, fmt.Errorf("unable to parse user information: %w", err)
	}

	user, err := w.processUserInfo(ctx, userInfo, model.UserSourceOauth, authenticator)
	if err != nil {
		return nil, fmt.Errorf("unable to process user information: %w", err)
	}

	return user, nil
}

func (w *wgPortal) processUserInfo(ctx context.Context, userInfo *authentication.AuthenticatorUserInfo, source model.UserSource, provider authentication.RegistrationAuthenticator) (*model.User, error) {
	registrationEnabled := provider.RegistrationEnabled()

	// Search user in backend
	user, err := w.users.GetUser(userInfo.Identifier)
	switch {
	case err != nil && registrationEnabled:
		user, err = w.registerNewUser(ctx, userInfo, source)
		if err != nil {
			return nil, fmt.Errorf("failed to register user: %w", err)
		}
	case err != nil:
		return nil, fmt.Errorf("registration disabled, cannot create missing user: %w", err)
	}

	return user, nil
}

func (w *wgPortal) registerNewUser(ctx context.Context, userInfo *authentication.AuthenticatorUserInfo, source model.UserSource) (*model.User, error) {
	user := &model.User{
		Identifier: userInfo.Identifier,
		Email:      userInfo.Email,
		Source:     source,
		IsAdmin:    userInfo.IsAdmin,
		Firstname:  userInfo.Firstname,
		Lastname:   userInfo.Lastname,
		Phone:      userInfo.Phone,
		Department: userInfo.Department,
	}

	var err error
	createOptions := UserCreateOptions().WithDefaultPeer(w.cfg.Core.CreateDefaultPeer, w.cfg.DefaultPeerInterfaces...)
	if user, err = w.CreateUser(ctx, user, createOptions); err != nil {
		return nil, fmt.Errorf("failed to create new user: %w", err)
	}

	return user, nil
}

func (w *wgPortal) getAuthenticatorConfig(id string) (interface{}, error) {
	for i := range w.cfg.Auth.OpenIDConnect {
		if w.cfg.Auth.OpenIDConnect[i].ProviderName == id {
			return w.cfg.Auth.OpenIDConnect[i], nil
		}
	}

	for i := range w.cfg.Auth.OAuth {
		if w.cfg.Auth.OAuth[i].ProviderName == id {
			return w.cfg.Auth.OAuth[i], nil
		}
	}

	return nil, fmt.Errorf("no configuration for authenticator id %s", id)
}
