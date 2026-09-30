package register

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"tibi/internal/doctors/contracts"
	"tibi/internal/identity/register/db"
	"tibi/internal/identity/user"
	patients "tibi/internal/patients/contracts"
	"tibi/internal/platform/auth"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

type Service struct {
	db       *database.DB
	hasher   *auth.PasswordHasher
	jwt      *auth.TokenIssuer
	refresh  *auth.RefreshStore
	patients patients.API
	doctors  contracts.API
}

func NewService(db *database.DB, hasher *auth.PasswordHasher, jwt *auth.TokenIssuer, refresh *auth.RefreshStore, patientsAPI patients.API, doctorsAPI contracts.API) *Service {
	return &Service{db: db, hasher: hasher, jwt: jwt, refresh: refresh, patients: patientsAPI, doctors: doctorsAPI}
}

type Response struct {
	User struct {
		ID              int64     `json:"id"`
		Email           string    `json:"email"`
		Role            string    `json:"role"`
		ProfileImageURL *string   `json:"profile_image_url"`
		CreatedAt       time.Time `json:"created_at"`
	} `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (s *Service) Execute(ctx context.Context, cmd Command) (*Response, error) {
	var resp *Response

	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

		// Uniqueness check inside the tx.
		if _, err := q.FindUserByEmail(ctx, cmd.Email); err == nil {
			return httpx.AlreadyExists("email already registered")
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return httpx.Internal(err)
		}

		hash, err := s.hasher.Hash(cmd.Password)
		if err != nil {
			return httpx.Internal(err)
		}

		role := user.Role(cmd.Role)
		if role == user.RoleDoctor {
			role = user.RolePendingDoctor
		}

		row, err := q.InsertUser(ctx, db.InsertUserParams{
			Email:        cmd.Email,
			PasswordHash: hash,
			Role:         role,
		})
		if err != nil {
			return httpx.Internal(err)
		}

		// Cross-module call. Shares this transaction.
		switch role {
		case user.RolePatient:
			if _, err := s.patients.CreatePatientProfile(ctx, patients.CreatePatientProfileInput{
				UserID:      row.ID,
				FullName:    cmd.FullName,
				PhoneNumber: cmd.PhoneNumber,
			}); err != nil {
				return err
			}
		case user.RolePendingDoctor:
			if _, err := s.doctors.CreateDoctorProfile(ctx, contracts.CreateDoctorProfileInput{
				UserID:   row.ID,
				FullName: cmd.FullName,
			}); err != nil {
				return err
			}
		}

		access, err := s.jwt.IssueAccess(row.ID, string(role))
		if err != nil {
			return httpx.Internal(err)
		}
		refresh, err := s.refresh.Issue(ctx, row.ID)
		if err != nil {
			return httpx.Internal(err)
		}

		resp = &Response{
			AccessToken:  access,
			RefreshToken: refresh,
			ExpiresIn:    s.jwt.ExpiresIn(),
		}
		resp.User.ID = row.ID
		resp.User.Email = cmd.Email
		resp.User.Role = string(role)
		resp.User.CreatedAt = row.CreatedAt.Time
		return nil
	})

	if err != nil {
		return nil, err
	}
	return resp, nil
}
