package accesschecker

import (
	"context"
	"errors"
	"sync"

	"tibi/internal/files/getmetadata"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
)

// ScopeRule is implemented by modules that own files of a given scope.
// Registered at startup (see Register) so the access checker can delegate
// the "may this user read this file?" question to the module that
// understands the resource the file is embedded in.
type ScopeRule interface {
	AuthorizeFileRead(ctx context.Context, userID int64, fileID int64, uploaderID int64) (bool, error)
}

type Registry struct {
	db    *database.DB
	mu    sync.RWMutex
	rules map[string]ScopeRule
}

func New(db *database.DB) *Registry {
	return &Registry{db: db, rules: map[string]ScopeRule{}}
}

// Register adds a scope rule. Startup only; not thread-safe if called while
// requests are in flight (there is no reason to).
func (r *Registry) Register(scope string, rule ScopeRule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules[scope] = rule
}

// Authorize returns nil if the caller may read the file, otherwise a 404
// (anti-enumeration) as httpx.Error-compatible error.
func (r *Registry) Authorize(ctx context.Context, userID, fileID int64) error {
	meta, err := getmetadata.NewService(r.db).Execute(ctx, fileID)
	if err != nil {
		if errors.Is(err, getmetadata.ErrNotFound) {
			return httpx.NotFound("file not found")
		}
		return httpx.Internal(err)
	}
	// Uploader always allowed.
	if meta.UploaderID == userID {
		return nil
	}
	r.mu.RLock()
	rule, ok := r.rules[meta.Scope]
	r.mu.RUnlock()
	if !ok {
		return httpx.NotFound("file not found")
	}
	allowed, err := rule.AuthorizeFileRead(ctx, userID, fileID, meta.UploaderID)
	if err != nil {
		return err
	}
	if !allowed {
		return httpx.NotFound("file not found")
	}
	return nil
}
