package register

import (
	"context"
	"errors"
	"log/slog"
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

// EmailVerifier is the slice-15 integration point. Declared here so
// register does not import the whole requestemailverification package.
type EmailVerifier interface {
	RequestVerification(ctx context.Context, userID int64) error
}

type Service struct {
	db       *database.DB
	hasher   *auth.PasswordHasher
	jwt      *auth.TokenIssuer
	refresh  *auth.RefreshStore
	patients patients.API
	doctors  contracts.API
	verifier EmailVerifier
}

func NewService(
	db *database.DB,
	hasher *auth.PasswordHasher,
	jwt *auth.TokenIssuer,
	refresh *auth.RefreshStore,
	patientsAPI patients.API,
	doctorsAPI contracts.API,
	verifier EmailVerifier,
) *Service {
	return &Service{
		db: db, hasher: hasher, jwt: jwt, refresh: refresh,
		patients: patientsAPI, doctors: doctorsAPI,
		verifier: verifier,
	}
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
	var userID int64

	err := s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))

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
		userID = row.ID

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

	// Slice 15: after commit, best-effort verification email.
	// SD28: failure here does not roll back registration.
	if s.verifier != nil {
		if err := s.verifier.RequestVerification(ctx, userID); err != nil {
			slog.Warn("verification email failed",
				"user_id", userID, "err", err)
		}
	}

	return resp, nil
}
