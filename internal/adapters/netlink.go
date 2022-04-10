package adapters

import (
	"sync"

	"github.com/h44z/wg-portal/internal/domain"
	"github.com/vishvananda/netlink"
)

type netlinkRepository struct {
	mux sync.RWMutex
}

func NewNetlinkRepository() (*netlinkRepository, error) {
	return &netlinkRepository{
		mux: sync.RWMutex{},
	}, nil
}

func (n *netlinkRepository) convertLink(link *domain.NetLink) netlink.Link {
	return nil
}

func (n *netlinkRepository) convertLibLink(link netlink.Link) *domain.NetLink {
	return nil
}

func (n *netlinkRepository) convertAddr(link *domain.LinkAddr) *netlink.Addr {
	return &netlink.Addr{
		IPNet: link.IPNet,
		Label: link.Label,
	}
}

func (n *netlinkRepository) convertLibAddr(link *netlink.Addr) *domain.LinkAddr {
	return &domain.LinkAddr{
		IPNet: link.IPNet,
		Label: link.Label,
	}
}

func (n *netlinkRepository) Create(link *domain.NetLink) error {
	l := n.convertLink(link)
	return netlink.LinkAdd(l)
}

func (n *netlinkRepository) Delete(link *domain.NetLink) error {
	l := n.convertLink(link)
	return netlink.LinkDel(l)
}

func (n *netlinkRepository) Get(name string) (*domain.NetLink, error) {
	l, err := netlink.LinkByName(name)
	if err != nil {
		return nil, err
	}

	return n.convertLibLink(l), nil
}

func (n *netlinkRepository) Up(link *domain.NetLink) error {
	l := n.convertLink(link)
	return netlink.LinkSetUp(l)
}

func (n *netlinkRepository) Down(link *domain.NetLink) error {
	l := n.convertLink(link)
	return netlink.LinkSetDown(l)
}

func (n *netlinkRepository) SetMTU(link *domain.NetLink, mtu int) error {
	l := n.convertLink(link)
	return netlink.LinkSetMTU(l, mtu)
}

func (n *netlinkRepository) ReplaceAddr(link *domain.NetLink, addr *domain.LinkAddr) error {
	l := n.convertLink(link)
	a := n.convertAddr(addr)
	return netlink.AddrReplace(l, a)
}

func (n *netlinkRepository) AddAddr(link *domain.NetLink, addr *domain.LinkAddr) error {
	l := n.convertLink(link)
	a := n.convertAddr(addr)
	return netlink.AddrAdd(l, a)
}

func (n *netlinkRepository) ListAddr(link *domain.NetLink) ([]*domain.LinkAddr, error) {
	l := n.convertLink(link)

	listIPv4, err := netlink.AddrList(l, netlink.FAMILY_V4)
	if err != nil {
		return nil, err
	}

	listIPv6, err := netlink.AddrList(l, netlink.FAMILY_V6)
	if err != nil {
		return nil, err
	}

	ipAddresses := make([]*domain.LinkAddr, 0, len(listIPv4)+len(listIPv6))
	for i := range listIPv4 { // first add IPv4 addresses
		ipAddresses = append(ipAddresses, n.convertLibAddr(&listIPv4[i]))
	}
	for i := range listIPv6 { // next add IPv6 addresses
		ipAddresses = append(ipAddresses, n.convertLibAddr(&listIPv6[i]))
	}

	return ipAddresses, nil
}
