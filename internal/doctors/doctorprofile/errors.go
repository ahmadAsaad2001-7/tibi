package doctorprofile

import (
	"errors"
	"strings"
)

var (
	ErrAlreadyVerified      = errors.New("profile is already verified")
	ErrAlreadyPendingReview = errors.New("profile is already pending review")
)

type IncompleteProfileError struct {
	Missing []string
}

func (e *IncompleteProfileError) Error() string {
	return "profile incomplete: missing " + strings.Join(e.Missing, ", ")
}

func (e *IncompleteProfileError) Fields() map[string]string {
	m := make(map[string]string, len(e.Missing))
	for _, f := range e.Missing {
		m[f] = "required before submission"
	}
	return m
}
