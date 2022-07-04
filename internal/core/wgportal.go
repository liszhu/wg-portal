package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/h44z/wg-portal/internal/authentication"
	"github.com/h44z/wg-portal/internal/lowlevel"
	"github.com/h44z/wg-portal/internal/model"
	"github.com/h44z/wg-portal/internal/persistence"
	"github.com/h44z/wg-portal/internal/user"
	"github.com/h44z/wg-portal/internal/wireguard"
	"github.com/sirupsen/logrus"
	"github.com/skip2/go-qrcode"
	"golang.zx2c4.com/wireguard/wgctrl"
)

// wgPortal combines wireguard.Manager and user.Manager and other components.
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
		return fmt.Errorf("failed to parse external url: %w", err)
	}

	for i := range w.cfg.Auth.OpenIDConnect {
		providerCfg := &w.cfg.Auth.OpenIDConnect[i]
		providerId := strings.ToLower(providerCfg.ProviderName)

		if _, exists := w.oauthAuthenticators[providerId]; exists {
			return fmt.Errorf("auth provider with name %s is already registerd", providerId)
		}

		redirectUrl := *extUrl
		redirectUrl.Path = path.Join(redirectUrl.Path, "/auth/login/", providerId, "/callback")

		authenticator, err := authentication.NewOidcAuthenticator(ctx, redirectUrl.String(), providerCfg)
		if err != nil {
			return fmt.Errorf("failed to setup oidc authentication provider %s: %w", providerCfg.ProviderName, err)
		}
		w.oauthAuthenticators[providerId] = authenticator
	}
	for i := range w.cfg.Auth.OAuth {
		providerCfg := &w.cfg.Auth.OAuth[i]
		providerId := strings.ToLower(providerCfg.ProviderName)

		if _, exists := w.oauthAuthenticators[providerId]; exists {
			return fmt.Errorf("auth provider with name %s is already registerd", providerId)
		}

		redirectUrl := *extUrl
		redirectUrl.Path = path.Join(redirectUrl.Path, "/auth/login/", providerId, "/callback")

		authenticator, err := authentication.NewPlainOauthAuthenticator(ctx, redirectUrl.String(), providerCfg)
		if err != nil {
			return fmt.Errorf("failed to setup oauth authentication provider %s: %w", providerId, err)
		}
		w.oauthAuthenticators[providerId] = authenticator
	}
	for i := range w.cfg.Auth.Ldap {
		providerCfg := &w.cfg.Auth.Ldap[i]
		providerId := strings.ToLower(providerCfg.URL)

		if _, exists := w.ldapAuthenticators[providerId]; exists {
			return fmt.Errorf("auth provider with name %s is already registerd", providerId)
		}

		authenticator, err := authentication.NewLdapAuthenticator(ctx, providerCfg)
		if err != nil {
			return fmt.Errorf("failed to setup ldap authentication provider %s: %w", providerId, err)
		}
		w.ldapAuthenticators[providerId] = authenticator
	}

	return nil
}

func (w *wgPortal) RunBackgroundTasks(ctx context.Context) {
	logrus.Info("Running background tasks...")

	// TODO: check for "temporary" peers and cleanup
	// TODO: check ldap authenticator for users (if sync is enabled)
	// TODO: gather stats of peers and interfaces

	logrus.Info("Finished background tasks")
}

func (w *wgPortal) GetUser(ctx context.Context, identifier model.UserIdentifier) (*model.User, error) {
	userEntry, err := w.users.GetUser(identifier)
	if err != nil {
		return nil, err
	}

	return userEntry, nil
}

func (w *wgPortal) GetUsers(ctx context.Context, options *userSearchOptions) (Paginator[*model.User], error) {
	if options == nil {
		options = UserSearchOptions()
	}

	users, err := w.filterUsers(options)
	if err != nil {
		return nil, err
	}

	return NewInMemoryPaginator(users), nil
}

func (w *wgPortal) GetUserIds(ctx context.Context, options *userSearchOptions) (Paginator[model.UserIdentifier], error) {
	if options == nil {
		options = UserSearchOptions()
	}

	users, err := w.filterUsers(options)
	if err != nil {
		return nil, err
	}

	userIds := make([]model.UserIdentifier, len(users))
	for i := range users {
		userIds[i] = users[i].Identifier
	}

	return NewInMemoryPaginator(userIds), nil
}

func (w *wgPortal) filterUsers(options *userSearchOptions) ([]*model.User, error) {
	var users []*model.User
	var err error

	// find
	switch options.filter {
	case "":
		users, err = w.users.GetUsers()
	default:
		filterStr := strings.ToLower(options.filter)
		users, err = w.users.GetUsers(func(user *model.User) bool {
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

	return users, nil
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
		if len(options.defaultPeerInterfaces) == 0 {
			options.defaultPeerInterfaces = w.cfg.DefaultPeerInterfaces
		}

		for _, interfaceId := range options.defaultPeerInterfaces {
			preparedPeer, err := w.preparePeer(ctx, interfaceId, u.Identifier)
			if err != nil {
				return nil, fmt.Errorf("failed to prepare default peer for %s: %w", interfaceId, err)
			}

			preparedPeer.DisplayName = fmt.Sprintf("Default Peer (%s)", preparedPeer.PublicKey[0:8])
			preparedPeer.Temporary = nil

			err = w.wg.SavePeers(preparedPeer)
			if err != nil {
				return nil, fmt.Errorf("failed to create default peer for %s: %w", interfaceId, err)
			}
		}
	}

	return u, nil
}

func (w *wgPortal) preparePeer(ctx context.Context, interfaceId model.InterfaceIdentifier, userId model.UserIdentifier) (*model.Peer, error) {
	i, err := w.wg.GetInterface(interfaceId)
	if err != nil {
		return nil, err
	}

	keyPair, err := w.wg.GetFreshKeypair()
	if err != nil {
		return nil, err
	}

	presharedKey, err := w.wg.GetPreSharedKey()
	if err != nil {
		return nil, err
	}

	// generate fresh IP's for all subnets that the interface has in use
	addressStr, err := w.wg.GetFreshIps(interfaceId)
	if err != nil {
		return nil, err
	}

	peerInterface := &model.PeerInterfaceConfig{
		Identifier:   interfaceId,
		Type:         i.Type,
		PublicKey:    i.PublicKey,
		AddressStr:   model.StringConfigOption{Value: addressStr, Overridable: true},
		DnsStr:       model.StringConfigOption{Value: i.PeerDefDnsStr, Overridable: true},
		DnsSearchStr: model.StringConfigOption{Value: i.PeerDefDnsSearchStr, Overridable: true},
		Mtu:          model.IntConfigOption{Value: i.PeerDefMtu, Overridable: true},
		FirewallMark: model.Int32ConfigOption{Value: i.PeerDefFirewallMark, Overridable: true},
		RoutingTable: model.StringConfigOption{Value: i.PeerDefRoutingTable, Overridable: true},
		PreUp:        model.StringConfigOption{Value: i.PeerDefPreUp, Overridable: true},
		PostUp:       model.StringConfigOption{Value: i.PeerDefPostUp, Overridable: true},
		PreDown:      model.StringConfigOption{Value: i.PeerDefPreDown, Overridable: true},
		PostDown:     model.StringConfigOption{Value: i.PeerDefPostDown, Overridable: true},
	}

	now := time.Now()
	peer := &model.Peer{
		Endpoint:            model.StringConfigOption{Value: i.PeerDefEndpoint, Overridable: true},
		AllowedIPsStr:       model.StringConfigOption{Value: i.PeerDefAllowedIPsStr, Overridable: true},
		ExtraAllowedIPsStr:  "",
		KeyPair:             keyPair,
		PresharedKey:        presharedKey,
		PersistentKeepalive: model.IntConfigOption{Value: i.PeerDefPersistentKeepalive, Overridable: true},
		DisplayName:         "Prepared Peer " + keyPair.PublicKey[0:8],
		Identifier:          model.PeerIdentifier(keyPair.PublicKey),
		UserIdentifier:      userId,
		Temporary:           &now,
		Interface:           peerInterface,
	}

	err = w.wg.SavePeers(peer)
	if err != nil {
		return nil, err
	}

	return peer, nil
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
		peers, err := w.wg.GetPeersForUser(u.Identifier)
		if err != nil {
			return nil, fmt.Errorf("failed to retrive peers: %w", err)
		}

		for _, peer := range peers {
			peer.Disabled = u.Disabled // copy disabled flag
		}

		err = w.wg.SavePeers(peers...)
		if err != nil {
			return nil, fmt.Errorf("failed to update peers: %w", err)
		}
	}

	return u, nil
}

func (w *wgPortal) DeleteUser(ctx context.Context, identifier model.UserIdentifier, options *userDeleteOptions) error {
	if options == nil {
		options = UserDeleteOptions()
	}

	peers, err := w.wg.GetPeersForUser(identifier)
	if err != nil {
		return fmt.Errorf("failed to retrive peers: %w", err)
	}

	// delete all peers of the given user
	if options.deletePeers {
		for _, peer := range peers {
			err = w.wg.RemovePeer(peer.Identifier)
			if err != nil {
				return fmt.Errorf("failed to delete peer %s: %w", peer.Identifier, err)
			}
		}
	} else { // unlink all previous linked peers
		for _, peer := range peers {
			peer.UserIdentifier = ""
		}

		err = w.wg.SavePeers(peers...)
		if err != nil {
			return fmt.Errorf("failed to update peers: %w", err)
		}
	}

	err = w.users.DeleteUser(identifier)
	if err != nil {
		return fmt.Errorf("deletion error: %w", err)
	}

	return nil
}

func (w *wgPortal) GetInterface(ctx context.Context, identifier model.InterfaceIdentifier) (*model.Interface, error) {
	iface, err := w.wg.GetInterface(identifier)
	if err != nil {
		return nil, err
	}

	return iface, nil
}

func (w *wgPortal) GetInterfaces(ctx context.Context, options *interfaceSearchOptions) (Paginator[*model.Interface], error) {
	if options == nil {
		options = InterfaceSearchOptions()
	}

	interfaces, err := w.filterInterfaces(options)
	if err != nil {
		return nil, err
	}

	return NewInMemoryPaginator(interfaces), nil
}

func (w *wgPortal) filterInterfaces(options *interfaceSearchOptions) ([]*model.Interface, error) {
	var interfaces []*model.Interface
	var err error

	// find
	interfaces, err = w.wg.GetInterfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to load interfaces from manager: %w", err)
	}

	// filter in place
	if options.filter != "" {
		n := 0
		filterStr := strings.ToLower(options.filter)
		for _, iface := range interfaces {
			if options.typ != "" && iface.Type != options.typ {
				continue
			}

			if strings.Contains(strings.ToLower(string(iface.Identifier)), filterStr) {
				interfaces[n] = iface
				n++
			}
			if strings.Contains(strings.ToLower(iface.DisplayName), filterStr) {
				interfaces[n] = iface
				n++
			}
		}
		interfaces = interfaces[:n]
	}

	return interfaces, nil
}

func (w *wgPortal) CreateInterface(ctx context.Context, m *model.Interface) (*model.Interface, error) {
	err := w.wg.CreateInterface(m.Identifier)
	if err != nil {
		return nil, fmt.Errorf("creation error: %w", err)
	}

	err = w.wg.UpdateInterface(m)
	if err != nil {
		return nil, fmt.Errorf("update error: %w", err)
	}

	return m, nil
}

func (w *wgPortal) UpdateInterface(ctx context.Context, m *model.Interface) (*model.Interface, error) {
	err := w.wg.UpdateInterface(m)
	if err != nil {
		return nil, fmt.Errorf("update error: %w", err)
	}

	return m, nil
}

func (w *wgPortal) DeleteInterface(ctx context.Context, identifier model.InterfaceIdentifier) error {
	err := w.wg.DeleteInterface(identifier)
	if err != nil {
		return fmt.Errorf("deletion error: %w", err)
	}

	return nil
}

func (w *wgPortal) PrepareNewInterface(ctx context.Context, identifier model.InterfaceIdentifier) (*model.Interface, error) {
	if identifier == "" { // find free identifier
		identifier = w.getFreshInterfaceId()
	}

	keyPair, err := w.wg.GetFreshKeypair()
	if err != nil {
		return nil, err
	}

	interfaces, err := w.wg.GetInterfaces()
	if err != nil {
		return nil, err
	}

	usedPorts := make([]int, len(interfaces))
	for _, iface := range interfaces {
		usedPorts = append(usedPorts, iface.ListenPort)
	}
	sort.Ints(usedPorts)

	freePort := 51820
	if len(usedPorts) > 0 {
		freePort = usedPorts[len(usedPorts)-1] + 1
	}

	i := &model.Interface{
		Identifier:  identifier,
		KeyPair:     keyPair,
		ListenPort:  freePort,
		DisplayName: string(identifier),
		Type:        "server",
		DriverType:  "linux",
	}

	return i, nil
}

func (w *wgPortal) getFreshInterfaceId() model.InterfaceIdentifier {
	interfaces, err := w.wg.GetInterfaces()
	if err != nil {
		return "wg0" // just use wg0 as default...
	}

	var index = 0
	var freshId model.InterfaceIdentifier
	for {
		freshId = model.InterfaceIdentifier(fmt.Sprintf("wg%d", index))

		for _, iface := range interfaces {
			if iface.Identifier == freshId {
				index++
				break
			}
		}

		return freshId
	}
}

func (w *wgPortal) GetInterfaceWgQuickConfig(ctx context.Context, identifier model.InterfaceIdentifier) (io.Reader, error) {
	iface, err := w.wg.GetInterface(identifier)
	if err != nil {
		return nil, err
	}

	peers, err := w.wg.GetPeers(identifier)
	if err != nil {
		return nil, err
	}

	config, err := w.wg.GetInterfaceConfig(iface, peers)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (w *wgPortal) ApplyGlobalSettings(ctx context.Context, identifier model.InterfaceIdentifier) error {
	err := w.wg.ApplyDefaultConfigs(identifier)
	if err != nil {
		return err
	}

	return nil
}

func (w *wgPortal) GetImportableInterfaces(ctx context.Context) (Paginator[*model.ImportableInterface], error) {
	interfaces, err := w.wg.GetImportableInterfaces()
	if err != nil {
		return nil, err
	}

	return NewInMemoryPaginator(interfaces), nil
}

func (w *wgPortal) ImportInterface(ctx context.Context, identifier model.InterfaceIdentifier, options *importOptions) (*model.Interface, error) {
	interfaces, err := w.wg.GetImportableInterfaces()
	if err != nil {
		return nil, err
	}

	var importableInterface *model.ImportableInterface
	for _, i := range interfaces {
		if i.Interface.Identifier != identifier {
			continue
		}

		importableInterface = i
		break
	}
	if importableInterface == nil {
		return nil, fmt.Errorf("no such importable interface: %s", identifier)
	}

	err = w.wg.ImportInterface(importableInterface)
	if err != nil {
		return nil, err
	}

	importedInterface, err := w.wg.GetInterface(identifier)
	if err != nil {
		return nil, err
	}

	return importedInterface, nil
}

func (w *wgPortal) GetPeer(ctx context.Context, identifier model.PeerIdentifier) (*model.Peer, error) {
	peer, err := w.wg.GetPeer(identifier)
	if err != nil {
		return nil, err
	}

	return peer, nil
}

func (w *wgPortal) GetPeers(ctx context.Context, options *peerSearchOptions) (Paginator[*model.Peer], error) {
	if options == nil {
		options = PeerSearchOptions()
	}

	peers, err := w.filterPeers(options)
	if err != nil {
		return nil, err
	}

	return NewInMemoryPaginator(peers), nil
}

func (w *wgPortal) GetPeerIds(ctx context.Context, options *peerSearchOptions) (Paginator[model.PeerIdentifier], error) {
	if options == nil {
		options = PeerSearchOptions()
	}

	peers, err := w.filterPeers(options)
	if err != nil {
		return nil, err
	}

	peerIds := make([]model.PeerIdentifier, len(peers))
	for i := range peers {
		peerIds[i] = peers[i].Identifier
	}

	return NewInMemoryPaginator(peerIds), nil
}

func (w *wgPortal) filterPeers(options *peerSearchOptions) ([]*model.Peer, error) {
	var peers []*model.Peer
	var err error

	// find
	switch {
	case options.interfaceFilter != "" && options.userFilter != "":
		peers, err = w.wg.GetPeersForUser(options.userFilter) // search by user, after that filter by interface
		n := 0
		for _, peer := range peers {
			if peer.Interface.Identifier == options.interfaceFilter {
				peers[n] = peer
				n++
			}
		}
		peers = peers[:n]
	case options.interfaceFilter != "":
		peers, err = w.wg.GetPeers(options.interfaceFilter)
	case options.userFilter != "":
		peers, err = w.wg.GetPeersForUser(options.userFilter)
	default:
		return nil, fmt.Errorf("specify at least interface or user filter")
	}

	// filter
	if options.filter != "" {
		n := 0
		filterStr := strings.ToLower(options.filter)
		for _, peer := range peers {
			if strings.Contains(strings.ToLower(string(peer.Identifier)), filterStr) {
				peers[n] = peer
				n++
			}
			if strings.Contains(strings.ToLower(peer.DisplayName), filterStr) {
				peers[n] = peer
				n++
			}
			if strings.Contains(strings.ToLower(peer.PublicKey), filterStr) {
				peers[n] = peer
				n++
			}
			if strings.Contains(strings.ToLower(peer.Interface.AddressStr.GetValue()), filterStr) {
				peers[n] = peer
				n++
			}
		}
		peers = peers[:n]
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load peers from manager: %w", err)
	}

	return peers, nil
}

func (w *wgPortal) CreatePeer(ctx context.Context, peer *model.Peer) (*model.Peer, error) {
	_, err := w.wg.GetPeer(peer.Identifier)
	if err != nil && err != wireguard.ErrPeerNotFound {
		return nil, err
	}

	err = w.wg.SavePeers(peer)
	if err != nil {
		return nil, err
	}

	return peer, nil
}

func (w *wgPortal) PrepareNewPeer(ctx context.Context, identifier model.InterfaceIdentifier) (*model.Peer, error) {
	preparedPeer, err := w.preparePeer(ctx, identifier, "")
	if err != nil {
		return nil, fmt.Errorf("failed to prepare default peer for %s: %w", identifier, err)
	}

	return preparedPeer, nil
}

func (w *wgPortal) UpdatePeer(ctx context.Context, peer *model.Peer) (*model.Peer, error) {
	_, err := w.wg.GetPeer(peer.Identifier)
	if err != nil {
		return nil, err
	}

	err = w.wg.SavePeers(peer)
	if err != nil {
		return nil, err
	}

	return peer, nil
}

func (w *wgPortal) DeletePeer(ctx context.Context, identifier model.PeerIdentifier) error {
	err := w.wg.RemovePeer(identifier)
	if err != nil {
		return err
	}

	return nil
}

func (w *wgPortal) GetPeerQrCode(ctx context.Context, identifier model.PeerIdentifier) (io.Reader, error) {
	config, err := w.GetPeerWgQuickConfig(ctx, identifier)
	if err != nil {
		return nil, err
	}

	var configStr string
	if b, err := io.ReadAll(config); err == nil {
		configStr = string(b)
	} else {
		return nil, err
	}

	png, err := qrcode.Encode(configStr, qrcode.Low, 250)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(png), nil
}

func (w *wgPortal) GetPeerWgQuickConfig(ctx context.Context, identifier model.PeerIdentifier) (io.Reader, error) {
	peer, err := w.wg.GetPeer(identifier)
	if err != nil {
		return nil, err
	}

	config, err := w.wg.GetPeerConfig(peer)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (w *wgPortal) SendWgQuickConfigMail(ctx context.Context, options *mailOptions) error {
	//TODO implement me
	return fmt.Errorf("UNIMPLEMENTED")
}
