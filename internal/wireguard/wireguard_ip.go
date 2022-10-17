package wireguard

import (
	"bytes"
	"fmt"
	"net"
	"sort"

	"github.com/h44z/wg-portal/internal/model"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
)

func (m *wgCtrlManager) GetAllUsedIPs(id model.InterfaceIdentifier) ([]*netlink.Addr, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	if !m.deviceExists(id) {
		return nil, ErrInterfaceNotFound
	}

	var usedAddresses []*netlink.Addr
	for _, peer := range m.peers[id] {
		addresses, err := m.ParseIpAddressString(peer.Interface.AddressStr.GetValue())
		if err != nil {
			return nil, errors.WithMessagef(err, "unable to parse addresses of peer %s", peer.Identifier)
		}

		usedAddresses = append(usedAddresses, addresses...)
	}

	sort.Slice(usedAddresses, func(i, j int) bool {
		return bytes.Compare(usedAddresses[i].IP, usedAddresses[j].IP) < 0
	})

	return usedAddresses, nil
}

func (m *wgCtrlManager) GetUsedIPs(id model.InterfaceIdentifier, subnetCidr string) ([]*netlink.Addr, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	if !m.deviceExists(id) {
		return nil, ErrInterfaceNotFound
	}

	subnet, err := parseCIDR(subnetCidr)
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to parse subnet addresses")
	}

	var usedAddresses []*netlink.Addr
	for _, peer := range m.peers[id] {
		addresses, err := m.ParseIpAddressString(peer.Interface.AddressStr.GetValue())
		if err != nil {
			return nil, errors.WithMessagef(err, "unable to parse addresses of peer %s", peer.Identifier)
		}

		for _, address := range addresses {
			if subnet.Contains(address.IP) {
				usedAddresses = append(usedAddresses, address)
			}
		}
	}

	sort.Slice(usedAddresses, func(i, j int) bool {
		return bytes.Compare(usedAddresses[i].IP, usedAddresses[j].IP) < 0
	})

	return usedAddresses, nil
}

func (m *wgCtrlManager) GetFreshIp(id model.InterfaceIdentifier, subnetCidr string, reservedIps ...*netlink.Addr) (*netlink.Addr, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	if !m.deviceExists(id) {
		return nil, ErrInterfaceNotFound
	}

	subnet, err := parseCIDR(subnetCidr)
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to parse subnet addresses")
	}
	subnet.IP = subnet.IP.Mask(subnet.Mask).To16() // use network address here (this is always the lowest possible address)
	isV4 := isV4(subnet)

	usedIPs, err := m.GetUsedIPs(id, subnetCidr) // highest IP is at the end of the array
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to load used IP addresses")
	}

	// these two addresses are not usable
	broadcastAddr := broadcastAddr(subnet)
	networkAddr := subnet.IP

	// start with the lowest IP and check all others
	ip := &netlink.Addr{
		IPNet: &net.IPNet{IP: subnet.IP.Mask(subnet.Mask).To16(), Mask: subnet.Mask}, // copy network address
	}

	for ; subnet.Contains(ip.IP); increaseIP(ip) {
		if bytes.Compare(ip.IP, networkAddr) == 0 {
			continue
		}
		if isV4 && bytes.Compare(ip.IP, broadcastAddr.IP) == 0 {
			continue
		}

		isReserved := false
		for _, reservedIp := range reservedIps {
			if bytes.Compare(ip.IP, reservedIp.IP) == 0 {
				isReserved = true
				break
			}
		}
		if isReserved {
			continue
		}

		ok := true
		for _, r := range usedIPs {
			if bytes.Compare(ip.IP, r.IP) == 0 {
				ok = false
				break
			}
		}

		if ok {
			return ip, nil
		}
	}

	return nil, errors.New("ip range exceeded")
}

func (m *wgCtrlManager) GetFreshIps(id model.InterfaceIdentifier) (string, error) {
	m.mux.RLock()
	defer m.mux.RUnlock()

	if !m.deviceExists(id) {
		return "", ErrInterfaceNotFound
	}

	interfaceIps, err := m.ParseIpAddressString(m.interfaces[id].AddressStr)
	if err != nil {
		return "", err
	}

	freshIPs := make([]netlink.Addr, len(interfaceIps))
	for i, interfaceIp := range interfaceIps {
		ip, err := m.GetFreshIp(id, interfaceIp.String(), interfaceIp)
		if err != nil {
			return "", fmt.Errorf("failed to get fresh IP for %s: %w", interfaceIp.String(), err)
		}
		freshIPs[i] = *ip
	}

	return m.IpAddressesToString(freshIPs), nil
}
