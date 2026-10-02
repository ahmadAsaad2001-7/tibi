package queueentry

import "errors"

var (
	ErrInvalidTransition = errors.New("invalid queue status transition")
	ErrAlreadyCheckedIn  = errors.New("consultation is already checked in")
	ErrAnotherActive     = errors.New("another entry in this session is already called or in progress")
)
