package domain

import "strings"

type BaseProvider struct {
	Id AuthenticatorId `yaml:"id"`

	// ProviderName is an internal name that is used to distinguish oauth endpoints. It must not contain spaces or special characters.
	ProviderName string `yaml:"provider_name"`

	// DisplayName is shown to the user on the login page. If it is empty, ProviderName will be displayed.
	DisplayName string `yaml:"display_name"`

	// If RegistrationEnabled is set to true, wg-portal will create new users that do not exist in the database.
	IsRegistrationEnabled bool `yaml:"registration_enabled"`
}

func (p *BaseProvider) RegistrationEnabled() bool {
	return p.IsRegistrationEnabled
}

func (p *BaseProvider) GetId() AuthenticatorId {
	return p.Id
}

func (p *BaseProvider) GetName() string {
	return strings.ReplaceAll(p.ProviderName, " ", "_")
}

func (p *BaseProvider) GetDisplayName() string {
	return p.DisplayName
}

type BaseFields struct {
	UserIdentifier string `yaml:"user_identifier"`
	Email          string `yaml:"email"`
	Firstname      string `yaml:"firstname"`
	Lastname       string `yaml:"lastname"`
	Phone          string `yaml:"phone"`
	Department     string `yaml:"department"`
}

type OauthFields struct {
	BaseFields `yaml:",inline"`
	IsAdmin    string `yaml:"is_admin"`
}

type LdapFields struct {
	BaseFields      `yaml:",inline"`
	GroupMembership string `yaml:"memberof"`
}

type LdapProvider struct {
	BaseProvider `yaml:",inline"`

	Urls           []string `yaml:"urls"` // ldap server urls, if one of the servers fails, the next one in the list is used
	StartTLS       bool     `yaml:"start_tls"`
	CertValidation bool     `yaml:"cert_validation"`
	BaseDN         string   `yaml:"base_dn"`
	BindUser       string   `yaml:"bind_user"`
	BindPass       string   `yaml:"bind_pass"`

	FieldMap LdapFields `yaml:"field_map"`

	LoginFilter  string `yaml:"login_filter"` // {{login_identifier}} gets replaced with the login email address
	AdminGroupDN string `yaml:"admin_group"`  // Members of this group receive admin rights in WG-Portal

	Synchronize bool `yaml:"synchronize"`

	// If DeleteMissing is false, missing users will be deactivated
	DeleteMissing bool   `yaml:"delete_missing"`
	SyncFilter    string `yaml:"sync_filter"`
}

func (p *LdapProvider) GetType() AuthenticatorType {
	return AuthenticatorTypePlain
}

func (p *LdapProvider) InitiateFieldMap() {
	defaultMap := LdapFields{
		BaseFields: BaseFields{
			UserIdentifier: "mail",
			Email:          "mail",
			Firstname:      "givenName",
			Lastname:       "sn",
			Phone:          "telephoneNumber",
			Department:     "department",
		},
		GroupMembership: "memberOf",
	}
	if p.FieldMap.UserIdentifier != "" {
		defaultMap.UserIdentifier = p.FieldMap.UserIdentifier
	}
	if p.FieldMap.Email != "" {
		defaultMap.Email = p.FieldMap.Email
	}
	if p.FieldMap.Firstname != "" {
		defaultMap.Firstname = p.FieldMap.Firstname
	}
	if p.FieldMap.Lastname != "" {
		defaultMap.Lastname = p.FieldMap.Lastname
	}
	if p.FieldMap.Phone != "" {
		defaultMap.Phone = p.FieldMap.Phone
	}
	if p.FieldMap.Department != "" {
		defaultMap.Department = p.FieldMap.Department
	}
	if p.FieldMap.GroupMembership != "" {
		defaultMap.GroupMembership = p.FieldMap.GroupMembership
	}

	p.FieldMap = defaultMap
}

type OAuthBaseProvider struct {
	BaseProvider `yaml:",inline"`

	// ClientID is the application's ID.
	ClientID string `yaml:"client_id"`

	// ClientSecret is the application's secret.
	ClientSecret string `yaml:"client_secret"`

	// FieldMap is used to map the names of the user-info endpoint fields to wg-portal fields
	FieldMap OauthFields `yaml:"field_map"`
}

func (p *OAuthBaseProvider) InitiateFieldMap() {
	defaultMap := OauthFields{
		BaseFields: BaseFields{
			UserIdentifier: "sub",
			Email:          "email",
			Firstname:      "given_name",
			Lastname:       "family_name",
			Phone:          "phone",
			Department:     "department",
		},
		IsAdmin: "admin_flag",
	}
	if p.FieldMap.UserIdentifier != "" {
		defaultMap.UserIdentifier = p.FieldMap.UserIdentifier
	}
	if p.FieldMap.Email != "" {
		defaultMap.Email = p.FieldMap.Email
	}
	if p.FieldMap.Firstname != "" {
		defaultMap.Firstname = p.FieldMap.Firstname
	}
	if p.FieldMap.Lastname != "" {
		defaultMap.Lastname = p.FieldMap.Lastname
	}
	if p.FieldMap.Phone != "" {
		defaultMap.Phone = p.FieldMap.Phone
	}
	if p.FieldMap.Department != "" {
		defaultMap.Department = p.FieldMap.Department
	}
	if p.FieldMap.IsAdmin != "" {
		defaultMap.IsAdmin = p.FieldMap.IsAdmin
	}

	p.FieldMap = defaultMap
}

type OpenIDConnectProvider struct {
	OAuthBaseProvider `yaml:",inline"`

	BaseUrl string `yaml:"base_url"`

	// ExtraScopes specifies optional requested permissions.
	ExtraScopes []string `yaml:"extra_scopes"`
}

func (p *OpenIDConnectProvider) GetType() AuthenticatorType {
	return AuthenticatorTypeOidc
}

type OAuthProvider struct {
	OAuthBaseProvider `yaml:",inline"`

	AuthURL     string `yaml:"auth_url"`
	TokenURL    string `yaml:"token_url"`
	UserInfoURL string `yaml:"user_info_url"`

	// RedirectURL is the URL to redirect users going through
	// the OAuth flow, after the resource owner's URLs.
	RedirectURL string `yaml:"redirect_url"`

	// Scope specifies optional requested permissions.
	Scopes []string `yaml:"scopes"`
}

func (p *OAuthProvider) GetType() AuthenticatorType {
	return AuthenticatorTypeOAuth
}
