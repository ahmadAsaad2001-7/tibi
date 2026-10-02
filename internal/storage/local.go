package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// LocalFS is a dev-only driver that stores objects on the local filesystem
// and signs presigned URLs with an HMAC. Do not use in production; it has
// no replication, no durability guarantees, and no CDN.
type LocalFS struct {
	root    string
	signKey []byte
	baseURL string // e.g. "http://localhost:8080/files"
}

func NewLocalFS(root, baseURL, signKey string) (*LocalFS, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", root, err)
	}
	return &LocalFS{root: root, baseURL: baseURL, signKey: []byte(signKey)}, nil
}

func (l *LocalFS) Put(ctx context.Context, key string, body io.Reader, contentType string, size int64) error {
	path := filepath.Join(l.root, key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, body); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

func (l *LocalFS) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	expires := time.Now().Add(ttl).Unix()
	sig := l.sign(key, expires)
	return fmt.Sprintf("%s/%s?expires=%d&sig=%s",
		l.baseURL, url.PathEscape(key), expires, sig), nil
}

func (l *LocalFS) Delete(ctx context.Context, key string) error {
	path := filepath.Join(l.root, key)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (l *LocalFS) sign(key string, expires int64) string {
	mac := hmac.New(sha256.New, l.signKey)
	fmt.Fprintf(mac, "%s:%d", key, expires)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// VerifySignature validates a presigned URL's signature. Called by the
// dev-only serving handler.
func (l *LocalFS) VerifySignature(key string, expires int64, sig string) bool {
	if time.Now().Unix() > expires {
		return false
	}
	expected := l.sign(key, expires)
	return hmac.Equal([]byte(expected), []byte(sig))
}

// AbsPath returns the filesystem path for a stored object key.
func (l *LocalFS) AbsPath(key string) string {
	return filepath.Join(l.root, filepath.FromSlash(key))
}
