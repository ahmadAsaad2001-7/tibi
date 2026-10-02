package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3 is a driver for AWS S3 and S3-compatible services (MinIO, R2, etc.).
// Uses minio-go because it works against any S3-compatible endpoint.
type S3 struct {
	client    *minio.Client
	bucket    string
	publicURL string // optional; if set, presigned URLs are rewritten to this host
}

type S3Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	Region    string
	PublicURL string
}

func NewS3(cfg S3Config) (*S3, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("minio new: %w", err)
	}
	return &S3{client: client, bucket: cfg.Bucket, publicURL: cfg.PublicURL}, nil
}

func (s *S3) Put(ctx context.Context, key string, body io.Reader, contentType string, size int64) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("s3 put: %w", err)
	}
	return nil
}

func (s *S3) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, ttl, url.Values{})
	if err != nil {
		return "", fmt.Errorf("s3 presign: %w", err)
	}
	if s.publicURL != "" {
		u.Host = s.publicURL
	}
	return u.String(), nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("s3 delete: %w", err)
	}
	return nil
}
