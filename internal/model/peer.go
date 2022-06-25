package model

import "time"

type PeerIdentifier string

type PeerInterfaceConfig struct {
	Identifier InterfaceIdentifier `gorm:"index;column:iface_identifier"` // the interface identifier
	Type       InterfaceType       `gorm:"column:iface_type"`             // the interface type
	PublicKey  string              `gorm:"column:iface_pubkey"`           // the interface public key

	AddressStr   StringConfigOption `gorm:"embedded;embeddedPrefix:iface_address_str_"`    // the interface ip addresses, comma separated
	DnsStr       StringConfigOption `gorm:"embedded;embeddedPrefix:iface_dns_str_"`        // the dns server that should be set if the interface is up, comma separated
	DnsSearchStr StringConfigOption `gorm:"embedded;embeddedPrefix:iface_dns_search_str_"` // the dns search option string that should be set if the interface is up, will be appended to DnsStr
	Mtu          IntConfigOption    `gorm:"embedded;embeddedPrefix:iface_mtu_"`            // the device MTU
	FirewallMark Int32ConfigOption  `gorm:"embedded;embeddedPrefix:iface_firewall_mark_"`  // a firewall mark
	RoutingTable StringConfigOption `gorm:"embedded;embeddedPrefix:iface_routing_table_"`  // the routing table

	PreUp    StringConfigOption `gorm:"embedded;embeddedPrefix:iface_pre_up_"`    // action that is executed before the device is up
	PostUp   StringConfigOption `gorm:"embedded;embeddedPrefix:iface_post_up_"`   // action that is executed after the device is up
	PreDown  StringConfigOption `gorm:"embedded;embeddedPrefix:iface_pre_down_"`  // action that is executed before the device is down
	PostDown StringConfigOption `gorm:"embedded;embeddedPrefix:iface_post_down_"` // action that is executed after the device is down
}

type Peer struct {
	BaseModel

	// WireGuard specific (for the [peer] section of the config file)

	Endpoint            StringConfigOption `gorm:"embedded;embeddedPrefix:endpoint_"`        // the endpoint address
	AllowedIPsStr       StringConfigOption `gorm:"embedded;embeddedPrefix:allowed_ips_str_"` // all allowed ip subnets, comma seperated
	ExtraAllowedIPsStr  string             // all allowed ip subnets on the server side, comma seperated
	KeyPair                                // private/public Key of the peer
	PresharedKey        PreSharedKey       // the pre-shared Key of the peer
	PersistentKeepalive IntConfigOption    `gorm:"embedded;embeddedPrefix:persistent_keep_alive_"` // the persistent keep-alive interval

	// WG Portal specific

	DisplayName    string         // a nice display name/ description for the peer
	Identifier     PeerIdentifier `gorm:"primaryKey"` // peer unique identifier
	UserIdentifier UserIdentifier `gorm:"index"`      // the owner
	Temporary      *time.Time     `gorm:"-"`          // is this a temporary peer (only prepared, but never saved to db)
	Disabled       *time.Time     `gorm:"index"`      // if this field is set, the peer is disabled

	// Interface settings for the peer, used to generate the [interface] section in the peer config file
	Interface *PeerInterfaceConfig `gorm:"embedded"`

	// Stats of the peer, will be lazy loaded by the application logic if needed
	Stats *PeerStats `gorm:"-"`
}

func (p Peer) IsDisabled() bool {
	return p.Disabled != nil
}

type PeerStats struct {
	BaseModel

	Identifier PeerIdentifier `gorm:"primaryKey"`

	LastHandshake       *time.Time
	LastEndpointAddress string

	LastPingTime         *time.Time
	LastPingValueMs      int
	IsConnected          bool
	ConnectionInitiated  *time.Time
	ConnectionTerminated *time.Time

	BytesUpload   uint64
	BytesDownload uint64
}
