package wireguard

import "fmt"

var (
	ErrInterfaceNotFound      = fmt.Errorf("no such interface")
	ErrPeerNotFound           = fmt.Errorf("no such peer")
	ErrInterfaceAlreadyExists = fmt.Errorf("interface already exists")
	ErrPeerAlreadyExists      = fmt.Errorf("peer already exists")
	ErrInterfaceDisabled      = fmt.Errorf("interface disabled")
	ErrPeerDisabled           = fmt.Errorf("interface disabled")
)
