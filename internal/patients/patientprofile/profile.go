package patientprofile

import "time"

type Profile struct {
	ID                    int64
	UserID                int64
	FullName              string
	PhoneNumber           string
	DateOfBirth           *time.Time
	InsuranceProvider     *string
	InsurancePolicyNumber *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	DeletedAt             *time.Time
}
