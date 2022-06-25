package model

import (
	"database/sql"
	"time"
)

type BaseModel struct {
	CreatedBy  string
	UpdatedBy  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DisabledAt sql.NullTime
}

type PrivateString string

func (PrivateString) MarshalJSON() ([]byte, error) {
	return []byte(`""`), nil
}

func (PrivateString) String() string {
	return ""
}
