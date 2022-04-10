package auth

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/h44z/wg-portal/internal/domain"
	"github.com/sirupsen/logrus"
)

type ldapAuthenticator struct {
	cfg          *domain.LdapProvider
	adminGroupDN *ldap.DN
}

func NewLdapAuthenticator(cfg *domain.LdapProvider) (*ldapAuthenticator, error) {
	authenticator := &ldapAuthenticator{
		cfg: cfg,
	}

	authenticator.cfg.InitiateFieldMap()

	dn, err := ldap.ParseDN(cfg.AdminGroupDN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse admin group DN: %w", err)
	}
	authenticator.adminGroupDN = dn

	return authenticator, nil
}

func (l ldapAuthenticator) Config() domain.AuthenticatorConfig {
	return l.cfg
}

func (l ldapAuthenticator) Authenticate(ctx context.Context, username, password string) error {
	conn, err := l.connect()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer l.disconnect(conn)

	attrs := []string{"dn"}

	loginFilter := strings.Replace(l.cfg.LoginFilter, "{{login_identifier}}", username, -1)
	searchRequest := ldap.NewSearchRequest(
		l.cfg.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 20, false, // 20 second time limit
		loginFilter, attrs, nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		return fmt.Errorf("failed to search: %w", err)
	}

	if len(sr.Entries) == 0 {
		return errors.New("user not found")
	}

	if len(sr.Entries) > 1 {
		return errors.New("no unique user found")
	}

	// Bind as the user to verify their password
	userDN := sr.Entries[0].DN
	err = conn.Bind(userDN, password)
	if err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}
	_ = conn.Unbind()

	return nil
}

func (l ldapAuthenticator) GetAllUserInfos(ctx context.Context) ([]domain.RawAuthenticatorUserInfo, error) {
	conn, err := l.connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer l.disconnect(conn)

	attrs := l.getLdapSearchAttributes()

	searchRequest := ldap.NewSearchRequest(
		l.cfg.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 20, false, // 20 second time limit, TODO: use context
		l.cfg.SyncFilter, attrs, nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	users := l.convertLdapEntries(sr)

	return users, nil
}

func (l ldapAuthenticator) GetUserInfo(ctx context.Context, username string) (domain.RawAuthenticatorUserInfo, error) {
	conn, err := l.connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer l.disconnect(conn)

	attrs := l.getLdapSearchAttributes()

	loginFilter := strings.Replace(l.cfg.LoginFilter, "{{login_identifier}}", username, -1)
	searchRequest := ldap.NewSearchRequest(
		l.cfg.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 20, false, // 20 second time limit
		loginFilter, attrs, nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	if len(sr.Entries) == 0 {
		return nil, errors.New("user not found")
	}

	if len(sr.Entries) > 1 {
		return nil, errors.New("no unique user found")
	}

	users := l.convertLdapEntries(sr)

	return users[0], nil
}

func (l ldapAuthenticator) ParseUserInfo(raw domain.RawAuthenticatorUserInfo) (*domain.AuthenticatorUserInfo, error) {
	groups, err := l.getGroups(raw[l.cfg.FieldMap.GroupMembership].([][]byte))
	if err != nil {
		return nil, fmt.Errorf("failed to parse group: %w", err)
	}
	userInfo := &domain.AuthenticatorUserInfo{
		Identifier: domain.UserIdentifier(mapDefaultString(raw, l.cfg.FieldMap.UserIdentifier, "")),
		Email:      mapDefaultString(raw, l.cfg.FieldMap.Email, ""),
		Firstname:  mapDefaultString(raw, l.cfg.FieldMap.Firstname, ""),
		Lastname:   mapDefaultString(raw, l.cfg.FieldMap.Lastname, ""),
		Phone:      mapDefaultString(raw, l.cfg.FieldMap.Phone, ""),
		Department: mapDefaultString(raw, l.cfg.FieldMap.Department, ""),
		Groups:     groups,
	}

	return userInfo, nil
}

func (l ldapAuthenticator) connect() (*ldap.Conn, error) {
	tlsConfig := &tls.Config{InsecureSkipVerify: !l.cfg.CertValidation}

	tries := 0
	var conn *ldap.Conn
	var err error
	for tries < len(l.cfg.Urls) {
		conn, err = ldap.DialURL(l.cfg.Urls[tries], ldap.DialWithTLSConfig(tlsConfig))
		if err != nil {
			logrus.Warnf("ldap connection to %s failed (retries: %d)", l.cfg.Urls[tries], tries)
			tries++
			continue
		}
		break
	}
	if err != nil {
		return nil, fmt.Errorf("dail error: %w", err)
	}

	if l.cfg.StartTLS { // Reconnect with TLS
		if err = conn.StartTLS(tlsConfig); err != nil {
			return nil, fmt.Errorf("tls error: %w", err)
		}
	}

	if err = conn.Bind(l.cfg.BindUser, l.cfg.BindPass); err != nil {
		return nil, fmt.Errorf("bind error: %w", err)
	}

	return conn, nil
}

func (l ldapAuthenticator) disconnect(conn *ldap.Conn) {
	if conn != nil {
		conn.Close()
	}
}

func (l ldapAuthenticator) convertLdapEntries(sr *ldap.SearchResult) []domain.RawAuthenticatorUserInfo {
	users := make([]domain.RawAuthenticatorUserInfo, len(sr.Entries))

	fieldMap := l.cfg.FieldMap
	for i, entry := range sr.Entries {
		userData := make(domain.RawAuthenticatorUserInfo)
		userData[fieldMap.UserIdentifier] = entry.DN
		userData[fieldMap.Email] = entry.GetAttributeValue(fieldMap.Email)
		userData[fieldMap.Firstname] = entry.GetAttributeValue(fieldMap.Firstname)
		userData[fieldMap.Lastname] = entry.GetAttributeValue(fieldMap.Lastname)
		userData[fieldMap.Phone] = entry.GetAttributeValue(fieldMap.Phone)
		userData[fieldMap.Department] = entry.GetAttributeValue(fieldMap.Department)
		userData[fieldMap.GroupMembership] = entry.GetRawAttributeValues(fieldMap.GroupMembership)

		users[i] = userData
	}
	return users
}

func (l ldapAuthenticator) getLdapSearchAttributes() []string {
	fieldMap := l.cfg.FieldMap
	attrs := []string{"dn", fieldMap.UserIdentifier}
	if fieldMap.Email != "" {
		attrs = append(attrs, fieldMap.Email)
	}
	if fieldMap.Firstname != "" {
		attrs = append(attrs, fieldMap.Firstname)
	}
	if fieldMap.Lastname != "" {
		attrs = append(attrs, fieldMap.Lastname)
	}
	if fieldMap.Phone != "" {
		attrs = append(attrs, fieldMap.Phone)
	}
	if fieldMap.Department != "" {
		attrs = append(attrs, fieldMap.Department)
	}
	if fieldMap.GroupMembership != "" {
		attrs = append(attrs, fieldMap.GroupMembership)
	}

	return uniqueStringSlice(attrs)
}

func (l ldapAuthenticator) getGroups(groupData [][]byte) ([]domain.UserGroup, error) {
	groups := make([]domain.UserGroup, 0)
	for _, group := range groupData {
		dn, err := ldap.ParseDN(string(group))
		if err != nil {
			return nil, fmt.Errorf("invalid group dn %s: %w", string(group), err)
		}
		if l.adminGroupDN.Equal(dn) {
			groups = append(groups, domain.UserGroupAdmin)
		}
		// TODO: add other dn's as groups?
	}

	return groups, nil
}

func uniqueStringSlice(slice []string) []string {
	keys := make(map[string]struct{})
	uniqueSlice := make([]string, 0, len(slice))
	for _, entry := range slice {
		if _, exists := keys[entry]; !exists {
			keys[entry] = struct{}{}
			uniqueSlice = append(uniqueSlice, entry)
		}
	}
	return uniqueSlice
}

// mapDefaultString returns the string value for the given key or a default value
func mapDefaultString(m map[string]interface{}, key string, dflt string) string {
	if m == nil {
		return dflt
	}
	if tmp, ok := m[key]; !ok {
		return dflt
	} else {
		switch v := tmp.(type) {
		case string:
			return v
		case nil:
			return dflt
		default:
			return fmt.Sprintf("%v", v)
		}
	}
}
