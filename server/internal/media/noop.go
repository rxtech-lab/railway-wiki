package media

import (
	"context"
	"errors"

	"github.com/rxtech-lab/railway-wiki/internal/api"
)

// ErrNotConfigured indicates media uploads are unavailable.
var ErrNotConfigured = errors.New("media upload storage is not configured")

// NoopPresigner is used when object storage is not configured; every call fails.
type NoopPresigner struct{}

// Presign always returns ErrNotConfigured.
func (NoopPresigner) Presign(context.Context, PresignInput) (*api.MediaUploadTarget, error) {
	return nil, ErrNotConfigured
}
