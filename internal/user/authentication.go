package user

import (
	"crypto/subtle"

	"github.com/h44z/wg-portal/internal/model"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

func (p *persistentManager) PlaintextAuthentication(userId model.UserIdentifier, plainPassword string) error {
	user, err := p.GetUser(userId)
	if err != nil {
		return errors.WithMessagef(err, "unable to load user %s", userId)
	}

	if user.Source == model.UserSourceOauth {
		return errors.New("password authentication unavailable")
	}

	if user.Password == "" {
		return errors.New("password authentication unavailable")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(plainPassword)); err != nil {
		return errors.WithMessage(err, "invalid password")
	}

	return nil
}

func (p *persistentManager) HashedAuthentication(userId model.UserIdentifier, hashedPassword string) error {
	user, err := p.GetUser(userId)
	if err != nil {
		return errors.WithMessagef(err, "unable to load user %s", userId)
	}

	if user.Source == model.UserSourceOauth {
		return errors.New("password authentication unavailable")
	}

	if user.Password == "" {
		return errors.New("password authentication unavailable")
	}

	if subtle.ConstantTimeCompare([]byte(user.Password), []byte(hashedPassword)) != 1 {
		return errors.New("invalid password")
	}

	return nil
}

func (p *persistentManager) HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", errors.WithMessage(err, "failed to hash password")
	}

	return string(hash), nil
}
