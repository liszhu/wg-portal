package domain

import (
	"golang.org/x/oauth2"
)

type AuthenticatorType string
type AuthenticatorId string

const (
	AuthenticatorTypeOAuth AuthenticatorType = "oauth"
	AuthenticatorTypeOidc  AuthenticatorType = "oidc"
	AuthenticatorTypePlain AuthenticatorType = "plain"
)

type OAuthToken struct {
	*oauth2.Token
}

// ContextAuthenticationInfo is stored as context attribute. This data is used to check if the given context was authenticated.
type ContextAuthenticationInfo struct {
	AuthenticatorId   AuthenticatorId
	AuthenticatorType AuthenticatorType
	Token             *OAuthToken

	UserId   UserIdentifier
	Username string
	Groups   []UserGroup
}

func (c ContextAuthenticationInfo) IsAuthenticated() bool {
	return c.AuthenticatorId != "" && c.AuthenticatorType != "" && c.Username != "" && c.UserId != ""
}

func (c ContextAuthenticationInfo) ContainsGroup(group UserGroup) bool {
	if !c.IsAuthenticated() {
		return false
	}

	for _, g := range c.Groups {
		if g == group {
			return true
		}
	}

	return false
}

type RawAuthenticatorUserInfo map[string]interface{}

type AuthenticatorUserInfo struct {
	Identifier UserIdentifier

	Email      string
	Firstname  string
	Lastname   string
	Phone      string
	Department string

	Groups []UserGroup
}

type AuthenticatorConfig interface {
	GetType() AuthenticatorType
	RegistrationEnabled() bool
	GetId() AuthenticatorId
	GetName() string
	GetDisplayName() string
	InitiateFieldMap()
}
