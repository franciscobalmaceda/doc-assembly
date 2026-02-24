package port

import (
	"context"

	"github.com/rendis/doc-assembly/core/internal/core/entity"
)

// DocumentFilters contains optional filters for document queries.
type DocumentFilters struct {
	Status                    *entity.DocumentStatus
	SignerProvider            *string
	ClientExternalReferenceID *string
	TemplateVersionID         *string
	Search                    string
	Limit                     int
	Offset                    int
}

// InternalCreateRequest contains the data required for the internal create/replay transaction.
type InternalCreateRequest struct {
	WorkspaceID     string
	DocumentTypeID  string
	ExternalID      string
	TransactionalID string
	ForceCreate     bool
	SupersedeReason *string
	Document        *entity.Document
	Recipients      []*entity.DocumentRecipient
}

// InternalCreateResult represents the outcome of internal create/replay processing.
type InternalCreateResult struct {
	DocumentID                   string
	IdempotentReplay             bool
	SupersededPreviousDocumentID *string
}

// DocumentRepository defines the interface for document data access.
type DocumentRepository interface {
	// Create creates a new document.
	Create(ctx context.Context, document *entity.Document) (string, error)

	// FindByID finds a document by ID.
	FindByID(ctx context.Context, id string) (*entity.Document, error)

	// FindByIDWithRecipients finds a document by ID with all recipients.
	FindByIDWithRecipients(ctx context.Context, id string) (*entity.DocumentWithRecipients, error)

	// FindByWorkspace lists all documents in a workspace with optional filters.
	FindByWorkspace(ctx context.Context, workspaceID string, filters DocumentFilters) ([]*entity.DocumentListItem, error)

	// FindBySignerDocumentID finds a document by the external signing provider's document ID.
	FindBySignerDocumentID(ctx context.Context, signerDocumentID string) (*entity.Document, error)

	// FindByClientExternalRef finds documents by the client's external reference ID.
	FindByClientExternalRef(ctx context.Context, workspaceID, clientExternalRef string) ([]*entity.Document, error)

	// FindByTemplateVersion finds all documents generated from a specific template version.
	FindByTemplateVersion(ctx context.Context, templateVersionID string) ([]*entity.DocumentListItem, error)

	// FindPendingProviderForUpload finds PENDING_PROVIDER documents waiting for provider upload.
	FindPendingProviderForUpload(ctx context.Context, limit int) ([]*entity.Document, error)

	// FindPendingForPolling finds documents that need status polling (PENDING or IN_PROGRESS).
	FindPendingForPolling(ctx context.Context, limit int) ([]*entity.Document, error)

	// FindErrorsForRetry finds ERROR documents eligible for retry (next_retry_at <= now, retry_count < max).
	FindErrorsForRetry(ctx context.Context, maxRetries, limit int) ([]*entity.Document, error)

	// FindExpired finds documents that have passed their expiration time and are still active.
	FindExpired(ctx context.Context, limit int) ([]*entity.Document, error)

	// Update updates a document.
	Update(ctx context.Context, document *entity.Document) error

	// UpdateStatus updates only the status of a document.
	UpdateStatus(ctx context.Context, id string, status entity.DocumentStatus) error

	// ClaimForSigning atomically transitions a document from AWAITING_INPUT to PENDING_PROVIDER.
	// Returns (doc, true, nil) if the claim succeeded; (nil, false, nil) if already claimed.
	ClaimForSigning(ctx context.Context, id string) (*entity.Document, bool, error)

	// Delete deletes a document and all its recipients (cascade).
	Delete(ctx context.Context, id string) error

	// CountByWorkspace returns the total number of documents in a workspace.
	CountByWorkspace(ctx context.Context, workspaceID string) (int, error)

	// CountByStatus returns the count of documents by status in a workspace.
	CountByStatus(ctx context.Context, workspaceID string, status entity.DocumentStatus) (int, error)

	// InternalCreateOrReplay executes transactional create/replay/supersede logic for internal create endpoint.
	InternalCreateOrReplay(ctx context.Context, req *InternalCreateRequest) (*InternalCreateResult, error)
}
