package model

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type IpAddress net.IP

// Value returns value as a integer.
func (ip *IpAddress) Value() (driver.Value, error) {
	return []byte(*ip), nil
}

// Scan scans a string value into CIDR.
func (ip *IpAddress) Scan(value any) error {
	switch v := value.(type) {
	case []byte:
		*ip = v
		return nil
	default:
		return errors.New(fmt.Sprintf("failed to scan value: %v", v))
	}
}

// GormDataType gorm common data type.
func (ip *IpAddress) GormDataType() string {
	return "ipaddr"
}

// GormDBDataType gorm db data type.
func (ip *IpAddress) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql", "sqlserver", "sqlite":
		return "VARBINARY(16)" // IPv6 has 16 bytes
	case "postgres":
		return "BYTEA"
	}
	return ""
}

type IpMask struct {
	net.IPMask
}

type Cidr struct {
	Ip        IpAddress `gorm:"primaryKey"`
	NetLength int       `gorm:"primaryKey;column:net_len"`
	Bits      int       `gorm:"column:bits"`
}

func CidrFromString(str string) (Cidr, error) {
	ip, ipNet, err := net.ParseCIDR(str)
	if err != nil {
		return Cidr{}, err
	}
	ones, bits := ipNet.Mask.Size()
	return Cidr{
		Ip:        IpAddress(ip),
		NetLength: ones,
		Bits:      bits,
	}, nil
}

func MustCidrFromString(str string) Cidr {
	cidr, err := CidrFromString(str)
	if err != nil {
		panic(err)
	}
	return cidr
}

func CidrFromIpNet(ipNet net.IPNet) Cidr {
	ones, bits := ipNet.Mask.Size()
	return Cidr{
		Ip:        IpAddress(ipNet.IP),
		NetLength: ones,
		Bits:      bits,
	}
}

func CidrFromNetlinkAddr(addr netlink.Addr) Cidr {
	ones, bits := addr.Mask.Size()
	return Cidr{
		Ip:        IpAddress(addr.IP),
		NetLength: ones,
		Bits:      bits,
	}
}

func (c *Cidr) IpNet() *net.IPNet {
	return &net.IPNet{
		IP:   net.IP(c.Ip),
		Mask: net.CIDRMask(c.NetLength, c.Bits),
	}
}

func (c *Cidr) NetlinkAddr() *netlink.Addr {
	return &netlink.Addr{
		IPNet: c.IpNet(),
	}
}

func (c *Cidr) String() string {
	return c.IpNet().String()
}

func (c *Cidr) IsV4() bool {
	if c.Bits == 32 {
		return true
	}

	return false
}

// BroadcastAddr returns the last address in the given network (for IPv6), or the broadcast address.
func (c *Cidr) BroadcastAddr() {
	// TODO
	/*// The golang net package doesn't make it easy to calculate the broadcast address. :(
	var broadcast = net.IPv6zero
	var mask = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff} // ensure that mask also has 16 bytes (also for IPv4)
	if len(n.Mask) == 4 {
		for i := 0; i < 4; i++ {
			mask[12+i] = n.Mask[i]
		}
	} else {
		for i := 0; i < 16; i++ {
			mask[i] = n.Mask[i]
		}
	}
	for i := 0; i < len(n.IP); i++ {
		broadcast[i] = n.IP[i] | ^mask[i]
	}
	return &netlink.Addr{
		IPNet: &net.IPNet{IP: broadcast, Mask: n.Mask},
	}*/
}

func (c *Cidr) NextAddr() Cidr {
	var next = Cidr{
		Ip:        make([]byte, len(c.Ip)),
		NetLength: c.NetLength,
		Bits:      c.Bits,
	}
	copy(next.Ip, c.Ip)

	// increase ip
	for j := len(next.Ip) - 1; j >= 0; j-- {
		next.Ip[j]++
		if next.Ip[j] > 0 {
			break
		}
	}

	return next
}
