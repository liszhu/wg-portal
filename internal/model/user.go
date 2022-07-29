package model

import (
	"time"
)

const (
	UserSourceLdap     UserSource = "ldap"  // LDAP / ActiveDirectory
	UserSourceDatabase UserSource = "db"    // sqlite / mysql database
	UserSourceOauth    UserSource = "oauth" // oauth / open id connect
)

type UserIdentifier string

type UserSource string

// User is the user model that gets linked to peer entries, by default an empty user model with only the email address is created
type User struct {
	BaseModel

	// required fields
	Identifier UserIdentifier `gorm:"primaryKey;column:identifier"`
	Email      string         `form:"email" binding:"required,email"`
	Source     UserSource
	IsAdmin    bool

	// optional fields
	Firstname  string `form:"firstname" binding:"omitempty"`
	Lastname   string `form:"lastname" binding:"omitempty"`
	Phone      string `form:"phone" binding:"omitempty"`
	Department string `form:"department" binding:"omitempty"`
	Notes      string `form:"notes" binding:"omitempty"`

	// optional, integrated password authentication
	Password PrivateString `form:"password" binding:"omitempty"`
	Disabled *time.Time    `gorm:"index;column:disabled"` // if this field is set, the peer is disabled
}

func (u User) IsDisabled() bool {
	return u.Disabled != nil
}
