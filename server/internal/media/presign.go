// Package media provides presigned upload targets for media object storage.
package media

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/rxtech-lab/railway-wiki/internal/api"
)

// PresignInput describes a requested upload.
type PresignInput struct {
	FileName    string
	ContentType string
	MediaType   *string
	SizeBytes   *int64
}

// Presigner produces presigned upload targets.
type Presigner interface {
	Presign(ctx context.Context, in PresignInput) (*api.MediaUploadTarget, error)
}

// objectKey derives a collision-free storage key from the file name.
func objectKey(fileName string) string {
	base := path.Base(filepathClean(fileName))
	if base == "" || base == "." || base == "/" {
		base = "file"
	}
	return fmt.Sprintf("media/%s/%s", uuid.NewString(), base)
}

// filepathClean sanitizes a client-supplied file name to its last path element.
func filepathClean(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimSpace(name)
	return name
}

func expiry(ttl time.Duration, now time.Time) time.Time {
	return now.Add(ttl)
}
