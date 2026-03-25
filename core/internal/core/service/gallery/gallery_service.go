package gallery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/rendis/doc-assembly/core/internal/core/entity"
	"github.com/rendis/doc-assembly/core/internal/core/port"
	galleryuc "github.com/rendis/doc-assembly/core/internal/core/usecase/gallery"
)

const (
	maxFileSize        = 10 * 1024 * 1024 // 10 MB
	galleryKeyPrefix   = "gallery"
)

// Service implements the GalleryUseCase input port.
type Service struct {
	adapter   port.StorageAdapter
	repo      port.GalleryRepository
	publicURL string
}

// New creates a new gallery service.
func New(adapter port.StorageAdapter, repo port.GalleryRepository, publicURL string) galleryuc.GalleryUseCase {
	return &Service{
		adapter:   adapter,
		repo:      repo,
		publicURL: strings.TrimRight(publicURL, "/"),
	}
}

// ListAssets returns a paginated list of workspace gallery assets.
func (s *Service) ListAssets(ctx context.Context, cmd galleryuc.ListAssetsCmd) (*galleryuc.AssetsPage, error) {
	page, perPage := normalizePagination(cmd.Page, cmd.PerPage)

	assets, total, err := s.repo.List(ctx, cmd.WorkspaceID, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("listing gallery assets: %w", err)
	}

	return &galleryuc.AssetsPage{
		Assets:  assets,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}

// SearchAssets returns assets matching the search query.
func (s *Service) SearchAssets(ctx context.Context, cmd galleryuc.SearchAssetsCmd) (*galleryuc.AssetsPage, error) {
	page, perPage := normalizePagination(cmd.Page, cmd.PerPage)

	assets, total, err := s.repo.Search(ctx, cmd.WorkspaceID, cmd.Query, page, perPage)
	if err != nil {
		return nil, fmt.Errorf("searching gallery assets: %w", err)
	}

	return &galleryuc.AssetsPage{
		Assets:  assets,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}

// UploadAsset uploads a new image file and registers it in the gallery.
// Returns the existing asset if an identical file (same SHA-256) already exists.
func (s *Service) UploadAsset(ctx context.Context, cmd galleryuc.UploadAssetCmd) (*entity.GalleryAsset, error) {
	if !strings.HasPrefix(cmd.ContentType, "image/") {
		return nil, entity.ErrGalleryInvalidContentType
	}
	if int64(len(cmd.Data)) > maxFileSize {
		return nil, entity.ErrGalleryFileTooLarge
	}

	// Compute SHA-256 for deduplication.
	sum := sha256.Sum256(cmd.Data)
	digest := hex.EncodeToString(sum[:])

	// Return existing asset if content is identical.
	existing, err := s.repo.FindBySHA256(ctx, cmd.WorkspaceID, digest)
	if err != nil {
		return nil, fmt.Errorf("checking duplicate asset: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	// Build a stable storage key: gallery/{workspaceID}/{sha256[:12]}-{filename}
	safeFilename := sanitizeFilename(cmd.Filename)
	key := fmt.Sprintf("%s/%s/%s-%s", galleryKeyPrefix, cmd.WorkspaceID, digest[:12], safeFilename)

	req := &port.StorageUploadRequest{
		Key:         key,
		Data:        cmd.Data,
		ContentType: cmd.ContentType,
		Environment: entity.EnvironmentProd,
	}
	if err := s.adapter.Upload(ctx, req); err != nil {
		return nil, fmt.Errorf("uploading gallery asset: %w", err)
	}

	asset := &entity.GalleryAsset{
		TenantID:    cmd.TenantID,
		WorkspaceID: cmd.WorkspaceID,
		Key:         key,
		Filename:    cmd.Filename,
		ContentType: cmd.ContentType,
		Size:        int64(len(cmd.Data)),
		SHA256:      digest,
		CreatedBy:   cmd.UserID,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.repo.Save(ctx, asset); err != nil {
		return nil, fmt.Errorf("saving gallery asset metadata: %w", err)
	}

	return asset, nil
}

// DeleteAsset removes an asset from both storage and the gallery registry.
func (s *Service) DeleteAsset(ctx context.Context, cmd galleryuc.DeleteAssetCmd) error {
	req := &port.StorageRequest{
		Key:         cmd.Key,
		Environment: entity.EnvironmentProd,
	}
	if err := s.adapter.Delete(ctx, req); err != nil {
		return fmt.Errorf("deleting gallery asset from storage: %w", err)
	}

	if err := s.repo.Delete(ctx, cmd.WorkspaceID, cmd.Key); err != nil {
		return fmt.Errorf("deleting gallery asset record: %w", err)
	}

	return nil
}

// GetAssetURL returns an HTTP-resolvable URL for the given asset key.
// For local storage (file:// URLs), it returns an API serve URL instead.
func (s *Service) GetAssetURL(ctx context.Context, cmd galleryuc.GetAssetURLCmd) (string, error) {
	req := &port.StorageRequest{
		Key:         cmd.Key,
		Environment: entity.EnvironmentProd,
	}
	rawURL, err := s.adapter.GetURL(ctx, req)
	if err != nil {
		return "", fmt.Errorf("resolving gallery asset URL: %w", err)
	}

	if strings.HasPrefix(rawURL, "file://") {
		// Local storage: serve via API endpoint instead of a filesystem path.
		return fmt.Sprintf("%s/api/v1/workspace/gallery/serve?key=%s",
			s.publicURL, url.QueryEscape(cmd.Key)), nil
	}

	return rawURL, nil
}

func normalizePagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	return page, perPage
}

func sanitizeFilename(name string) string {
	// Replace path separators and null bytes.
	replacer := strings.NewReplacer("/", "_", "\\", "_", "\x00", "")
	return replacer.Replace(name)
}
