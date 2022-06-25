package core

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"

	"golang.zx2c4.com/wireguard/wgctrl"

	"github.com/h44z/wg-portal/internal/lowlevel"

	"github.com/pkg/errors"

	"github.com/h44z/wg-portal/internal/authentication"
	"github.com/h44z/wg-portal/internal/model"
	"github.com/h44z/wg-portal/internal/persistence"
	"github.com/h44z/wg-portal/internal/user"
	"github.com/h44z/wg-portal/internal/wireguard"
)

type wgPortal struct {
	cfg *Config

	db                  *persistence.Database
	wg                  wireguard.Manager
	users               user.Manager
	oauthAuthenticators map[string]authentication.OauthAuthenticator
	ldapAuthenticators  map[string]authentication.LdapAuthenticator
}

func NewWgPortal(cfg *Config) (*wgPortal, error) {
	portal := &wgPortal{
		cfg:                 cfg,
		oauthAuthenticators: make(map[string]authentication.OauthAuthenticator),
		ldapAuthenticators:  make(map[string]authentication.LdapAuthenticator),
	}

	// initiate dependencies

	database, err := persistence.NewDatabase(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize persistent store: %w", err)
	}
	portal.db = database

	wg, err := wgctrl.New()
	if err != nil {
		return nil, fmt.Errorf("failed to get wgctrl handle: %w", err)
	}

	nl := &lowlevel.NetlinkManager{}

	wgManager, err := wireguard.NewPersistentManager(wg, nl, portal.db)
	if err != nil {
		return nil, fmt.Errorf("failed to setup WireGuard manager: %w", err)
	}
	portal.wg = wgManager

	userManager, err := user.NewPersistentManager(portal.db)
	if err != nil {
		return nil, fmt.Errorf("failed to setup user manager: %w", err)
	}
	portal.users = userManager

	// start setup procedures

	setupCtx, cancel := context.WithTimeout(context.Background(), cfg.Core.StartupTimeout)
	defer cancel()

	if err := portal.setup(setupCtx); err != nil {
		return nil, fmt.Errorf("failed to setup: %w", err)
	}

	return portal, nil
}

func (w *wgPortal) setup(ctx context.Context) error {
	if err := w.setupExternalAuthProviders(ctx); err != nil {
		return fmt.Errorf("external authentication provider error: %w", err)
	}

	return nil
}

func (w *wgPortal) setupExternalAuthProviders(ctx context.Context) error {
	extUrl, err := url.Parse(w.cfg.Core.ExternalUrl)
	if err != nil {
		return errors.WithMessage(err, "failed to parse external url")
	}

	for i := range w.cfg.Auth.OpenIDConnect {
		providerCfg := &w.cfg.Auth.OpenIDConnect[i]
		providerId := strings.ToLower(providerCfg.ProviderName)

		if _, exists := w.oauthAuthenticators[providerId]; exists {
			return errors.Errorf("auth provider with name %s is already registerd", providerId)
		}

		redirectUrl := *extUrl
		redirectUrl.Path = path.Join(redirectUrl.Path, "/auth/login/", providerId, "/callback")

		authenticator, err := authentication.NewOidcAuthenticator(ctx, redirectUrl.String(), providerCfg)
		if err != nil {
			return errors.WithMessagef(err, "failed to setup oidc authentication provider %s", providerCfg.ProviderName)
		}
		w.oauthAuthenticators[providerId] = authenticator
	}
	for i := range w.cfg.Auth.OAuth {
		providerCfg := &w.cfg.Auth.OAuth[i]
		providerId := strings.ToLower(providerCfg.ProviderName)

		if _, exists := w.oauthAuthenticators[providerId]; exists {
			return errors.Errorf("auth provider with name %s is already registerd", providerId)
		}

		redirectUrl := *extUrl
		redirectUrl.Path = path.Join(redirectUrl.Path, "/auth/login/", providerId, "/callback")

		authenticator, err := authentication.NewPlainOauthAuthenticator(ctx, redirectUrl.String(), providerCfg)
		if err != nil {
			return errors.WithMessagef(err, "failed to setup oauth authentication provider %s", providerId)
		}
		w.oauthAuthenticators[providerId] = authenticator
	}
	for i := range w.cfg.Auth.Ldap {
		providerCfg := &w.cfg.Auth.Ldap[i]
		providerId := strings.ToLower(providerCfg.URL)

		if _, exists := w.ldapAuthenticators[providerId]; exists {
			return errors.Errorf("auth provider with name %s is already registerd", providerId)
		}

		authenticator, err := authentication.NewLdapAuthenticator(ctx, providerCfg)
		if err != nil {
			return errors.WithMessagef(err, "failed to setup ldap authentication provider %s", providerId)
		}
		w.ldapAuthenticators[providerId] = authenticator
	}

	return nil
}

func (w *wgPortal) RunBackgroundTasks(ctx context.Context) {
	//TODO implement me
	logrus.Info("Running background tasks...")
	logrus.Info("Finished background tasks")
}

func (w *wgPortal) GetUsers(ctx context.Context, options *userSearchOptions) ([]model.User, error) {
	if options == nil {
		options = UserSearchOptions()
	}

	var filteredAndPagedUsers []model.User
	var users []*model.User
	var err error

	// find
	switch options.filter {
	case "":
		users, err = w.users.GetAllUsers()
	default:
		filterStr := strings.ToLower(options.filter)
		users, err = w.users.GetFilteredUsers(func(user *model.User) bool {
			if strings.Contains(strings.ToLower(string(user.Identifier)), filterStr) {
				return true
			}
			if strings.Contains(strings.ToLower(user.Firstname), filterStr) {
				return true
			}
			if strings.Contains(strings.ToLower(user.Lastname), filterStr) {
				return true
			}
			if strings.Contains(strings.ToLower(user.Email), filterStr) {
				return true
			}
			return false
		})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load users from manager: %w", err)
	}

	// sort
	sort.Slice(users, func(i, j int) bool {
		switch strings.ToLower(options.sortBy) {
		case "firstname":
			return users[i].Firstname < users[j].Firstname
		case "lastname":
			return users[i].Lastname < users[j].Lastname
		case "email":
			return users[i].Email < users[j].Email
		case "source":
			return users[i].Source < users[j].Source
		default:
			return users[i].Identifier < users[j].Identifier
		}
	})

	// page
	if options.pageSize != PageSizeAll {
		if options.pageOffset >= len(users) {
			return nil, errors.New("invalid page offset")
		}

		filteredAndPagedUsers = make([]model.User, 0, options.pageSize)
		for i := options.pageOffset; i < options.pageOffset+options.pageSize; i++ {
			if i >= len(users) {
				break // check if we reached the end
			}
			filteredAndPagedUsers = append(filteredAndPagedUsers, *users[i])
		}
	} else {
		filteredAndPagedUsers = make([]model.User, len(users))
		for i := range users {
			filteredAndPagedUsers[i] = *users[i]
		}
	}

	return filteredAndPagedUsers, nil
}

func (w *wgPortal) GetUserIds(ctx context.Context, options *userSearchOptions) ([]model.UserIdentifier, error) {
	if options == nil {
		options = UserSearchOptions()
	}

	var filteredAndPagedIds []model.UserIdentifier
	var users []*model.User
	var err error

	// find
	switch options.filter {
	case "":
		users, err = w.users.GetAllUsers()
	default:
		filterStr := strings.ToLower(options.filter)
		users, err = w.users.GetFilteredUsers(func(user *model.User) bool {
			if strings.Contains(strings.ToLower(string(user.Identifier)), filterStr) {
				return true
			}
			if strings.Contains(strings.ToLower(user.Firstname), filterStr) {
				return true
			}
			if strings.Contains(strings.ToLower(user.Lastname), filterStr) {
				return true
			}
			if strings.Contains(strings.ToLower(user.Email), filterStr) {
				return true
			}
			return false
		})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load users from manager: %w", err)
	}

	// sort
	sort.Slice(users, func(i, j int) bool {
		switch strings.ToLower(options.sortBy) {
		case "firstname":
			return users[i].Firstname < users[j].Firstname
		case "lastname":
			return users[i].Lastname < users[j].Lastname
		case "email":
			return users[i].Email < users[j].Email
		case "source":
			return users[i].Source < users[j].Source
		default:
			return users[i].Identifier < users[j].Identifier
		}
	})

	// page
	if options.pageSize != PageSizeAll {
		if options.pageOffset >= len(users) {
			return nil, errors.New("invalid page offset")
		}

		filteredAndPagedIds = make([]model.UserIdentifier, 0, options.pageSize)
		for i := options.pageOffset; i < options.pageOffset+options.pageSize; i++ {
			if i >= len(users) {
				break // check if we reached the end
			}
			filteredAndPagedIds = append(filteredAndPagedIds, users[i].Identifier)
		}
	} else {
		filteredAndPagedIds = make([]model.UserIdentifier, len(users))
		for i := range users {
			filteredAndPagedIds[i] = users[i].Identifier
		}
	}

	return filteredAndPagedIds, nil
}

func (w *wgPortal) CreateUser(ctx context.Context, u *model.User, options *userCreateOptions) (*model.User, error) {
	if options == nil {
		options = UserCreateOptions()
	}

	err := w.users.CreateUser(u)
	if err != nil {
		return nil, fmt.Errorf("creation error: %w", err)
	}

	// create a default peer for the given user
	if options.createDefaultPeer {
		// TODO: implement
	}

	return u, nil
}

func (w *wgPortal) UpdateUser(ctx context.Context, u *model.User, options *userUpdateOptions) (*model.User, error) {
	if options == nil {
		options = UserUpdateOptions()
	}

	err := w.users.UpdateUser(u)
	if err != nil {
		return nil, fmt.Errorf("update error: %w", err)
	}

	// update peer state (disable all peers if user has been disabled)
	if options.syncPeerState {
		// TODO: implement
	}

	return u, nil
}

func (w *wgPortal) DeleteUser(ctx context.Context, identifier model.UserIdentifier, options *userDeleteOptions) error {
	if options == nil {
		options = UserDeleteOptions()
	}

	err := w.users.DeleteUser(identifier)
	if err != nil {
		return fmt.Errorf("deletion error: %w", err)
	}

	// delete all peers of the given user
	if options.deletePeers {
		// TODO: implement
	} else { // unlink all previous linked peers
		// TODO: implement
	}

	return nil
}

func (w *wgPortal) GetInterfaces(ctx context.Context, options *interfaceSearchOptions) ([]model.Interface, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wgPortal) CreateInterface(ctx context.Context, m *model.Interface) (*model.Interface, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wgPortal) UpdateInterface(ctx context.Context, m *model.Interface) (*model.Interface, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wgPortal) DeleteInterface(ctx context.Context, identifier model.InterfaceIdentifier) error {
	//TODO implement me
	panic("implement me")
}

func (w *wgPortal) GetInterfaceWgQuickConfig(ctx context.Context, m *model.Interface) (io.Reader, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wgPortal) GetImportableInterfaces(ctx context.Context, options *interfaceSearchOptions) ([]model.ImportableInterface, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wgPortal) ImportInterface(ctx context.Context, importableInterface *model.ImportableInterface, options *importOptions) (*model.Interface, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wgPortal) GetPeers(ctx context.Context, options *peerSearchOptions) ([]model.Peer, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wgPortal) GetPeerIds(ctx context.Context, options *peerSearchOptions) ([]model.PeerIdentifier, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wgPortal) CreatePeer(ctx context.Context, peer *model.Peer) (*model.Peer, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wgPortal) PrepareNewPeer(ctx context.Context, identifier model.InterfaceIdentifier) (*model.Peer, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wgPortal) UpdatePeer(ctx context.Context, peer *model.Peer) (*model.Peer, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wgPortal) DeletePeer(ctx context.Context, identifier model.PeerIdentifier) error {
	//TODO implement me
	panic("implement me")
}

func (w *wgPortal) GetPeerQrCode(ctx context.Context, peer *model.Peer) (io.Reader, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wgPortal) GetPeerWgQuickConfig(ctx context.Context, peer *model.Peer) (io.Reader, error) {
	//TODO implement me
	panic("implement me")
}

func (w *wgPortal) SendWgQuickConfigMail(ctx context.Context, options *mailOptions) error {
	//TODO implement me
	panic("implement me")
}
