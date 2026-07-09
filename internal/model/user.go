package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	Status       string     `json:"status"` // active, inactive, invited
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`

	// Joined data (not always populated)
	Roles []Role `json:"roles,omitempty"`
}

func (u User) FullName() string {
	return u.FirstName + " " + u.LastName
}

// UserFilter narrows UserRepository.List results. RoleSlug is the slug of a
// role the user must hold; an empty string means "no role filter".
type UserFilter struct {
	RoleSlug string
	Search   string
	Status   string
}
