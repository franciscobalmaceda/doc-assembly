package dto

import "time"

// GalleryAssetResponse represents a gallery asset in API responses.
type GalleryAssetResponse struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"createdAt"`
}

// GalleryListResponse holds a paginated list of gallery assets.
type GalleryListResponse struct {
	Items   []*GalleryAssetResponse `json:"items"`
	Total   int                     `json:"total"`
	Page    int                     `json:"page"`
	PerPage int                     `json:"perPage"`
}

// GalleryURLResponse holds a resolved URL for a gallery asset.
type GalleryURLResponse struct {
	URL string `json:"url"`
}
