package authentication

import "github.com/h44z/wg-portal/internal/model"

type PlainAuthenticator interface {
	PlaintextAuthentication(userId model.UserIdentifier, plainPassword string) error
	HashedAuthentication(userId model.UserIdentifier, hashedPassword string) error
}
