package entity

import (
	"errors"
	"time"
)

// GalleryAsset represents an image stored in the workspace gallery.
type GalleryAsset struct {
	ID          string
	TenantID    string
	WorkspaceID string
	Key         string
	Filename    string
	ContentType string
	Size        int64
	SHA256      string
	CreatedBy   string
	CreatedAt   time.Time
}

// Gallery errors.
var (
	ErrGalleryAssetNotFound      = errors.New("gallery asset not found")
	ErrGalleryInvalidContentType = errors.New("invalid content type: only images are allowed")
	ErrGalleryFileTooLarge       = errors.New("file too large: maximum size is 10MB")
)
