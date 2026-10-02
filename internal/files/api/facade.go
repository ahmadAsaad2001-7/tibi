package api

import (
	"context"
	"errors"

	"tibi/internal/files/contracts"
	"tibi/internal/files/getmetadata"
	"tibi/internal/storage"
	"time"
)

type API struct {
	meta    *getmetadata.Service
	storage storage.Storage
}

func New(meta *getmetadata.Service, s storage.Storage) *API {
	return &API{meta: meta, storage: s}
}

var _ contracts.API = (*API)(nil)

func (a *API) Get(ctx context.Context, fileID int64) (*contracts.FileMetadata, error) {
	m, err := a.meta.Execute(ctx, fileID)
	if err != nil {
		if errors.Is(err, getmetadata.ErrNotFound) {
			return nil, contracts.ErrNotFound
		}
		return nil, err
	}
	return &contracts.FileMetadata{
		ID:           m.ID,
		UploaderID:   m.UploaderID,
		Scope:        m.Scope,
		OriginalName: m.OriginalName,
		ContentType:  m.ContentType,
		SizeBytes:    m.SizeBytes,
	}, nil
}

func (a *API) PresignGet(ctx context.Context, fileID int64, ttlSeconds int) (string, error) {
	m, err := a.meta.Execute(ctx, fileID)
	if err != nil {
		return "", err
	}
	return a.storage.PresignGet(ctx, m.ObjectKey, time.Duration(ttlSeconds)*time.Second)
}
