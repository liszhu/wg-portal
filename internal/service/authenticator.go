package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/h44z/wg-portal/internal/domain"
	"github.com/h44z/wg-portal/internal/ports"
)

type contextKey string

var contextAuthValueKey contextKey = "wgPortalAuthData"

type authenticatorService struct {
	plainAuthenticators map[domain.AuthenticatorId]ports.PlainAuthenticatorRepository
	oauthAuthenticators map[domain.AuthenticatorId]ports.OauthAuthenticatorRepository
}

func NewAuthenticatorService(plain []ports.PlainAuthenticatorRepository, oauth []ports.OauthAuthenticatorRepository) (*authenticatorService, error) {
	a := &authenticatorService{
		plainAuthenticators: make(map[domain.AuthenticatorId]ports.PlainAuthenticatorRepository),
		oauthAuthenticators: make(map[domain.AuthenticatorId]ports.OauthAuthenticatorRepository),
	}

	idMap := make(map[domain.AuthenticatorId]struct{})

	for i := range plain {
		id := plain[i].Config().GetId()

		if _, contains := idMap[id]; contains {
			return nil, fmt.Errorf("duplicate authenticator id %s", id)
		}
		idMap[id] = struct{}{}

		a.plainAuthenticators[id] = plain[i]
	}

	for i := range oauth {
		id := oauth[i].Config().GetId()

		if _, contains := idMap[id]; contains {
			return nil, fmt.Errorf("duplicate authenticator id %s", id)
		}
		idMap[id] = struct{}{}

		a.oauthAuthenticators[id] = oauth[i]
	}

	return a, nil
}

func (a authenticatorService) IsAuthenticated(ctx context.Context) bool {
	authData := a.getContextData(ctx)
	if authData == nil {
		return false
	}

	return authData.IsAuthenticated()
}

func (a authenticatorService) AuthenticationInfo(ctx context.Context) *domain.ContextAuthenticationInfo {
	return a.getContextData(ctx)
}

func (a authenticatorService) DestroyLogin(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextAuthValueKey, nil)
}

func (a authenticatorService) GetAuthenticators() []domain.AuthenticatorConfig {
	var allConfigs []domain.AuthenticatorConfig

	for _, repo := range a.plainAuthenticators {
		allConfigs = append(allConfigs, repo.Config())
	}

	for _, repo := range a.oauthAuthenticators {
		allConfigs = append(allConfigs, repo.Config())
	}

	return allConfigs
}

func (a authenticatorService) GetAuthenticator(authenticator domain.AuthenticatorId) (domain.AuthenticatorConfig, error) {
	for _, repo := range a.plainAuthenticators {
		if repo.Config().GetId() == authenticator {
			return repo.Config(), nil
		}
	}

	for _, repo := range a.oauthAuthenticators {
		if repo.Config().GetId() == authenticator {
			return repo.Config(), nil
		}
	}

	return nil, errors.New("authenticator not found")
}

func (a authenticatorService) AuthenticateContext(ctx context.Context, authenticator domain.AuthenticatorId, username, password string) (context.Context, error) {
	authCfg, err := a.GetAuthenticator(authenticator)
	if err != nil {
		return nil, err
	}
	if authCfg.GetType() != domain.AuthenticatorTypePlain {
		return nil, errors.New("unsupported authentication method")
	}

	err = a.plainAuthenticators[authenticator].Authenticate(ctx, username, password)
	if err != nil {
		return nil, errors.New("authentication failed")
	}

	info, err := a.plainAuthenticators[authenticator].GetUserInfo(ctx, username)
	if err != nil {
		return nil, err
	}
	user, err := a.plainAuthenticators[authenticator].ParseUserInfo(info)
	if err != nil {
		return nil, err
	}

	authInfo := &domain.ContextAuthenticationInfo{
		AuthenticatorId:   authenticator,
		AuthenticatorType: authCfg.GetType(),
		UserId:            user.Identifier,
		Username:          username,
		Groups:            user.Groups,
	}

	return context.WithValue(ctx, contextAuthValueKey, authInfo), nil
}

func (a authenticatorService) GetOauthUrl(authenticator domain.AuthenticatorId) (url, state, nonce string, err error) {
	authCfg, err := a.GetAuthenticator(authenticator)
	if err != nil {
		return "", "", "", err
	}
	if authCfg.GetType() != domain.AuthenticatorTypeOAuth && authCfg.GetType() != domain.AuthenticatorTypeOidc {
		return "", "", "", errors.New("unsupported authentication method")
	}

	state = "random_state" // TODO
	nonce = "random_nonce" // TODO
	url = a.oauthAuthenticators[authenticator].AuthCodeUrl(state, nonce)

	return
}

func (a authenticatorService) AuthenticateContextWithCode(ctx context.Context, authenticator domain.AuthenticatorId, code, state, nonce string) (context.Context, error) {
	authCfg, err := a.GetAuthenticator(authenticator)
	if err != nil {
		return nil, err
	}
	if authCfg.GetType() != domain.AuthenticatorTypeOAuth && authCfg.GetType() != domain.AuthenticatorTypeOidc {
		return nil, errors.New("unsupported authentication method")
	}

	token, err := a.oauthAuthenticators[authenticator].Exchange(ctx, code)
	if err != nil {
		return nil, errors.New("authentication failed")
	}

	// TODO: check nonce in token (for OIDC)?

	info, err := a.oauthAuthenticators[authenticator].GetUserInfo(ctx, token)
	if err != nil {
		return nil, err
	}
	user, err := a.oauthAuthenticators[authenticator].ParseUserInfo(info)
	if err != nil {
		return nil, err
	}

	authInfo := &domain.ContextAuthenticationInfo{
		AuthenticatorId:   authenticator,
		AuthenticatorType: authCfg.GetType(),
		Token:             token,
		UserId:            user.Identifier,
		Username:          string(user.Identifier),
		Groups:            user.Groups,
	}

	return context.WithValue(ctx, contextAuthValueKey, authInfo), nil
}

func (a authenticatorService) GetUserInfo(ctx context.Context) (*domain.AuthenticatorUserInfo, error) {
	authData := a.getContextData(ctx)
	if authData == nil || !authData.IsAuthenticated() {
		return nil, errors.New("not authenticated")
	}

	var userInfo *domain.AuthenticatorUserInfo
	switch authData.AuthenticatorType {
	case domain.AuthenticatorTypePlain:
		raw, err := a.plainAuthenticators[authData.AuthenticatorId].GetUserInfo(ctx, authData.Username)
		if err != nil {
			return nil, err
		}
		userInfo, err = a.plainAuthenticators[authData.AuthenticatorId].ParseUserInfo(raw)
		if err != nil {
			return nil, err
		}
	case domain.AuthenticatorTypeOAuth, domain.AuthenticatorTypeOidc:
		raw, err := a.oauthAuthenticators[authData.AuthenticatorId].GetUserInfo(ctx, authData.Token)
		if err != nil {
			return nil, err
		}
		userInfo, err = a.oauthAuthenticators[authData.AuthenticatorId].ParseUserInfo(raw)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unknown authenticator type")
	}

	return userInfo, nil
}

func (a authenticatorService) GetAllUserInfos(ctx context.Context) ([]*domain.AuthenticatorUserInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (a authenticatorService) getContextData(ctx context.Context) *domain.ContextAuthenticationInfo {
	value := ctx.Value(contextAuthValueKey)
	if value == nil {
		return nil
	}

	authData, ok := value.(*domain.ContextAuthenticationInfo)
	if !ok {
		return nil
	}

	return authData
}
