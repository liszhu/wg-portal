package wireguard

import (
	"github.com/h44z/wg-portal/internal/model"
)

type store interface {
	GetAvailableInterfaces() ([]model.InterfaceIdentifier, error)

	GetAllInterfaces(interfaceIdentifiers ...model.InterfaceIdentifier) (map[model.Interface][]model.Peer, error)
	GetInterface(identifier model.InterfaceIdentifier) (model.Interface, []model.Peer, error)

	SaveInterface(cfg *model.Interface) error
	SavePeer(peer *model.Peer) error

	DeleteInterface(identifier model.InterfaceIdentifier) error
	DeletePeer(peer model.PeerIdentifier) error
}
