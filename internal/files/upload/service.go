package upload

import (
	"context"
	cr "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"strings"
	file "tibi/internal/files"
	"tibi/internal/files/upload/db"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
	"tibi/internal/storage"
	"time"
)

type Service struct {
	db      *database.DB
	storage storage.Storage
	clock   func() time.Time
}

func NewService(db *database.DB, s storage.Storage) *Service {
	return &Service{db: db, storage: s, clock: time.Now}
}

type Response struct {
	ID           int64     `json:"id"`
	OriginalName string    `json:"original_name"`
	ContentType  string    `json:"content_type"`
	SizeBytes    int64     `json:"size_bytes"`
	FileURL      string    `json:"file_url"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func (s *Service) Execute(ctx context.Context, cmd Command) (*Response, error) {
	scope := file.Scope(cmd.Scope)
	if !scope.Valid() {
		return nil, httpx.ValidationFailed(map[string]string{"scope": "unknown scope"})
	}
	if !ScopeAllowsMIME(cmd.Scope, cmd.ContentType) {
		return nil, httpx.ValidationFailed(map[string]string{
			"file": "content type not permitted for scope " + cmd.Scope,
		})
	}
	if !IsAllowedSize(cmd.Size) {
		return nil, httpx.ValidationFailed(map[string]string{
			"file": fmt.Sprintf("size must be 1..%d bytes", maxUploadBytes),
		})
	}

	// Read the whole body once so we can compute SHA-256 and pass bytes to
	// storage. 20 MB cap makes this safe. For larger files we would stream
	// to a temp file, hash incrementally, and re-open for storage.
	raw, err := io.ReadAll(io.LimitReader(cmd.Body, maxUploadBytes+1))
	if err != nil {
		return nil, httpx.Internal(err)
	}
	if int64(len(raw)) > maxUploadBytes {
		return nil, httpx.ValidationFailed(map[string]string{
			"file": "exceeds max size",
		})
	}
	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])

	objectKey := buildObjectKey(cmd.UploaderID, scope, cmd.Filename)

	if err := s.storage.Put(ctx, objectKey, strings.NewReader(string(raw)),
		cmd.ContentType, int64(len(raw))); err != nil {
		return nil, httpx.Internal(err)
	}

	var fileID int64
	err = s.db.WithTx(ctx, func(ctx context.Context) error {
		q := db.New(s.db.Querier(ctx))
		row, err := q.InsertFile(ctx, db.InsertFileParams{
			UploaderID:    cmd.UploaderID,
			Scope:         scope,
			ObjectKey:     objectKey,
			OriginalName:  cmd.Filename,
			ContentType:   cmd.ContentType,
			SizeBytes:     int64(len(raw)),
			ContentSha256: hash,
		})
		if err != nil {
			return httpx.Internal(err)
		}
		fileID = row.ID
		return nil
	})
	if err != nil {
		// Best-effort cleanup of the orphan object. Not transactional with
		// the DB row; a sweeper covers missed cases.
		_ = s.storage.Delete(ctx, objectKey)
		return nil, err
	}

	ttl := 24 * time.Hour
	url, err := s.storage.PresignGet(ctx, objectKey, ttl)
	if err != nil {
		return nil, httpx.Internal(err)
	}

	return &Response{
		ID:           fileID,
		OriginalName: cmd.Filename,
		ContentType:  cmd.ContentType,
		SizeBytes:    int64(len(raw)),
		FileURL:      url,
		ExpiresAt:    s.clock().Add(ttl),
	}, nil
}

var cryptoRandRead = cr.Read

func buildObjectKey(uploaderID int64, scope file.Scope, filename string) string {
	ext := path.Ext(filename)
	randBytes := make([]byte, 12)
	_, _ = cryptoRandRead(randBytes)
	return fmt.Sprintf("%s/%d/%s/%s%s",
		strings.ToLower(string(scope)),
		uploaderID,
		time.Now().UTC().Format("2006/01/02"),
		hex.EncodeToString(randBytes),
		ext)
}
