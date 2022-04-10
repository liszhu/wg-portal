package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/h44z/wg-portal/internal/domain"
	"golang.org/x/oauth2"
)

type oidcAuthenticator struct {
	callbackBaseUrl string

	oauthCfg *oauth2.Config
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier

	cfg    *domain.OpenIDConnectProvider
	client *http.Client
}

func NewOidcAuthenticator(baseUrl string, cfg *domain.OpenIDConnectProvider) (*oidcAuthenticator, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	provider, err := oidc.NewProvider(ctx, cfg.BaseUrl)
	if err != nil {
		return nil, fmt.Errorf("oidc provider error: %w", err)
	}

	scopes := []string{oidc.ScopeOpenID}
	scopes = append(scopes, cfg.ExtraScopes...)

	authenticator := &oidcAuthenticator{
		cfg:             cfg,
		callbackBaseUrl: baseUrl,
		client: &http.Client{
			Timeout: time.Second * 10,
		},
		oauthCfg: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint:     provider.Endpoint(),
			Scopes:       scopes,
		},
		provider: provider,
		verifier: provider.Verifier(&oidc.Config{
			ClientID: cfg.ClientID,
		}),
	}

	authenticator.oauthCfg.RedirectURL, err = authenticator.GetCallbackUrl()
	if err != nil {
		return nil, fmt.Errorf("callback url error: %w", err)
	}

	authenticator.cfg.InitiateFieldMap()

	return authenticator, nil
}

func (p oidcAuthenticator) Config() domain.AuthenticatorConfig {
	return p.cfg
}

func (p oidcAuthenticator) GetCallbackUrl() (string, error) {
	callbackUrl, err := url.Parse(p.callbackBaseUrl)
	if err != nil {
		return "", err
	}
	callbackUrl.Path = path.Join(callbackUrl.Path, "oidc", p.cfg.ProviderName, "callback")
	return callbackUrl.String(), nil
}

func (p oidcAuthenticator) Exchange(ctx context.Context, code string) (*domain.OAuthToken, error) {
	responseToken, err := p.oauthCfg.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	return &domain.OAuthToken{Token: responseToken}, nil
}

func (p oidcAuthenticator) AuthCodeUrl(state, nonce string) string {
	codeUrl := p.oauthCfg.AuthCodeURL(state, oidc.Nonce(nonce))

	return codeUrl
}

func (p oidcAuthenticator) GetUserInfo(ctx context.Context, token *domain.OAuthToken) (domain.RawAuthenticatorUserInfo, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("token does not contain id_token")
	}

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("token validation error: %w", err)
	}

	// TODO: do we need a nonce here?! this should be validated in the exchange step
	/*if idToken.Nonce != nonce {
		return nil, errors.New("nonce mismatch")
	}*/

	var tokenFields domain.RawAuthenticatorUserInfo
	if err = idToken.Claims(&tokenFields); err != nil {
		return nil, fmt.Errorf("failed to read extra claims: %w", err)
	}

	return tokenFields, nil
}

func (p oidcAuthenticator) ParseUserInfo(raw domain.RawAuthenticatorUserInfo) (*domain.AuthenticatorUserInfo, error) {
	isAdmin, _ := strconv.ParseBool(mapDefaultString(raw, p.cfg.FieldMap.IsAdmin, ""))
	groups := make([]domain.UserGroup, 0)
	if isAdmin {
		groups = append(groups, domain.UserGroupAdmin)
	}
	userInfo := &domain.AuthenticatorUserInfo{
		Identifier: domain.UserIdentifier(mapDefaultString(raw, p.cfg.FieldMap.UserIdentifier, "")),
		Email:      mapDefaultString(raw, p.cfg.FieldMap.Email, ""),
		Firstname:  mapDefaultString(raw, p.cfg.FieldMap.Firstname, ""),
		Lastname:   mapDefaultString(raw, p.cfg.FieldMap.Lastname, ""),
		Phone:      mapDefaultString(raw, p.cfg.FieldMap.Phone, ""),
		Department: mapDefaultString(raw, p.cfg.FieldMap.Department, ""),
		Groups:     groups,
	}

	return userInfo, nil
}
