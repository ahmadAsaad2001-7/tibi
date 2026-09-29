package user

import "errors"

var (
	ErrEmailTaken            = errors.New("email already registered")
	ErrInvalidRoleTransition = errors.New("invalid role transition")
)
