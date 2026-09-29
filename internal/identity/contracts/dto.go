package contracts

import "time"

// UserResponse is the shape returned to other modules and to HTTP handlers.
// It deliberately does not carry PasswordHash.
type UserResponse struct {
	ID              int64     `json:"id"`
	Email           string    `json:"email"`
	Role            string    `json:"role"`
	ProfileImageURL *string   `json:"profile_image_url"`
	CreatedAt       time.Time `json:"created_at"`
}

// Role constants exported for use in RequireRole calls.
const (
	RolePatient       = "Patient"
	RoleDoctor        = "Doctor"
	RolePendingDoctor = "PendingDoctor"
	RoleAdmin         = "Admin"
)
