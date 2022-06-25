package wireguard

import (
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/h44z/wg-portal/internal/lowlevel"
	"github.com/h44z/wg-portal/internal/model"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type wgCtrlManager struct {
	mux sync.RWMutex // mutex to synchronize access to maps and external api clients

	// external api clients
	wg lowlevel.WireGuardClient
	nl lowlevel.NetlinkClient

	// optional persistent backend
	store store

	// internal holder of interface configurations
	interfaces map[model.InterfaceIdentifier]*model.Interface
	// internal holder of peer configurations
	peers map[model.InterfaceIdentifier]map[model.PeerIdentifier]*model.Peer
}

func newWgCtrlManager(wg lowlevel.WireGuardClient, nl lowlevel.NetlinkClient, store store) (*wgCtrlManager, error) {
	m := &wgCtrlManager{
		mux:        sync.RWMutex{},
		wg:         wg,
		nl:         nl,
		store:      store,
		interfaces: make(map[model.InterfaceIdentifier]*model.Interface),
		peers:      make(map[model.InterfaceIdentifier]map[model.PeerIdentifier]*model.Peer),
	}

	if err := m.initializeFromStore(); err != nil {
		return nil, errors.WithMessage(err, "failed to initialize manager from store")
	}

	return m, nil
}

func (m *wgCtrlManager) GetInterfaces() ([]*model.Interface, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	interfaces := make([]*model.Interface, 0, len(m.interfaces))
	for _, iface := range m.interfaces {
		interfaces = append(interfaces, iface)
	}
	// Order the interfaces by device name
	sort.Slice(interfaces, func(i, j int) bool {
		return interfaces[i].Identifier < interfaces[j].Identifier
	})

	return interfaces, nil
}

func (m *wgCtrlManager) GetInterface(id model.InterfaceIdentifier) (*model.Interface, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	if !m.deviceExists(id) {
		return nil, ErrInterfaceNotFound
	}

	return m.interfaces[id], nil
}

func (m *wgCtrlManager) CreateInterface(id model.InterfaceIdentifier) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.deviceExists(id) {
		return ErrInterfaceAlreadyExists
	}

	err := m.createLowLevelInterface(id)
	if err != nil {
		return errors.WithMessage(err, "failed to create low level interface")
	}

	newInterface := &model.Interface{Identifier: id, Type: model.InterfaceTypeServer}
	m.interfaces[id] = newInterface
	m.peers[id] = make(map[model.PeerIdentifier]*model.Peer)

	err = m.persistInterface(id, false)
	if err != nil {
		return errors.WithMessage(err, "failed to persist created interface")
	}

	return nil
}

func (m *wgCtrlManager) DeleteInterface(id model.InterfaceIdentifier) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	if !m.deviceExists(id) {
		return ErrInterfaceNotFound
	}

	err := m.nl.LinkDel(&netlink.GenericLink{
		LinkAttrs: netlink.LinkAttrs{
			Name: string(id),
		},
		LinkType: "wireguard",
	})
	if err != nil {
		return errors.WithMessage(err, "failed to delete low level interface")
	}

	err = m.persistInterface(id, true)
	if err != nil {
		return errors.WithMessage(err, "failed to persist deleted interface")
	}

	for peerId := range m.peers[id] {
		err = m.persistPeer(peerId, true)
		if err != nil {
			return errors.WithMessagef(err, "failed to persist deleted peer %s", peerId)
		}
	}

	delete(m.interfaces, id)
	delete(m.peers, id)

	return nil
}

func (m *wgCtrlManager) UpdateInterface(cfg *model.Interface) error {
	if err := m.checkInterface(cfg); err != nil {
		return errors.WithMessage(err, "interface validation failed")
	}

	m.mux.Lock()
	defer m.mux.Unlock()

	if !m.deviceExists(cfg.Identifier) {
		return ErrInterfaceNotFound
	}

	// Update net-link attributes
	link, err := m.nl.LinkByName(string(cfg.Identifier))
	if err != nil {
		return errors.WithMessage(err, "failed to open low level interface")
	}
	if cfg.Mtu != 0 {
		if err := m.nl.LinkSetMTU(link, cfg.Mtu); err != nil {
			return errors.WithMessage(err, "failed to set MTU")
		}
	}
	addresses, err := m.ParseIpAddressString(cfg.AddressStr)
	if err != nil {
		return errors.WithMessage(err, "failed to parse ip address")
	}
	for i := 0; i < len(addresses); i++ {
		var err error
		if i == 0 {
			err = m.nl.AddrReplace(link, addresses[i])
		} else {
			err = m.nl.AddrAdd(link, addresses[i])
		}
		if err != nil {
			return errors.WithMessage(err, "failed to set ip address")
		}
	}

	// Update WireGuard attributes
	pKey, err := wgtypes.NewKey(GetPrivateKeyBytes(cfg.KeyPair))
	if err != nil {
		return errors.WithMessage(err, "failed to parse private key bytes")
	}

	var fwMark *int
	if cfg.FirewallMark != 0 {
		*fwMark = int(cfg.FirewallMark)
	}
	err = m.wg.ConfigureDevice(string(cfg.Identifier), wgtypes.Config{
		PrivateKey:   &pKey,
		ListenPort:   &cfg.ListenPort,
		FirewallMark: fwMark,
	})
	if err != nil {
		return errors.WithMessage(err, "failed to update WireGuard settings")
	}

	// Update link state
	if cfg.Enabled {
		if err := m.nl.LinkSetUp(link); err != nil {
			return errors.WithMessage(err, "failed to enable low level interface")
		}
	} else {
		if err := m.nl.LinkSetDown(link); err != nil {
			return errors.WithMessage(err, "failed to disable low level interface")
		}
	}

	// update internal map
	m.interfaces[cfg.Identifier] = cfg

	err = m.persistInterface(cfg.Identifier, false)
	if err != nil {
		return errors.WithMessage(err, "failed to persist updated interface")
	}

	return nil
}

func (m *wgCtrlManager) ApplyDefaultConfigs(id model.InterfaceIdentifier) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	if !m.deviceExists(id) {
		return ErrInterfaceNotFound
	}

	cfg := m.interfaces[id]

	for p := range m.peers[id] {
		m.peers[id][p].Endpoint.TrySetValue(cfg.PeerDefEndpoint)
		m.peers[id][p].AllowedIPsStr.TrySetValue(cfg.PeerDefAllowedIPsStr)

		m.peers[id][p].Interface.Identifier = cfg.Identifier
		m.peers[id][p].Interface.Type = cfg.Type
		m.peers[id][p].Interface.PublicKey = cfg.KeyPair.PublicKey

		m.peers[id][p].Interface.DnsStr.TrySetValue(cfg.PeerDefDnsStr)
		m.peers[id][p].Interface.Mtu.TrySetValue(cfg.PeerDefMtu)
		m.peers[id][p].Interface.FirewallMark.TrySetValue(cfg.PeerDefFirewallMark)
		m.peers[id][p].Interface.RoutingTable.TrySetValue(cfg.PeerDefRoutingTable)

		m.peers[id][p].Interface.PreUp.TrySetValue(cfg.PeerDefPreUp)
		m.peers[id][p].Interface.PostUp.TrySetValue(cfg.PeerDefPostUp)
		m.peers[id][p].Interface.PreDown.TrySetValue(cfg.PeerDefPreDown)
		m.peers[id][p].Interface.PostDown.TrySetValue(cfg.PeerDefPostDown)

		err := m.persistPeer(m.peers[id][p].Identifier, false)
		if err != nil {
			return errors.Wrapf(err, "failed to persist peer defaults to %s", m.peers[id][p].Identifier)
		}
	}

	return nil
}

func (m *wgCtrlManager) GetPeers(interfaceId model.InterfaceIdentifier) ([]*model.Peer, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	if !m.deviceExists(interfaceId) {
		return nil, ErrInterfaceNotFound
	}

	peers := make([]*model.Peer, 0, len(m.peers[interfaceId]))
	for i := range m.peers[interfaceId] {
		peers = append(peers, m.peers[interfaceId][i])
	}

	return peers, nil
}

func (m *wgCtrlManager) GetPeersForUser(userId model.UserIdentifier) ([]*model.Peer, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	var peers []*model.Peer
	for _, interfacePeers := range m.peers {
		for _, peer := range interfacePeers {
			if peer.UserIdentifier == userId {
				peers = append(peers, peer)
			}
		}
	}

	return peers, nil
}

func (m *wgCtrlManager) SavePeers(peers ...*model.Peer) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	for _, peer := range peers {
		if err := m.checkPeer(peer); err != nil {
			return errors.WithMessage(err, "peer validation failed")
		}

		deviceId := peer.Interface.Identifier
		if !m.deviceExists(deviceId) {
			return ErrInterfaceNotFound
		}
		deviceConfig := m.interfaces[deviceId]

		m.peers[deviceId][peer.Identifier] = peer

		if peer.Temporary != nil {
			continue // do not persist temporary peer to database or perform any WireGuard actions
		}

		wgPeer, err := m.getWireGuardPeerConfig(deviceConfig.Type, peer)
		if err != nil {
			return errors.WithMessagef(err, "could not generate WireGuard peer configuration for %s", peer.Identifier)
		}

		err = m.wg.ConfigureDevice(string(deviceId), wgtypes.Config{Peers: []wgtypes.PeerConfig{wgPeer}})
		if err != nil {
			return errors.Wrapf(err, "could not save peer %s to WireGuard device %s", peer.Identifier, deviceId)
		}

		err = m.persistPeer(peer.Identifier, false)
		if err != nil {
			return errors.Wrapf(err, "failed to persist updated peer %s", peer.Identifier)
		}
	}

	return nil
}

func (m *wgCtrlManager) RemovePeer(id model.PeerIdentifier) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	if !m.peerExists(id) {
		return ErrPeerNotFound
	}

	peer, _ := m.getPeer(id)
	deviceId := peer.Interface.Identifier

	publicKey, err := wgtypes.ParseKey(peer.KeyPair.PublicKey)
	if err != nil {
		return errors.WithMessage(err, "invalid public key")
	}

	wgPeer := wgtypes.PeerConfig{
		PublicKey: publicKey,
		Remove:    true,
	}

	err = m.wg.ConfigureDevice(string(deviceId), wgtypes.Config{Peers: []wgtypes.PeerConfig{wgPeer}})
	if err != nil {
		return errors.WithMessage(err, "could not remove peer from WireGuard interface")
	}

	err = m.persistPeer(id, true)
	if err != nil {
		return errors.WithMessage(err, "failed to persist deleted peer")
	}

	delete(m.peers[deviceId], id)

	return nil
}

func (m *wgCtrlManager) GetImportableInterfaces() ([]*model.ImportableInterface, error) {
	devices, err := m.wg.Devices()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get WireGuard device list")
	}

	m.mux.RLock()
	defer m.mux.RUnlock()

	interfaces := make([]*model.ImportableInterface, len(devices))
	for d, device := range devices {
		if _, exists := m.interfaces[model.InterfaceIdentifier(device.Name)]; exists {
			continue // interface already managed, skip
		}

		cfg, err := m.convertWireGuardInterface(devices[d])
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to convert WireGuard interface %s", device.Name)
		}
		cfg.ImportLocation = "interface" // TODO: interface, file, ... ?
		cfg.ImportType = "unknown"

		cfg.Peers = make([]model.Peer, 0, len(device.Peers))

		for p, peer := range device.Peers {
			err := m.convertWireGuardPeer(&device.Peers[p], cfg)
			if err != nil {
				return nil, errors.WithMessagef(err, "failed to convert WireGuard peer %s from %s",
					peer.PublicKey.String(), device.Name)
			}
		}
	}

	return interfaces, nil
}

func (m *wgCtrlManager) ImportInterface(cfg *model.ImportableInterface) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	newInterface := &cfg.Interface
	if err := m.checkInterface(newInterface); err != nil {
		return errors.WithMessage(err, "interface validation failed")
	}

	m.interfaces[newInterface.Identifier] = newInterface
	m.peers[newInterface.Identifier] = make(map[model.PeerIdentifier]*model.Peer)

	err := m.persistInterface(newInterface.Identifier, false)
	if err != nil {
		return errors.WithMessage(err, "failed to persist imported interface")
	}

	for i, peer := range cfg.Peers {
		if err := m.checkPeer(&cfg.Peers[i]); err != nil {
			return errors.WithMessage(err, "peer validation failed")
		}

		m.peers[newInterface.Identifier][peer.Identifier] = &cfg.Peers[i]

		err = m.persistPeer(peer.Identifier, false)
		if err != nil {
			return errors.Wrapf(err, "failed to persist imported peer %s", peer.Identifier)
		}
	}

	return nil
}

//
// -- Helpers
//

func (m *wgCtrlManager) initializeFromStore() error {
	if m.store == nil {
		return nil // no store, nothing to do
	}

	interfaceIds, err := m.store.GetInterfaceIds()
	if err != nil {
		return errors.WithMessage(err, "failed to get available interfaces")
	}

	interfaces, err := m.store.GetAllInterfaces(interfaceIds...)
	if err != nil {
		return errors.WithMessage(err, "failed to get all interfaces")
	}

	for tmpCfg, tmpPeers := range interfaces {
		cfg := tmpCfg
		peers := tmpPeers
		m.interfaces[cfg.Identifier] = &cfg
		if _, ok := m.peers[cfg.Identifier]; !ok {
			m.peers[cfg.Identifier] = make(map[model.PeerIdentifier]*model.Peer)
		}
		for p, peer := range peers {
			m.peers[cfg.Identifier][peer.Identifier] = &peers[p]
		}
	}

	return nil
}

func (m *wgCtrlManager) createLowLevelInterface(id model.InterfaceIdentifier) error {
	link := &netlink.GenericLink{
		LinkAttrs: netlink.LinkAttrs{
			Name: string(id),
		},
		LinkType: "wireguard",
	}
	err := m.nl.LinkAdd(link)
	if err != nil {
		return errors.Wrapf(err, "failed to create netlink interface")
	}

	if err := m.nl.LinkSetUp(link); err != nil {
		return errors.Wrapf(err, "failed to enable netlink interface")
	}

	return nil
}

func (m *wgCtrlManager) deviceExists(id model.InterfaceIdentifier) bool {
	if _, ok := m.interfaces[id]; ok {
		return true
	}
	return false
}

func (m *wgCtrlManager) persistInterface(id model.InterfaceIdentifier, delete bool) error {
	if m.store == nil {
		return nil // nothing to do
	}

	var err error
	if delete {
		err = m.store.DeleteInterface(id)
	} else {
		err = m.store.SaveInterface(m.interfaces[id])
	}
	if err != nil {
		return errors.Wrapf(err, "failed to persist interface")
	}

	return nil
}

func (m *wgCtrlManager) peerExists(id model.PeerIdentifier) bool {
	for _, peers := range m.peers {
		if _, ok := peers[id]; ok {
			return true
		}
	}

	return false
}

func (m *wgCtrlManager) persistPeer(id model.PeerIdentifier, delete bool) error {
	if m.store == nil {
		return nil // nothing to do
	}

	var peer *model.Peer
	for _, peers := range m.peers {
		if p, ok := peers[id]; ok {
			peer = p
			break
		}
	}

	var err error
	if delete {
		err = m.store.DeletePeer(id)
	} else {
		err = m.store.SavePeer(peer)
	}
	if err != nil {
		return errors.Wrapf(err, "failed to persist peer %s", id)
	}

	return nil
}

func (m *wgCtrlManager) getPeer(id model.PeerIdentifier) (*model.Peer, error) {
	for _, peers := range m.peers {
		if _, ok := peers[id]; ok {
			return peers[id], nil
		}
	}

	return nil, errors.New("peer not found")
}

func (m *wgCtrlManager) convertWireGuardInterface(device *wgtypes.Device) (*model.ImportableInterface, error) {
	cfg := &model.ImportableInterface{}

	cfg.Interface.Identifier = model.InterfaceIdentifier(device.Name)
	cfg.Interface.Type = model.InterfaceTypeServer // default assume that the imported device is a server device
	cfg.Interface.FirewallMark = int32(device.FirewallMark)
	cfg.Interface.KeyPair = model.KeyPair{
		PrivateKey: device.PrivateKey.String(),
		PublicKey:  device.PublicKey.String(),
	}
	cfg.Interface.ListenPort = device.ListenPort
	cfg.Interface.DriverType = device.Type.String()

	lowLevelInterface, err := m.nl.LinkByName(device.Name)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get low level interface for %s", device.Name)
	}
	cfg.Interface.Mtu = lowLevelInterface.Attrs().MTU
	ipAddresses, err := m.nl.AddrList(lowLevelInterface)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get low level addresses for %s", device.Name)
	}
	cfg.Interface.AddressStr = m.IpAddressesToString(ipAddresses)

	return cfg, nil
}

func (m *wgCtrlManager) convertWireGuardPeer(peer *wgtypes.Peer, dev *model.ImportableInterface) error {
	peerCfg := model.Peer{}
	peerCfg.Identifier = model.PeerIdentifier(peer.PublicKey.String())
	peerCfg.KeyPair = model.KeyPair{
		PublicKey: peer.PublicKey.String(),
	}
	peerCfg.DisplayName = "Autodetected Peer (" + peer.PublicKey.String()[0:8] + ")"
	if peer.Endpoint != nil {
		peerCfg.Endpoint = model.NewStringConfigOption(peer.Endpoint.String(), true)
	}
	if peer.PresharedKey != (wgtypes.Key{}) {
		peerCfg.PresharedKey = model.PreSharedKey(peer.PresharedKey.String())
	}
	allowedIPs := make([]string, len(peer.AllowedIPs)) // use allowed IP's as the peer IP's
	for i, ip := range peer.AllowedIPs {
		allowedIPs[i] = ip.String()
	}
	peerCfg.AllowedIPsStr = model.NewStringConfigOption(strings.Join(allowedIPs, ","), true)
	peerCfg.PersistentKeepalive = model.NewIntConfigOption(int(peer.PersistentKeepaliveInterval.Seconds()), true)

	peerCfg.Interface = &model.PeerInterfaceConfig{
		Identifier: dev.Interface.Identifier,
		AddressStr: model.NewStringConfigOption(dev.AddressStr, true), // todo: correct?
		DnsStr:     model.NewStringConfigOption(dev.DnsStr, true),
		Mtu:        model.NewIntConfigOption(dev.Mtu, true),
	}

	dev.Peers = append(dev.Peers, peerCfg)

	return nil
}

func (m *wgCtrlManager) checkInterface(cfg *model.Interface) error {
	if cfg == nil {
		return errors.New("interface config must not be nil")
	}
	if cfg.Identifier == "" {
		return errors.New("missing interface identifier")
	}
	if cfg.Type == "" {
		return errors.New("missing interface type")
	}

	return nil
}

func (m *wgCtrlManager) checkPeer(cfg *model.Peer) error {
	if cfg == nil {
		return errors.New("peer config must not be nil")
	}
	if cfg.Identifier == "" {
		return errors.New("missing peer identifier")
	}
	if cfg.Interface == nil {
		return errors.New("missing peer interface")
	}
	if cfg.Interface.Identifier == "" {
		return errors.New("missing peer interface identifier")
	}

	return nil
}

func (m *wgCtrlManager) getWireGuardPeerConfig(devType model.InterfaceType, cfg *model.Peer) (wgtypes.PeerConfig, error) {
	publicKey, err := wgtypes.ParseKey(cfg.KeyPair.PublicKey)
	if err != nil {
		return wgtypes.PeerConfig{}, errors.WithMessage(err, "invalid public key for peer")
	}

	var presharedKey *wgtypes.Key
	if tmpPresharedKey, err := wgtypes.ParseKey(string(cfg.PresharedKey)); err == nil {
		presharedKey = &tmpPresharedKey
	}

	var endpoint *net.UDPAddr
	if cfg.Endpoint.Value != "" && devType == model.InterfaceTypeClient {
		addr, err := net.ResolveUDPAddr("udp", cfg.Endpoint.GetValue())
		if err == nil {
			endpoint = addr
		}
	}

	var keepAlive *time.Duration
	if cfg.PersistentKeepalive.GetValue() != 0 {
		keepAliveDuration := time.Duration(cfg.PersistentKeepalive.GetValue()) * time.Second
		keepAlive = &keepAliveDuration
	}

	allowedIPs := make([]net.IPNet, 0)
	var peerAllowedIPs []*netlink.Addr
	switch devType {
	case model.InterfaceTypeClient:
		peerAllowedIPs, err = m.ParseIpAddressString(cfg.AllowedIPsStr.GetValue())
		if err != nil {
			return wgtypes.PeerConfig{}, errors.WithMessage(err, "failed to parse allowed IP's")
		}
	case model.InterfaceTypeServer:
		peerAllowedIPs, err = m.ParseIpAddressString(cfg.AllowedIPsStr.GetValue())
		if err != nil {
			return wgtypes.PeerConfig{}, errors.WithMessage(err, "failed to parse allowed IP's")
		}
		peerExtraAllowedIPs, err := m.ParseIpAddressString(cfg.ExtraAllowedIPsStr)
		if err != nil {
			return wgtypes.PeerConfig{}, errors.WithMessage(err, "failed to parse extra allowed IP's")
		}

		peerAllowedIPs = append(peerAllowedIPs, peerExtraAllowedIPs...)
	}
	for _, ip := range peerAllowedIPs {
		allowedIPs = append(allowedIPs, *ip.IPNet)
	}

	wgPeer := wgtypes.PeerConfig{
		PublicKey:                   publicKey,
		Remove:                      false,
		UpdateOnly:                  true,
		PresharedKey:                presharedKey,
		Endpoint:                    endpoint,
		PersistentKeepaliveInterval: keepAlive,
		ReplaceAllowedIPs:           true,
		AllowedIPs:                  allowedIPs,
	}

	return wgPeer, nil
}
