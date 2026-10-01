package user

import (
	"errors"
	"testing"
)

func TestRoleValid(t *testing.T) {
	for _, role := range []Role{RolePatient, RoleDoctor, RolePendingDoctor, RoleAdmin} {
		if !role.Valid() {
			t.Fatalf("%s should be valid", role)
		}
	}
	if Role("Nurse").Valid() {
		t.Fatal("unknown role reported valid")
	}
}

func TestNew(t *testing.T) {
	u := New("ada@example.com", "hash", RolePatient)
	if u.Email != "ada@example.com" || u.PasswordHash != "hash" || u.Role != RolePatient {
		t.Fatalf("user = %+v", u)
	}
}

func TestPromoteToDoctor(t *testing.T) {
	pending := New("ada@example.com", "hash", RolePendingDoctor)
	if err := pending.PromoteToDoctor(); err != nil {
		t.Fatal(err)
	}
	if pending.Role != RoleDoctor {
		t.Fatalf("role = %s", pending.Role)
	}
	if err := pending.PromoteToDoctor(); !errors.Is(err, ErrInvalidRoleTransition) {
		t.Fatalf("second promote err = %v", err)
	}

	patient := New("pat@example.com", "hash", RolePatient)
	if err := patient.PromoteToDoctor(); !errors.Is(err, ErrInvalidRoleTransition) {
		t.Fatalf("patient promote err = %v", err)
	}
}
