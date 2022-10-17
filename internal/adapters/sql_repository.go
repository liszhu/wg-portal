package adapters

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/h44z/wg-portal/internal/model"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("record not found")

type sqlRepo struct {
	db *gorm.DB
}

func NewSqlRepository(db *gorm.DB) *sqlRepo {
	repo := &sqlRepo{
		db: db,
	}

	return repo
}

// region interfaces

func (r *sqlRepo) GetInterface(ctx context.Context, id model.InterfaceIdentifier) (*model.Interface, error) {
	var in model.Interface

	err := r.db.WithContext(ctx).First(&in, id).Error

	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &in, nil
}

func (r *sqlRepo) GetInterfaceAndPeers(ctx context.Context, id model.InterfaceIdentifier) (*model.Interface, []model.Peer, error) {
	in, err := r.GetInterface(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load interface: %w", err)
	}

	peers, err := r.GetInterfacePeers(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load peers: %w", err)
	}

	return in, peers, nil
}

func (r *sqlRepo) GetAllInterfaces(ctx context.Context) ([]model.Interface, error) {
	var interfaces []model.Interface

	err := r.db.WithContext(ctx).Find(&interfaces).Error
	if err != nil {
		return nil, err
	}

	return interfaces, nil
}

func (r *sqlRepo) FindInterfaces(ctx context.Context, search string) ([]model.Interface, error) {
	var users []model.Interface

	searchValue := "%" + strings.ToLower(search) + "%"
	err := r.db.WithContext(ctx).
		Where("identifier LIKE ?", searchValue).
		Or("display_name LIKE ?", searchValue).
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (r *sqlRepo) SaveInterface(ctx context.Context, id model.InterfaceIdentifier, updateFunc func(in *model.Interface) (*model.Interface, error)) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		in, err := r.getOrCreateInterface(tx, id)
		if err != nil {
			return err // return any error will roll back
		}

		in, err = updateFunc(in)
		if err != nil {
			return err
		}

		err = r.upsertInterface(tx, in)
		if err != nil {
			return err
		}

		// return nil will commit the whole transaction
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *sqlRepo) getOrCreateInterface(tx *gorm.DB, id model.InterfaceIdentifier) (*model.Interface, error) {
	var in model.Interface

	// interfaceDefaults will be applied to newly created interface records
	interfaceDefaults := model.Interface{
		BaseModel: model.BaseModel{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	err := tx.Attrs(interfaceDefaults).FirstOrCreate(&in, id).Error
	if err != nil {
		return nil, err
	}

	return &in, nil
}

func (r *sqlRepo) upsertInterface(tx *gorm.DB, in *model.Interface) error {
	err := tx.Save(in).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *sqlRepo) DeleteInterface(ctx context.Context, id model.InterfaceIdentifier) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := r.db.WithContext(ctx).Where("interface_identifier = ?", id).Delete(&model.Peer{}).Error
		if err != nil {
			return err
		}

		err = r.db.WithContext(ctx).Delete(&model.Interface{}, id).Error
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// endregion interfaces

// region peers

func (r *sqlRepo) GetInterfacePeers(ctx context.Context, id model.InterfaceIdentifier) ([]model.Peer, error) {
	var peers []model.Peer

	err := r.db.WithContext(ctx).Where("interface_identifier = ?", id).Find(&peers).Error
	if err != nil {
		return nil, err
	}

	return peers, nil
}

func (r *sqlRepo) FindInterfacePeers(ctx context.Context, id model.InterfaceIdentifier, search string) ([]model.Peer, error) {
	var peers []model.Peer

	searchValue := "%" + strings.ToLower(search) + "%"
	err := r.db.WithContext(ctx).Where("interface_identifier = ?", id).
		Where("identifier LIKE ?", searchValue).
		Or("display_name LIKE ?", searchValue).
		Or("iface_address_str_v LIKE ?", searchValue).
		Find(&peers).Error
	if err != nil {
		return nil, err
	}

	return peers, nil
}

func (r *sqlRepo) GetUserPeers(ctx context.Context, id model.UserIdentifier) ([]model.Peer, error) {
	var peers []model.Peer

	err := r.db.WithContext(ctx).Where("user_identifier = ?", id).Find(&peers).Error
	if err != nil {
		return nil, err
	}

	return peers, nil
}

func (r *sqlRepo) FindUserPeers(ctx context.Context, id model.UserIdentifier, search string) ([]model.Peer, error) {
	var peers []model.Peer

	searchValue := "%" + strings.ToLower(search) + "%"
	err := r.db.WithContext(ctx).Where("user_identifier = ?", id).
		Where("identifier LIKE ?", searchValue).
		Or("display_name LIKE ?", searchValue).
		Or("iface_address_str_v LIKE ?", searchValue).
		Find(&peers).Error
	if err != nil {
		return nil, err
	}

	return peers, nil
}

func (r *sqlRepo) SavePeer(ctx context.Context, id model.PeerIdentifier, updateFunc func(in *model.Peer) (*model.Peer, error)) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		peer, err := r.getOrCreatePeer(tx, id)
		if err != nil {
			return err // return any error will roll back
		}

		peer, err = updateFunc(peer)
		if err != nil {
			return err
		}

		err = r.upsertPeer(tx, peer)
		if err != nil {
			return err
		}

		// return nil will commit the whole transaction
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *sqlRepo) getOrCreatePeer(tx *gorm.DB, id model.PeerIdentifier) (*model.Peer, error) {
	var peer model.Peer

	// interfaceDefaults will be applied to newly created interface records
	interfaceDefaults := model.Peer{
		BaseModel: model.BaseModel{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	err := tx.Attrs(interfaceDefaults).FirstOrCreate(&peer, id).Error
	if err != nil {
		return nil, err
	}

	return &peer, nil
}

func (r *sqlRepo) upsertPeer(tx *gorm.DB, peer *model.Peer) error {
	err := tx.Save(peer).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *sqlRepo) DeletePeer(ctx context.Context, id model.PeerIdentifier) error {
	err := r.db.WithContext(ctx).Delete(&model.Peer{}, id).Error
	if err != nil {
		return err
	}

	return nil
}

// endregion peers

// region users

func (r *sqlRepo) GetUser(ctx context.Context, id model.UserIdentifier) (*model.User, error) {
	var user model.User

	err := r.db.WithContext(ctx).First(&user, id).Error

	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *sqlRepo) GetAllUsers(ctx context.Context) ([]model.User, error) {
	var users []model.User

	err := r.db.WithContext(ctx).Find(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (r *sqlRepo) FindUsers(ctx context.Context, search string) ([]model.User, error) {
	var users []model.User

	searchValue := "%" + strings.ToLower(search) + "%"
	err := r.db.WithContext(ctx).
		Where("identifier LIKE ?", searchValue).
		Or("firstname LIKE ?", searchValue).
		Or("lastname LIKE ?", searchValue).
		Or("email LIKE ?", searchValue).
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (r *sqlRepo) SaveUser(ctx context.Context, id model.UserIdentifier, updateFunc func(u *model.User) (*model.User, error)) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user, err := r.getOrCreateUser(tx, id)
		if err != nil {
			return err // return any error will roll back
		}

		user, err = updateFunc(user)
		if err != nil {
			return err
		}

		err = r.upsertUser(tx, user)
		if err != nil {
			return err
		}

		// return nil will commit the whole transaction
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *sqlRepo) DeleteUser(ctx context.Context, id model.UserIdentifier) error {
	err := r.db.WithContext(ctx).Delete(&model.User{}, id).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *sqlRepo) getOrCreateUser(tx *gorm.DB, id model.UserIdentifier) (*model.User, error) {
	var user model.User

	// userDefaults will be applied to newly created user records
	userDefaults := model.User{
		BaseModel: model.BaseModel{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Source:  model.UserSourceDatabase,
		IsAdmin: false,
	}

	err := tx.Attrs(userDefaults).FirstOrCreate(&user, id).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *sqlRepo) upsertUser(tx *gorm.DB, user *model.User) error {
	err := tx.Save(user).Error
	if err != nil {
		return err
	}

	return nil
}

// endregion users
