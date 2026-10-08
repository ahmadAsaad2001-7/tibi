package onetimetoken

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"
)

var (
	ErrTokenExpired  = errors.New("token has expired")
	ErrTokenConsumed = errors.New("token has already been used")
)

// Token holds the raw form (returned to the caller, sent by email) and the
// hash form (stored in DB). The struct does not know which table it belongs
// to; that is the slice's concern.
type Token struct {
	Raw       string
	Hash      string
	ExpiresAt time.Time
}

// New generates a 32-byte URL-safe random token and its SHA-256 hash.
func New(ttl time.Duration, now time.Time) (Token, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return Token{}, err
	}
	rawStr := base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(rawStr))
	return Token{
		Raw:       rawStr,
		Hash:      hex.EncodeToString(sum[:]),
		ExpiresAt: now.Add(ttl),
	}, nil
}

// HashToken re-derives the hash for a token the client supplied.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}