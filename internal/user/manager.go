package user

import (
	"errors"
	"fmt"
	"sync"

	"github.com/h44z/wg-portal/internal/authentication"
	"github.com/h44z/wg-portal/internal/model"
)

type loader interface {
	GetActiveUser(id model.UserIdentifier) (*model.User, error)
	GetUser(id model.UserIdentifier) (*model.User, error)
	GetActiveUsers() ([]*model.User, error)
	GetAllUsers() ([]*model.User, error)
	GetFilteredUsers(filter Filter) ([]*model.User, error)
}

type updater interface {
	CreateUser(user *model.User) error
	UpdateUser(user *model.User) error
	DeleteUser(identifier model.UserIdentifier) error
}

// Filter can be used to filter users. If this function returns true, the given user is included in the result.
type Filter func(user *model.User) bool

type Manager interface {
	loader
	updater
	authentication.PlainAuthenticator
	authentication.PasswordHasher
}

type persistentManager struct {
	mux sync.RWMutex // mutex to synchronize access to maps and external api clients

	store store

	// internal holder of user objects
	users map[model.UserIdentifier]*model.User
}

func NewPersistentManager(store store) (*persistentManager, error) {
	mgr := &persistentManager{
		store: store,

		users: make(map[model.UserIdentifier]*model.User),
	}

	if err := mgr.initializeFromStore(); err != nil {
		return nil, fmt.Errorf("failed to initialize manager from store: %w", err)
	}

	return mgr, nil
}

func (p *persistentManager) GetUser(id model.UserIdentifier) (*model.User, error) {
	p.mux.RLock()
	defer p.mux.RUnlock()

	if !p.userExists(id) {
		return nil, ErrNotFound
	}

	return p.users[id], nil
}

func (p *persistentManager) GetActiveUser(id model.UserIdentifier) (*model.User, error) {
	p.mux.RLock()
	defer p.mux.RUnlock()

	if !p.userExists(id) {
		return nil, ErrNotFound
	}

	if !p.userIsEnabled(id) {
		return nil, ErrDisabled
	}

	return p.users[id], nil
}

func (p *persistentManager) GetActiveUsers() ([]*model.User, error) {
	p.mux.RLock()
	defer p.mux.RUnlock()

	users := make([]*model.User, 0, len(p.users))
	for _, user := range p.users {
		if !user.DeletedAt.Valid {
			users = append(users, user)
		}
	}

	return users, nil
}

func (p *persistentManager) GetAllUsers() ([]*model.User, error) {
	p.mux.RLock()
	defer p.mux.RUnlock()

	users := make([]*model.User, 0, len(p.users))
	for _, user := range p.users {
		users = append(users, user)
	}

	return users, nil
}

func (p *persistentManager) GetFilteredUsers(filter Filter) ([]*model.User, error) {
	p.mux.RLock()
	defer p.mux.RUnlock()

	users := make([]*model.User, 0, len(p.users))
	for _, user := range p.users {
		if filter == nil || filter(user) {
			users = append(users, user)
		}
	}

	return users, nil
}

func (p *persistentManager) CreateUser(user *model.User) error {
	if err := p.checkUser(user); err != nil {
		return fmt.Errorf("user validation failed: %w", err)
	}

	p.mux.Lock()
	defer p.mux.Unlock()

	if p.userExists(user.Identifier) {
		return ErrAlreadyExists
	}

	p.users[user.Identifier] = user

	err := p.persistUser(user.Identifier, false)
	if err != nil {
		return fmt.Errorf("failed to persist created user: %w", err)
	}

	return nil
}

func (p *persistentManager) UpdateUser(user *model.User) error {
	if err := p.checkUser(user); err != nil {
		return fmt.Errorf("user validation failed: %w", err)
	}

	p.mux.Lock()
	defer p.mux.Unlock()

	if !p.userExists(user.Identifier) {
		return ErrNotFound
	}

	p.users[user.Identifier] = user

	err := p.persistUser(user.Identifier, false)
	if err != nil {
		return fmt.Errorf("failed to persist updated user: %w", err)
	}

	return nil
}

func (p *persistentManager) DeleteUser(id model.UserIdentifier) error {
	p.mux.Lock()
	defer p.mux.Unlock()
	if !p.userExists(id) {
		return ErrNotFound
	}

	err := p.persistUser(id, true)
	if err != nil {
		return fmt.Errorf("failed to persist deleted user: %w", err)
	}

	delete(p.users, id)

	return nil
}

//
// -- Helpers
//

func (p *persistentManager) initializeFromStore() error {
	if p.store == nil {
		return nil // no store, nothing to do
	}

	users, err := p.store.GetUsersUnscoped()
	if err != nil {
		return fmt.Errorf("failed to get all users: %w", err)
	}

	for _, tmpUser := range users {
		user := tmpUser
		p.users[user.Identifier] = &user
	}

	return nil
}

func (p *persistentManager) userExists(id model.UserIdentifier) bool {
	if _, ok := p.users[id]; ok {
		return true
	}
	return false
}

func (p *persistentManager) userIsEnabled(id model.UserIdentifier) bool {
	if user, ok := p.users[id]; ok && !user.DeletedAt.Valid {
		return true
	}
	return false
}

func (p *persistentManager) persistUser(id model.UserIdentifier, delete bool) error {
	if p.store == nil {
		return nil // nothing to do
	}

	var err error
	if delete {
		err = p.store.DeleteUser(id)
	} else {
		err = p.store.SaveUser(p.users[id])
	}
	if err != nil {
		return fmt.Errorf("failed to persist user: %w", err)
	}

	return nil
}

func (p *persistentManager) checkUser(user *model.User) error {
	if user == nil {
		return errors.New("user must not be nil")
	}
	if user.Identifier == "" {
		return errors.New("missing user identifier")
	}
	if user.Source == "" {
		return errors.New("missing user source")
	}

	return nil
}
