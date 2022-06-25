package user

import "github.com/h44z/wg-portal/internal/model"

type store interface {
	GetUsers() ([]model.User, error)
	SaveUser(user *model.User) error
	DeleteUser(identifier model.UserIdentifier) error
}
