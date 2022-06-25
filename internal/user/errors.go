package user

import "fmt"

var (
	ErrNotFound      = fmt.Errorf("no such user")
	ErrAlreadyExists = fmt.Errorf("user already exists")
	ErrDisabled      = fmt.Errorf("user disabled")
)
