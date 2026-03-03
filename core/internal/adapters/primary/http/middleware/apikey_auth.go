package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rendis/doc-assembly/core/internal/core/entity"
	"github.com/rendis/doc-assembly/core/internal/core/port"
)

// APIKeyHeader is the HTTP header name for API key authentication.
const APIKeyHeader = "X-API-Key" //nolint:gosec // This is a header name, not a credential

// InternalKeyAuth creates a middleware that validates an internal API key
// against the database. Uses SHA-256 hashing for key lookup.
func InternalKeyAuth(keyRepo port.AutomationAPIKeyRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawKey := c.GetHeader(APIKeyHeader)
		if rawKey == "" {
			abortWithError(c, http.StatusUnauthorized, entity.ErrMissingAPIKey)
			return
		}

		// Hash the raw key with SHA-256
		sum := sha256.Sum256([]byte(rawKey))
		keyHash := hex.EncodeToString(sum[:])

		// Look up via repository
		key, err := keyRepo.FindByHash(c.Request.Context(), keyHash)
		if err != nil || key == nil {
			abortWithError(c, http.StatusUnauthorized, entity.ErrInvalidAPIKey)
			return
		}

		// Verify the key is active, not revoked, and is an internal key
		if !key.IsActive || key.IsRevoked() || key.KeyType != entity.KeyTypeInternal {
			abortWithError(c, http.StatusUnauthorized, entity.ErrInvalidAPIKey)
			return
		}

		// Fire and forget: update last used timestamp (use background context
		// because the request context may be cancelled before the goroutine runs).
		go func() {
			_ = keyRepo.TouchLastUsed(context.Background(), key.ID)
		}()

		c.Next()
	}
}
