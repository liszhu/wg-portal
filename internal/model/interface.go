package model

import "time"

const (
	InterfaceTypeServer InterfaceType = "server"
	InterfaceTypeClient InterfaceType = "client"
	InterfaceTypeAny    InterfaceType = "any"
)

type InterfaceIdentifier string
type InterfaceType string

type Interface struct {
	BaseModel

	// WireGuard specific (for the [interface] section of the config file)

	Identifier InterfaceIdentifier `gorm:"primaryKey"` // device name, for example: wg0
	KeyPair                        // private/public Key of the server interface
	ListenPort int                 // the listening port, for example: 51820

	AddressStr   string // the interface ip addresses, comma separated
	DnsStr       string // the dns server that should be set if the interface is up, comma separated
	DnsSearchStr string // the dns search option string that should be set if the interface is up, will be appended to DnsStr

	Mtu          int    // the device MTU
	FirewallMark int32  // a firewall mark
	RoutingTable string // the routing table

	PreUp    string // action that is executed before the device is up
	PostUp   string // action that is executed after the device is up
	PreDown  string // action that is executed before the device is down
	PostDown string // action that is executed after the device is down

	SaveConfig bool // automatically persist config changes to the wgX.conf file

	// WG Portal specific
	DisplayName string        // a nice display name/ description for the interface
	Type        InterfaceType // the interface type, either InterfaceTypeServer or InterfaceTypeClient
	DriverType  string        // the interface driver type (linux, software, ...)
	Disabled    *time.Time    `gorm:"index"` // flag that specifies if the interface is enabled (up) or not (down)

	// Default settings for the peer, used for new peers, those settings will be published to ConfigOption options of
	// the peer config

	PeerDefNetworkStr          string // the default subnets from which peers will get their IP addresses, comma seperated
	PeerDefDnsStr              string // the default dns server for the peer
	PeerDefDnsSearchStr        string // the default dns search options for the peer
	PeerDefEndpoint            string // the default endpoint for the peer
	PeerDefAllowedIPsStr       string // the default allowed IP string for the peer
	PeerDefMtu                 int    // the default device MTU
	PeerDefPersistentKeepalive int    // the default persistent keep-alive Value
	PeerDefFirewallMark        int32  // default firewall mark
	PeerDefRoutingTable        string // the default routing table

	PeerDefPreUp    string // default action that is executed before the device is up
	PeerDefPostUp   string // default action that is executed after the device is up
	PeerDefPreDown  string // default action that is executed before the device is down
	PeerDefPostDown string // default action that is executed after the device is down

	// Stats will be lazy loaded by the application logic if needed
	Stats *InterfaceStats `gorm:"-"`
}

func (i Interface) IsValid() bool {
	return true // TODO: implement check
}

func (i Interface) IsDisabled() bool {
	return i.Disabled != nil
}

type ImportableInterface struct {
	Interface
	ImportLocation string
	ImportType     string

	Peers []Peer
}

type InterfaceStats struct {
	BaseModel

	Identifier InterfaceIdentifier `gorm:"primaryKey"`

	BytesUpload   uint64
	BytesDownload uint64
}
