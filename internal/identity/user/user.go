package user

import "time"

type Role string

const (
	RolePatient       Role = "Patient"
	RoleDoctor        Role = "Doctor"
	RolePendingDoctor Role = "PendingDoctor"
	RoleAdmin         Role = "Admin"
)

func (r Role) Valid() bool {
	switch r {
	case RolePatient, RoleDoctor, RolePendingDoctor, RoleAdmin:
		return true
	}
	return false
}

type User struct {
	ID              int64
	Email           string
	PasswordHash    string
	Role            Role
	ProfileImageURL *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}

func New(email, passwordHash string, role Role) *User {
	return &User{
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
	}
}

// PromoteToDoctor is the state transition used when a PendingDoctor
// completes verification. Domain method so the invariant lives here.
func (u *User) PromoteToDoctor() error {
	if u.Role != RolePendingDoctor {
		return ErrInvalidRoleTransition
	}
	u.Role = RoleDoctor
	return nil
}
