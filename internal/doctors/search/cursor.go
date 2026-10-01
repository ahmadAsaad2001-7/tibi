package search

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

type cursor struct {
	Rating    *string `json:"r,omitempty"`
	Fee       *string `json:"f,omitempty"`
	CreatedAt *string `json:"c,omitempty"`
	ID        int64   `json:"i"`
}

func encodeCursor(c cursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(s string) (cursor, error) {
	if s == "" {
		return cursor{}, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return cursor{}, errors.New("invalid cursor")
	}
	var c cursor
	if err := json.Unmarshal(b, &c); err != nil {
		return cursor{}, errors.New("invalid cursor")
	}
	return c, nil
}
