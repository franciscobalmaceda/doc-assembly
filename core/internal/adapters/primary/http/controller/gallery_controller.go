package controller

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rendis/doc-assembly/core/internal/adapters/primary/http/dto"
	"github.com/rendis/doc-assembly/core/internal/adapters/primary/http/middleware"
	"github.com/rendis/doc-assembly/core/internal/core/entity"
	"github.com/rendis/doc-assembly/core/internal/core/port"
	galleryuc "github.com/rendis/doc-assembly/core/internal/core/usecase/gallery"
)

const galleryMaxUploadBytes = 11 * 1024 * 1024 // 11 MB multipart limit (10 MB file + overhead)

var errGalleryMissingKey = errors.New("key query parameter is required")

// GalleryController handles image gallery HTTP requests.
type GalleryController struct {
	galleryUC galleryuc.GalleryUseCase
	adapter   port.StorageAdapter
}

// NewGalleryController creates a new gallery controller.
func NewGalleryController(galleryUC galleryuc.GalleryUseCase, adapter port.StorageAdapter) *GalleryController {
	return &GalleryController{
		galleryUC: galleryUC,
		adapter:   adapter,
	}
}

// RegisterRoutes registers all gallery routes under /workspace/gallery.
func (c *GalleryController) RegisterRoutes(rg *gin.RouterGroup, middlewareProvider *middleware.Provider) {
	gallery := rg.Group("/workspace/gallery")
	gallery.Use(middlewareProvider.WorkspaceContext())
	{
		gallery.GET("", c.ListAssets)                                 // VIEWER+
		gallery.GET("/search", c.SearchAssets)                       // VIEWER+
		gallery.POST("", middleware.RequireEditor(), c.UploadAsset)  // EDITOR+
		gallery.DELETE("", middleware.RequireAdmin(), c.DeleteAsset) // ADMIN+
		gallery.GET("/url", c.GetAssetURL)                           // VIEWER+
		gallery.GET("/serve", c.ServeAsset)                          // VIEWER+ (local storage fallback)
	}
}

// ListAssets returns a paginated list of gallery assets.
func (c *GalleryController) ListAssets(ctx *gin.Context) {
	workspaceID, _ := middleware.GetWorkspaceID(ctx)

	result, err := c.galleryUC.ListAssets(ctx.Request.Context(), galleryuc.ListAssetsCmd{
		WorkspaceID: workspaceID,
		Page:        parseQueryInt(ctx, "page", 1),
		PerPage:     parseQueryInt(ctx, "perPage", 20),
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, galleryPageToResponse(result))
}

// SearchAssets returns gallery assets matching a query string.
func (c *GalleryController) SearchAssets(ctx *gin.Context) {
	workspaceID, _ := middleware.GetWorkspaceID(ctx)

	result, err := c.galleryUC.SearchAssets(ctx.Request.Context(), galleryuc.SearchAssetsCmd{
		WorkspaceID: workspaceID,
		Query:       ctx.Query("q"),
		Page:        parseQueryInt(ctx, "page", 1),
		PerPage:     parseQueryInt(ctx, "perPage", 20),
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, galleryPageToResponse(result))
}

// UploadAsset accepts a multipart file upload and stores it in the gallery.
func (c *GalleryController) UploadAsset(ctx *gin.Context) {
	workspaceID, _ := middleware.GetWorkspaceID(ctx)
	tenantID, _ := middleware.GetTenantIDFromHeader(ctx)
	userID, _ := middleware.GetInternalUserID(ctx)

	if err := ctx.Request.ParseMultipartForm(galleryMaxUploadBytes); err != nil {
		respondError(ctx, http.StatusBadRequest, err)
		return
	}

	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		respondError(ctx, http.StatusBadRequest, err)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		respondError(ctx, http.StatusInternalServerError, err)
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	asset, err := c.galleryUC.UploadAsset(ctx.Request.Context(), galleryuc.UploadAssetCmd{
		TenantID:    tenantID,
		WorkspaceID: workspaceID,
		UserID:      userID,
		Filename:    header.Filename,
		ContentType: contentType,
		Data:        data,
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, galleryAssetToResponse(asset))
}

// DeleteAsset removes an asset from the gallery.
func (c *GalleryController) DeleteAsset(ctx *gin.Context) {
	workspaceID, _ := middleware.GetWorkspaceID(ctx)
	key := ctx.Query("key")
	if key == "" {
		respondError(ctx, http.StatusBadRequest, errGalleryMissingKey)
		return
	}

	if err := c.galleryUC.DeleteAsset(ctx.Request.Context(), galleryuc.DeleteAssetCmd{
		WorkspaceID: workspaceID,
		Key:         key,
	}); err != nil {
		HandleError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GetAssetURL resolves a gallery key to an HTTP URL.
func (c *GalleryController) GetAssetURL(ctx *gin.Context) {
	workspaceID, _ := middleware.GetWorkspaceID(ctx)
	key := ctx.Query("key")
	if key == "" {
		respondError(ctx, http.StatusBadRequest, errGalleryMissingKey)
		return
	}

	resolvedURL, err := c.galleryUC.GetAssetURL(ctx.Request.Context(), galleryuc.GetAssetURLCmd{
		WorkspaceID: workspaceID,
		Key:         key,
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.GalleryURLResponse{URL: resolvedURL})
}

// ServeAsset streams a gallery asset directly from local storage.
// Used as fallback when local storage (file://) is configured instead of S3.
func (c *GalleryController) ServeAsset(ctx *gin.Context) {
	workspaceID, _ := middleware.GetWorkspaceID(ctx)
	key := ctx.Query("key")
	if key == "" {
		respondError(ctx, http.StatusBadRequest, errGalleryMissingKey)
		return
	}

	req := &port.StorageRequest{
		Key:         key,
		Environment: entity.EnvironmentProd,
	}

	data, err := c.adapter.Download(ctx.Request.Context(), req)
	if err != nil {
		slog.WarnContext(ctx.Request.Context(), "gallery serve: asset not found",
			slog.String("workspace_id", workspaceID),
			slog.String("key", key),
		)
		HandleError(ctx, err)
		return
	}

	contentType := http.DetectContentType(data)
	ctx.Data(http.StatusOK, contentType, data)
}

// --- mapping helpers ---

func galleryAssetToResponse(a *entity.GalleryAsset) *dto.GalleryAssetResponse {
	return &dto.GalleryAssetResponse{
		ID:          a.ID,
		Key:         a.Key,
		Filename:    a.Filename,
		ContentType: a.ContentType,
		Size:        a.Size,
		CreatedAt:   a.CreatedAt,
	}
}

func galleryPageToResponse(page *galleryuc.AssetsPage) *dto.GalleryListResponse {
	items := make([]*dto.GalleryAssetResponse, 0, len(page.Assets))
	for _, a := range page.Assets {
		items = append(items, galleryAssetToResponse(a))
	}
	return &dto.GalleryListResponse{
		Items:   items,
		Total:   page.Total,
		Page:    page.Page,
		PerPage: page.PerPage,
	}
}

func parseQueryInt(ctx *gin.Context, key string, defaultVal int) int {
	if s := ctx.Query(key); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			return v
		}
	}
	return defaultVal
}
