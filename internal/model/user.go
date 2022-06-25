package model

import (
	"time"

	"gorm.io/gorm"
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
	// required fields
	Identifier UserIdentifier `gorm:"primaryKey"`
	Email      string         `form:"email" binding:"required,email"`
	Source     UserSource
	IsAdmin    bool

	// optional fields
	Firstname  string `form:"firstname" binding:"omitempty"`
	Lastname   string `form:"lastname" binding:"omitempty"`
	Phone      string `form:"phone" binding:"omitempty"`
	Department string `form:"department" binding:"omitempty"`

	// optional, integrated password authentication
	Password PrivateString `form:"password" binding:"omitempty"`

	// database internal fields
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index" json:",omitempty"` // used as a "deactivated" flag
}
