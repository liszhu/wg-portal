package persistence

import (
	"github.com/h44z/wg-portal/internal/model"
	"github.com/pkg/errors"
	"gorm.io/gorm/clause"
)

func (d *Database) GetAvailableInterfaces() ([]model.InterfaceIdentifier, error) {
	var interfaces []model.Interface
	if err := d.db.Select("identifier").Find(&interfaces).Error; err != nil {
		return nil, errors.WithMessage(err, "unable to find interfaces")
	}

	interfaceIds := make([]model.InterfaceIdentifier, len(interfaces))
	for i := range interfaces {
		interfaceIds[i] = interfaces[i].Identifier
	}

	return interfaceIds, nil
}

func (d *Database) GetAllInterfaces(ids ...model.InterfaceIdentifier) (map[model.Interface][]model.Peer, error) {
	var interfaces []model.Interface
	if err := d.db.Where("identifier IN ?", ids).Find(&interfaces).Error; err != nil {
		return nil, errors.WithMessage(err, "unable to find interfaces")
	}

	interfaceMap := make(map[model.Interface][]model.Peer, len(interfaces))
	for i := range interfaces {
		var peers []model.Peer
		if err := d.db.Where("iface_identifier = ?", interfaces[i].Identifier).Find(&peers).Error; err != nil {
			return nil, errors.WithMessagef(err, "unable to find peers for %s", interfaces[i].Identifier)
		}
		interfaceMap[interfaces[i]] = peers
	}

	return interfaceMap, nil
}

func (d *Database) GetInterface(id model.InterfaceIdentifier) (model.Interface, []model.Peer, error) {
	var iface model.Interface
	if err := d.db.First(&iface, id).Error; err != nil {
		return model.Interface{}, nil, errors.WithMessage(err, "unable to find interface")
	}

	var peers []model.Peer
	if err := d.db.Where("identifier = ?", id).Find(&peers).Error; err != nil {
		return model.Interface{}, nil, errors.WithMessage(err, "unable to find peers")
	}

	return iface, peers, nil
}

func (d *Database) SaveInterface(cfg *model.Interface) error {
	d.db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(cfg)
	return nil
}

func (d *Database) DeleteInterface(id model.InterfaceIdentifier) error {
	if err := d.db.Delete(&model.Interface{}, id).Error; err != nil {
		return errors.WithMessage(err, "unable to delete interface")
	}

	return nil
}

func (d *Database) SavePeer(peer *model.Peer) error {
	if err := d.db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(peer).Error; err != nil {
		return errors.WithMessage(err, "unable to save peer")
	}

	return nil
}

func (d *Database) DeletePeer(peerId model.PeerIdentifier) error {
	if err := d.db.Delete(&model.Peer{}, peerId).Error; err != nil {
		return errors.WithMessage(err, "unable to delete peer")
	}
	return nil
}
