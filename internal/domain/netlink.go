package domain

import "net"

type NetLink struct {
	Name string
	MTU  int

	HardwareAddr net.HardwareAddr
	Flags        net.Flags
}

type LinkAddr struct {
	*net.IPNet
	Label string // optional label
}
