package entity

import (
	"encoding/json"
	"time"
)

// Document represents a generated document instance from a template version.
// It tracks the document through the signing workflow with an external provider.
type Document struct {
	ID                        string          `json:"id"`
	WorkspaceID               string          `json:"workspaceId"`
	TemplateVersionID         string          `json:"templateVersionId"`
	DocumentTypeID            string          `json:"documentTypeId"`
	Title                     *string         `json:"title,omitempty"`
	ClientExternalReferenceID *string         `json:"clientExternalReferenceId,omitempty"`
	TransactionalID           *string         `json:"transactionalId,omitempty"`
	OperationType             OperationType   `json:"operationType"`
	RelatedDocumentID         *string         `json:"relatedDocumentId,omitempty"`
	SignerDocumentID          *string         `json:"signerDocumentId,omitempty"`
	SignerProvider            *string         `json:"signerProvider,omitempty"`
	Status                    DocumentStatus  `json:"status"`
	InjectedValuesSnapshot    json.RawMessage `json:"injectedValuesSnapshot,omitempty"`
	PDFStoragePath            *string         `json:"pdfStoragePath,omitempty"`
	CompletedPDFURL           *string         `json:"completedPdfUrl,omitempty"`
	IsActive                  bool            `json:"isActive"`
	SupersededAt              *time.Time      `json:"supersededAt,omitempty"`
	SupersededByDocumentID    *string         `json:"supersededByDocumentId,omitempty"`
	SupersedeReason           *string         `json:"supersedeReason,omitempty"`
	ExpiresAt                 *time.Time      `json:"expiresAt,omitempty"`
	Metadata                  json.RawMessage `json:"metadata,omitempty"`
	RetryCount                int             `json:"retryCount"`
	LastRetryAt               *time.Time      `json:"lastRetryAt,omitempty"`
	NextRetryAt               *time.Time      `json:"nextRetryAt,omitempty"`
	CreatedAt                 time.Time       `json:"createdAt"`
	UpdatedAt                 *time.Time      `json:"updatedAt,omitempty"`
}

// NewDocument creates a new document in DRAFT status with CREATE operation type.
func NewDocument(workspaceID, templateVersionID string) *Document {
	return &Document{
		WorkspaceID:       workspaceID,
		TemplateVersionID: templateVersionID,
		OperationType:     OperationCreate,
		Status:            DocumentStatusDraft,
		IsActive:          true,
		CreatedAt:         time.Now().UTC(),
	}
}

// SetTitle sets the document title.
func (d *Document) SetTitle(title string) {
	d.Title = &title
	d.touch()
}

// SetExternalReference sets the external CRM reference ID.
func (d *Document) SetExternalReference(refID string) {
	d.ClientExternalReferenceID = &refID
	d.touch()
}

// SetTransactionalID sets the transactional ID for tracing.
func (d *Document) SetTransactionalID(txnID string) {
	d.TransactionalID = &txnID
	d.touch()
}

// SetOperationType sets the operation type (CREATE, RENEW, AMEND, etc.).
func (d *Document) SetOperationType(opType OperationType) {
	d.OperationType = opType
	d.touch()
}

// SetRelatedDocumentID sets the related document ID for RENEW/AMEND operations.
func (d *Document) SetRelatedDocumentID(docID string) {
	d.RelatedDocumentID = &docID
	d.touch()
}

// SetSignerInfo sets the signing provider information.
func (d *Document) SetSignerInfo(provider, documentID string) {
	d.SignerProvider = &provider
	d.SignerDocumentID = &documentID
	d.touch()
}

// SetSignerDocumentID sets the signer document ID.
func (d *Document) SetSignerDocumentID(docID string) {
	d.SignerDocumentID = &docID
	d.touch()
}

// SetSignerProvider sets the signer provider name.
func (d *Document) SetSignerProvider(provider string) {
	d.SignerProvider = &provider
	d.touch()
}

// SetInjectedValues sets the snapshot of injected values.
func (d *Document) SetInjectedValues(values json.RawMessage) {
	d.InjectedValuesSnapshot = values
	d.touch()
}

// SetInjectedValuesSnapshot marshals and sets the snapshot of injected values from a map.
func (d *Document) SetInjectedValuesSnapshot(values map[string]any) error {
	data, err := json.Marshal(values)
	if err != nil {
		return err
	}
	d.InjectedValuesSnapshot = data
	d.touch()
	return nil
}

// SetPDFPath sets the PDF storage path.
func (d *Document) SetPDFPath(path string) {
	d.PDFStoragePath = &path
	d.touch()
}

// SetCompletedPDFURL sets the URL of the fully signed PDF.
func (d *Document) SetCompletedPDFURL(url string) {
	d.CompletedPDFURL = &url
	d.touch()
}

// SetMetadata marshals and sets the metadata from a map.
func (d *Document) SetMetadata(m map[string]string) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	d.Metadata = data
	d.touch()
	return nil
}

// SetExpiresAt sets the expiration timestamp for the document.
func (d *Document) SetExpiresAt(t time.Time) {
	d.ExpiresAt = &t
	d.touch()
}

// IsExpired returns true if the document has passed its expiration time.
func (d *Document) IsExpired() bool {
	return d.ExpiresAt != nil && time.Now().After(*d.ExpiresAt)
}

// MarkAsAwaitingInput transitions the document to AWAITING_INPUT status.
// This means the document has interactive fields that require recipient input before signing.
func (d *Document) MarkAsAwaitingInput() error {
	if d.Status != DocumentStatusDraft {
		return ErrInvalidDocumentStatusTransition
	}
	d.Status = DocumentStatusAwaitingInput
	d.touch()
	return nil
}

// IsAwaitingInput returns true if the document is waiting for interactive field input.
func (d *Document) IsAwaitingInput() bool {
	return d.Status == DocumentStatusAwaitingInput
}

// MarkAsPendingProvider transitions the document to PENDING_PROVIDER status.
// This means the PDF is saved and waiting for the worker to upload to the signing provider.
func (d *Document) MarkAsPendingProvider() error {
	if d.Status != DocumentStatusDraft && d.Status != DocumentStatusAwaitingInput {
		return ErrInvalidDocumentStatusTransition
	}
	d.Status = DocumentStatusPendingProvider
	d.touch()
	return nil
}

// MarkAsPending transitions the document to PENDING status (sent to provider).
// Allowed from DRAFT, PENDING_PROVIDER, or ERROR (for retry after failed upload).
func (d *Document) MarkAsPending() error {
	if d.Status != DocumentStatusDraft && d.Status != DocumentStatusPendingProvider && d.Status != DocumentStatusError {
		return ErrInvalidDocumentStatusTransition
	}
	d.Status = DocumentStatusPending
	d.touch()
	return nil
}

// MarkAsInProgress transitions the document to IN_PROGRESS status (at least one recipient interacted).
func (d *Document) MarkAsInProgress() error {
	if d.Status != DocumentStatusPending && d.Status != DocumentStatusInProgress {
		return ErrInvalidDocumentStatusTransition
	}
	d.Status = DocumentStatusInProgress
	d.touch()
	return nil
}

// MarkAsCompleted transitions the document to COMPLETED status (all recipients signed).
func (d *Document) MarkAsCompleted() error {
	if d.Status != DocumentStatusPending && d.Status != DocumentStatusInProgress {
		return ErrInvalidDocumentStatusTransition
	}
	d.Status = DocumentStatusCompleted
	d.touch()
	return nil
}

// MarkAsDeclined transitions the document to DECLINED status (a recipient rejected).
func (d *Document) MarkAsDeclined() error {
	if d.Status != DocumentStatusPending && d.Status != DocumentStatusInProgress {
		return ErrInvalidDocumentStatusTransition
	}
	d.Status = DocumentStatusDeclined
	d.touch()
	return nil
}

// MarkAsVoided transitions the document to VOIDED status (cancelled by user).
func (d *Document) MarkAsVoided() error {
	if d.Status == DocumentStatusCompleted || d.Status == DocumentStatusVoided {
		return ErrInvalidDocumentStatusTransition
	}
	d.Status = DocumentStatusVoided
	d.touch()
	return nil
}

// MarkAsExpired transitions the document to EXPIRED status.
func (d *Document) MarkAsExpired() error {
	if d.Status != DocumentStatusPending && d.Status != DocumentStatusInProgress {
		return ErrInvalidDocumentStatusTransition
	}
	d.Status = DocumentStatusExpired
	d.touch()
	return nil
}

// MarkAsError transitions the document to ERROR status (provider error).
func (d *Document) MarkAsError() error {
	d.Status = DocumentStatusError
	d.touch()
	return nil
}

// UpdateStatus updates the document status from provider status.
func (d *Document) UpdateStatus(newStatus DocumentStatus) error {
	if !newStatus.IsValid() {
		return ErrInvalidDocumentStatus
	}
	d.Status = newStatus
	d.touch()
	return nil
}

// IsDraft returns true if the document is in draft status.
func (d *Document) IsDraft() bool {
	return d.Status == DocumentStatusDraft
}

// IsPendingProvider returns true if the document is waiting for worker upload.
func (d *Document) IsPendingProvider() bool {
	return d.Status == DocumentStatusPendingProvider
}

// IsPending returns true if the document is pending signature.
func (d *Document) IsPending() bool {
	return d.Status == DocumentStatusPending
}

// IsInProgress returns true if signing is in progress.
func (d *Document) IsInProgress() bool {
	return d.Status == DocumentStatusInProgress
}

// IsCompleted returns true if all signatures are complete.
func (d *Document) IsCompleted() bool {
	return d.Status == DocumentStatusCompleted
}

// IsDeclined returns true if a recipient declined.
func (d *Document) IsDeclined() bool {
	return d.Status == DocumentStatusDeclined
}

// IsVoided returns true if the document was cancelled.
func (d *Document) IsVoided() bool {
	return d.Status == DocumentStatusVoided
}

// IsTerminal returns true if the document is in a terminal state (no more transitions possible).
func (d *Document) IsTerminal() bool {
	return d.Status == DocumentStatusCompleted ||
		d.Status == DocumentStatusDeclined ||
		d.Status == DocumentStatusVoided ||
		d.Status == DocumentStatusExpired
}

// CanBeSentForSigning returns true if the document can be sent to a signing provider.
func (d *Document) CanBeSentForSigning() bool {
	return d.Status == DocumentStatusDraft
}

// HasSignerInfo returns true if the document has been registered with a signing provider.
func (d *Document) HasSignerInfo() bool {
	return d.SignerDocumentID != nil && d.SignerProvider != nil
}

// Validate checks if the document data is valid.
func (d *Document) Validate() error {
	if d.WorkspaceID == "" {
		return ErrRequiredField
	}
	if d.TemplateVersionID == "" {
		return ErrRequiredField
	}
	if d.DocumentTypeID == "" {
		return ErrRequiredField
	}
	if !d.Status.IsValid() {
		return ErrInvalidDocumentStatus
	}
	if d.Title != nil && len(*d.Title) > 255 {
		return ErrFieldTooLong
	}
	return nil
}

// ScheduleRetry increments the retry counter and calculates the next retry time
// using exponential backoff: 60s * 2^retryCount, capped at 1 hour.
// Returns false if maxRetries has been reached (no retry scheduled).
func (d *Document) ScheduleRetry(maxRetries int) bool {
	if d.RetryCount >= maxRetries {
		return false
	}

	now := time.Now().UTC()
	d.RetryCount++
	d.LastRetryAt = &now

	backoff := 60 * time.Second * (1 << d.RetryCount)
	const maxBackoff = time.Hour
	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	nextRetry := now.Add(backoff)
	d.NextRetryAt = &nextRetry
	d.touch()
	return true
}

// ResetRetry clears all retry tracking fields after a successful recovery.
func (d *Document) ResetRetry() {
	d.RetryCount = 0
	d.LastRetryAt = nil
	d.NextRetryAt = nil
	d.touch()
}

// touch updates the UpdatedAt timestamp.
func (d *Document) touch() {
	now := time.Now().UTC()
	d.UpdatedAt = &now
}

// DocumentWithRecipients represents a document with its recipients.
type DocumentWithRecipients struct {
	Document
	Recipients []*DocumentRecipient `json:"recipients,omitempty"`
}

// DocumentListItem represents a document in list views (without full details).
type DocumentListItem struct {
	ID                        string         `json:"id"`
	WorkspaceID               string         `json:"workspaceId"`
	TemplateVersionID         string         `json:"templateVersionId"`
	Title                     *string        `json:"title,omitempty"`
	ClientExternalReferenceID *string        `json:"clientExternalReferenceId,omitempty"`
	SignerProvider            *string        `json:"signerProvider,omitempty"`
	Status                    DocumentStatus `json:"status"`
	CreatedAt                 time.Time      `json:"createdAt"`
	UpdatedAt                 *time.Time     `json:"updatedAt,omitempty"`
}
