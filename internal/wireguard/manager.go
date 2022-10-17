package wireguard

import (
	"io"

	"github.com/h44z/wg-portal/internal/lowlevel"
	"github.com/h44z/wg-portal/internal/model"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
)

// interfaceManager provides methods to create/update/delete physical WireGuard devices.
type interfaceManager interface {
	GetInterfaces() ([]*model.Interface, error)
	GetInterface(id model.InterfaceIdentifier) (*model.Interface, error)
	CreateInterface(id model.InterfaceIdentifier) error
	DeleteInterface(id model.InterfaceIdentifier) error
	UpdateInterface(cfg *model.Interface) error
	ApplyDefaultConfigs(id model.InterfaceIdentifier) error
}

type importManager interface {
	GetImportableInterfaces() ([]*model.PhysicalInterface, error)
	ImportInterface(cfg *model.PhysicalInterface) error
}

type configFileGenerator interface {
	GetInterfaceConfig(cfg *model.Interface, peers []*model.Peer) (io.Reader, error)
	GetPeerConfig(peer *model.Peer) (io.Reader, error)
}

type peerManager interface {
	GetPeer(id model.PeerIdentifier) (*model.Peer, error)
	GetPeers(device model.InterfaceIdentifier) ([]*model.Peer, error)
	GetPeersForUser(userId model.UserIdentifier) ([]*model.Peer, error)
	SavePeers(peers ...*model.Peer) error
	RemovePeer(peer model.PeerIdentifier) error
}

type ipManager interface {
	GetAllUsedIPs(device model.InterfaceIdentifier) ([]*netlink.Addr, error)
	GetUsedIPs(device model.InterfaceIdentifier, subnetCidr string) ([]*netlink.Addr, error)
	GetFreshIp(device model.InterfaceIdentifier, subnetCidr string, reservedIps ...*netlink.Addr) (*netlink.Addr, error)
	GetFreshIps(device model.InterfaceIdentifier) (string, error)
	ParseIpAddressString(addrStr string) ([]*netlink.Addr, error)
	IpAddressesToString(addresses []netlink.Addr) string
}

type Manager interface {
	keyGenerator
	interfaceManager
	peerManager
	ipManager
	importManager
	configFileGenerator
}

//
// -- Implementations
//

type persistentManager struct {
	*wgCtrlKeyGenerator
	*templateHandler
	*wgCtrlManager
}

func NewPersistentManager(wg lowlevel.WireGuardClient, nl lowlevel.NetlinkClient, store store) (*persistentManager, error) {
	wgManager, err := newWgCtrlManager(wg, nl, store)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to initialize WireGuard manager")
	}

	tplManager, err := newTemplateHandler()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to initialize template manager")
	}

	m := &persistentManager{
		wgCtrlKeyGenerator: &wgCtrlKeyGenerator{},
		wgCtrlManager:      wgManager,
		templateHandler:    tplManager,
	}

	return m, nil
}
