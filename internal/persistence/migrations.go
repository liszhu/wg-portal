package persistence

import "github.com/h44z/wg-portal/internal/model"

func (d *Database) Migrate() error {
	d.db.AutoMigrate(&model.Interface{}, &model.User{})
	d.db.AutoMigrate(&model.Peer{})
	return nil
}
