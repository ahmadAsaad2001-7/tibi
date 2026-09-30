package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"

	"tibi/internal/platform/database"
)

type RefreshStore struct {
	db  *database.DB
	ttl time.Duration
}

func NewRefreshStore(db *database.DB, ttl time.Duration) *RefreshStore {
	return &RefreshStore{db: db, ttl: ttl}
}

// Issue generates a random opaque token, stores its SHA-256 hash,
// and returns the raw token to hand to the client.
func (s *RefreshStore) Issue(ctx context.Context, userID int64) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	expiresAt := time.Now().Add(s.ttl)

	q := s.db.Querier(ctx)
	_, err := q.Exec(ctx,
		`INSERT INTO identity_refresh_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)`,
		userID, hex.EncodeToString(hash[:]), expiresAt)
	if err != nil {
		return "", err
	}
	return token, nil
}

// Consume validates a refresh token, revokes it, and returns the user ID.
// Rotation: the caller must issue a new refresh token immediately.
func (s *RefreshStore) Consume(ctx context.Context, token string) (int64, error) {
	hash := sha256.Sum256([]byte(token))
	q := s.db.Querier(ctx)

	var userID int64
	err := q.QueryRow(ctx,
		`UPDATE identity_refresh_tokens
		    SET revoked_at = now()
		  WHERE token_hash = $1
		    AND revoked_at IS NULL
		    AND expires_at > now()
		  RETURNING user_id`,
		hex.EncodeToString(hash[:])).Scan(&userID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func (s *RefreshStore) Revoke(ctx context.Context, token string) error {
	hash := sha256.Sum256([]byte(token))
	q := s.db.Querier(ctx)
	_, err := q.Exec(ctx,
		`UPDATE identity_refresh_tokens
		    SET revoked_at = now()
		  WHERE token_hash = $1 AND revoked_at IS NULL`,
		hex.EncodeToString(hash[:]))
	return err
}
