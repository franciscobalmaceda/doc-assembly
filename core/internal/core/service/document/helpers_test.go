package document

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rendis/doc-assembly/core/internal/core/entity"
	"github.com/rendis/doc-assembly/core/internal/core/port"
)

func TestResolvePreProviderRecoveryTarget(t *testing.T) {
	t.Run("storage enabled and object exists goes to pending provider", func(t *testing.T) {
		path := "documents/ws/doc/pre-signed.pdf"
		doc := &entity.Document{PDFStoragePath: &path}
		adapter := &testStorageAdapter{exists: true}

		status, clearPath := resolvePreProviderRecoveryTarget(context.Background(), adapter, true, doc)

		assert.Equal(t, entity.DocumentStatusPendingProvider, status)
		assert.False(t, clearPath)
		assert.Equal(t, path, adapter.lastExistsKey)
	})

	t.Run("storage enabled without path goes to awaiting input", func(t *testing.T) {
		doc := &entity.Document{}

		status, clearPath := resolvePreProviderRecoveryTarget(context.Background(), &testStorageAdapter{}, true, doc)

		assert.Equal(t, entity.DocumentStatusAwaitingInput, status)
		assert.False(t, clearPath)
	})

	t.Run("storage disabled always goes to awaiting input", func(t *testing.T) {
		path := "documents/ws/doc/pre-signed.pdf"
		doc := &entity.Document{PDFStoragePath: &path}

		status, clearPath := resolvePreProviderRecoveryTarget(context.Background(), &testStorageAdapter{exists: true}, false, doc)

		assert.Equal(t, entity.DocumentStatusAwaitingInput, status)
		assert.False(t, clearPath)
	})

	t.Run("missing storage adapter clears stale path and goes to awaiting input", func(t *testing.T) {
		path := "documents/ws/doc/pre-signed.pdf"
		doc := &entity.Document{PDFStoragePath: &path}

		status, clearPath := resolvePreProviderRecoveryTarget(context.Background(), nil, true, doc)

		assert.Equal(t, entity.DocumentStatusAwaitingInput, status)
		assert.True(t, clearPath)
	})

	t.Run("not found object clears stale path and goes to awaiting input", func(t *testing.T) {
		path := "documents/ws/doc/pre-signed.pdf"
		doc := &entity.Document{PDFStoragePath: &path}
		adapter := &testStorageAdapter{exists: false}

		status, clearPath := resolvePreProviderRecoveryTarget(context.Background(), adapter, true, doc)

		assert.Equal(t, entity.DocumentStatusAwaitingInput, status)
		assert.True(t, clearPath)
	})

	t.Run("storage exists error clears stale path and goes to awaiting input", func(t *testing.T) {
		path := "documents/ws/doc/pre-signed.pdf"
		doc := &entity.Document{PDFStoragePath: &path}
		adapter := &testStorageAdapter{existsErr: errors.New("storage unavailable")}

		status, clearPath := resolvePreProviderRecoveryTarget(context.Background(), adapter, true, doc)

		assert.Equal(t, entity.DocumentStatusAwaitingInput, status)
		assert.True(t, clearPath)
	})
}

func TestSignedDocumentFilename(t *testing.T) {
	t.Run("uses title when available", func(t *testing.T) {
		title := "Enrollment Contract"
		doc := &entity.Document{ID: "doc-1", Title: &title}
		assert.Equal(t, "Enrollment Contract-signed.pdf", signedDocumentFilename(doc))
	})

	t.Run("falls back to document id", func(t *testing.T) {
		doc := &entity.Document{ID: "doc-2"}
		assert.Equal(t, "document-doc-2-signed.pdf", signedDocumentFilename(doc))
	})
}

type testStorageAdapter struct {
	exists        bool
	existsErr     error
	lastExistsKey string
}

func (a *testStorageAdapter) Upload(_ context.Context, _ *port.StorageUploadRequest) error {
	return nil
}

func (a *testStorageAdapter) Download(_ context.Context, _ *port.StorageRequest) ([]byte, error) {
	return nil, nil
}

func (a *testStorageAdapter) GetURL(_ context.Context, _ *port.StorageRequest) (string, error) {
	return "", nil
}

func (a *testStorageAdapter) Delete(_ context.Context, _ *port.StorageRequest) error {
	return nil
}

func (a *testStorageAdapter) Exists(_ context.Context, req *port.StorageRequest) (bool, error) {
	if req != nil {
		a.lastExistsKey = req.Key
	}
	return a.exists, a.existsErr
}
