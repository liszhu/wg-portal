package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/h44z/wg-portal/internal/domain"
	"golang.org/x/oauth2"
)

type plainOauthAuthenticator struct {
	callbackBaseUrl string

	oauthCfg *oauth2.Config
	cfg      *domain.OAuthProvider
	client   *http.Client

	userInfoEndpoint string
}

func NewOauthAuthenticator(baseUrl string, cfg *domain.OAuthProvider) (*plainOauthAuthenticator, error) {
	authenticator := &plainOauthAuthenticator{
		cfg:             cfg,
		callbackBaseUrl: baseUrl,
		client: &http.Client{
			Timeout: time.Second * 10,
		},
		oauthCfg: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:   cfg.AuthURL,
				TokenURL:  cfg.TokenURL,
				AuthStyle: oauth2.AuthStyleAutoDetect,
			},
			Scopes: cfg.Scopes,
		},
	}

	var err error
	authenticator.oauthCfg.RedirectURL, err = authenticator.GetCallbackUrl()
	if err != nil {
		return nil, fmt.Errorf("callback url error: %w", err)
	}

	authenticator.cfg.InitiateFieldMap()

	return authenticator, nil
}

func (p plainOauthAuthenticator) Config() domain.AuthenticatorConfig {
	return p.cfg
}

func (p plainOauthAuthenticator) GetCallbackUrl() (string, error) {
	callbackUrl, err := url.Parse(p.callbackBaseUrl)
	if err != nil {
		return "", err
	}
	callbackUrl.Path = path.Join(callbackUrl.Path, "oauth", p.cfg.ProviderName, "callback")
	return callbackUrl.String(), nil
}

func (p plainOauthAuthenticator) Exchange(ctx context.Context, code string) (*domain.OAuthToken, error) {
	responseToken, err := p.oauthCfg.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	return &domain.OAuthToken{Token: responseToken}, nil
}

func (p plainOauthAuthenticator) AuthCodeUrl(state, _ string) string {
	codeUrl := p.oauthCfg.AuthCodeURL(state)

	return codeUrl
}

func (p plainOauthAuthenticator) GetUserInfo(ctx context.Context, token *domain.OAuthToken) (domain.RawAuthenticatorUserInfo, error) {
	req, err := http.NewRequest("GET", p.userInfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	req.WithContext(ctx)

	response, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execution error: %w", err)
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var userFields domain.RawAuthenticatorUserInfo
	err = json.Unmarshal(contents, &userFields)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return userFields, nil
}

func (p plainOauthAuthenticator) ParseUserInfo(raw domain.RawAuthenticatorUserInfo) (*domain.AuthenticatorUserInfo, error) {
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
