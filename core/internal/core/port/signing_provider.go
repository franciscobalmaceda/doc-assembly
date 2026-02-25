package port

import (
	"context"
	"errors"
	"time"

	"github.com/rendis/doc-assembly/core/internal/core/entity"
)

// ErrEmbeddingNotSupported is returned when a provider does not support embedded signing URLs.
var ErrEmbeddingNotSupported = errors.New("provider does not support embedded signing")

// SigningProvider defines the interface for external document signing services.
// Implementations handle the specifics of each provider (Documenso, DocuSign, PandaDoc, etc.)
// while exposing a unified interface to the application.
type SigningProvider interface {
	// UploadDocument uploads a PDF document to the signing provider and creates
	// a signing envelope/request. Returns the provider's document ID and recipient IDs.
	UploadDocument(ctx context.Context, req *UploadDocumentRequest) (*UploadDocumentResult, error)

	// GetSigningURL returns the URL where a specific recipient can sign the document.
	GetSigningURL(ctx context.Context, req *GetSigningURLRequest) (*GetSigningURLResult, error)

	// GetEmbeddedSigningURL returns a URL suitable for embedding in an iframe.
	// The CallbackURL in the request is our signing-callback bridge endpoint.
	// Providers that support redirect (DocuSign, PandaDoc) should incorporate it;
	// providers that don't (Documenso) may ignore it and rely on webhook + polling.
	// Returns ErrEmbeddingNotSupported if the provider cannot provide an embeddable URL.
	GetEmbeddedSigningURL(ctx context.Context, req *GetEmbeddedSigningURLRequest) (*GetEmbeddedSigningURLResult, error)

	// GetDocumentStatus retrieves the current status of a document from the provider.
	GetDocumentStatus(ctx context.Context, req *GetDocumentStatusRequest) (*DocumentStatusResult, error)

	// CancelDocument cancels/voids a document that is pending signatures.
	CancelDocument(ctx context.Context, req *CancelDocumentRequest) error

	// DownloadSignedPDF downloads the completed/signed PDF from the provider.
	DownloadSignedPDF(ctx context.Context, req *DownloadSignedPDFRequest) ([]byte, error)

	// ProviderName returns the name of this signing provider (e.g., "documenso", "docusign").
	ProviderName() string
}

// UploadDocumentRequest contains the data needed to upload a document for signing.
type UploadDocumentRequest struct {
	// PDF is the raw PDF bytes of the document to be signed.
	PDF []byte

	// Title is the display name for the document in the signing provider.
	Title string

	// Recipients is the list of people who need to sign the document.
	Recipients []SigningRecipient

	// ExternalRef is an optional external reference ID (e.g., CRM ID) for tracking.
	ExternalRef string

	// WebhookURL is the URL where the provider should send status updates.
	// Leave empty to use the default configured webhook URL.
	WebhookURL string

	// Metadata contains optional key-value pairs to attach to the document.
	Metadata map[string]string

	// SignatureFields contains position information for signature fields.
	SignatureFields []SignatureFieldPosition

	// Environment indicates dev or prod.
	Environment entity.Environment
}

// SignatureFieldPosition contains the position and size of a signature field.
type SignatureFieldPosition struct {
	// RoleID is the internal role ID that maps this field to a recipient.
	RoleID string

	// Page is the 1-indexed page number where the field should appear.
	Page int

	// PositionX is the X position as a percentage (0-100) from left edge.
	PositionX float64

	// PositionY is the Y position as a percentage (0-100) from top edge.
	PositionY float64

	// Width is the field width as a percentage (0-100) of page width.
	Width float64

	// Height is the field height as a percentage (0-100) of page height.
	Height float64
}

// SigningRecipient represents a person who needs to sign the document.
type SigningRecipient struct {
	// Email is the recipient's email address.
	Email string

	// Name is the recipient's display name.
	Name string

	// RoleID is the internal role ID (template_version_role_id) for this recipient.
	RoleID string

	// SignerOrder determines the signing sequence (1-based). Lower numbers sign first.
	SignerOrder int
}

// UploadDocumentResult contains the result of uploading a document.
type UploadDocumentResult struct {
	// ProviderDocumentID is the unique ID assigned by the signing provider.
	ProviderDocumentID string

	// ProviderName is the name of the signing provider (e.g., "documenso").
	ProviderName string

	// Recipients contains the provider-assigned IDs for each recipient.
	Recipients []RecipientResult

	// Status is the initial status of the document.
	Status entity.DocumentStatus
}

// RecipientResult contains the provider's response for a single recipient.
type RecipientResult struct {
	// RoleID is the internal role ID that was provided in the request.
	RoleID string

	// ProviderRecipientID is the unique ID assigned by the signing provider.
	ProviderRecipientID string

	// SigningURL is the URL where this recipient can sign the document.
	SigningURL string

	// Status is the initial status of this recipient.
	Status entity.RecipientStatus
}

// GetSigningURLRequest contains the data needed to get a signing URL.
type GetSigningURLRequest struct {
	// ProviderDocumentID is the document ID from the signing provider.
	ProviderDocumentID string

	// ProviderRecipientID is the recipient ID from the signing provider.
	ProviderRecipientID string

	// Environment indicates dev or prod.
	Environment entity.Environment
}

// GetSigningURLResult contains the signing URL for a recipient.
type GetSigningURLResult struct {
	// SigningURL is the URL where the recipient can sign.
	SigningURL string

	// ExpiresAt is when the signing URL expires (optional, provider-dependent).
	ExpiresAt *time.Time
}

// GetEmbeddedSigningURLRequest contains the data needed to get an embeddable signing URL.
type GetEmbeddedSigningURLRequest struct {
	// ProviderDocumentID is the document ID from the signing provider.
	ProviderDocumentID string

	// ProviderRecipientID is the recipient ID from the signing provider.
	ProviderRecipientID string

	// CallbackURL is our signing-callback bridge endpoint.
	// Providers that support redirect should append it as returnUrl/redirect.
	// Format: {publicURL}/public/sign/{token}/signing-callback
	CallbackURL string

	// Environment indicates dev or prod.
	Environment entity.Environment
}

// GetEmbeddedSigningURLResult contains the embedded signing URL and CSP info.
type GetEmbeddedSigningURLResult struct {
	// EmbeddedURL is the URL to load in an iframe for embedded signing.
	EmbeddedURL string

	// FrameSrcDomain is the domain to allow in CSP frame-src directive.
	FrameSrcDomain string

	// ExpiresAt is when the embedded URL expires (optional, provider-dependent).
	ExpiresAt *time.Time
}

// DocumentStatusResult contains the current status of a document.
type DocumentStatusResult struct {
	// Status is the overall document status.
	Status entity.DocumentStatus

	// Recipients contains the current status of each recipient.
	Recipients []RecipientStatusResult

	// CompletedPDFURL is the URL to download the signed document (available when completed).
	CompletedPDFURL *string

	// ProviderStatus is the raw status string from the provider (for debugging).
	ProviderStatus string
}

// RecipientStatusResult contains the current status of a recipient.
type RecipientStatusResult struct {
	// ProviderRecipientID is the recipient ID from the signing provider.
	ProviderRecipientID string

	// Status is the current recipient status.
	Status entity.RecipientStatus

	// SignedAt is when the recipient signed (nil if not yet signed).
	SignedAt *time.Time

	// ProviderStatus is the raw status string from the provider (for debugging).
	ProviderStatus string
}

// GetDocumentStatusRequest contains the data needed to get a document status.
type GetDocumentStatusRequest struct {
	ProviderDocumentID string
	Environment        entity.Environment
}

// CancelDocumentRequest contains the data needed to cancel a document.
type CancelDocumentRequest struct {
	ProviderDocumentID string
	Environment        entity.Environment
}

// DownloadSignedPDFRequest contains the data needed to download a signed PDF.
type DownloadSignedPDFRequest struct {
	ProviderDocumentID string
	Environment        entity.Environment
}

// ParseWebhookRequest contains the data needed to parse a webhook event.
type ParseWebhookRequest struct {
	Body        []byte
	Signature   string
	Environment entity.Environment
}

// WebhookEvent represents an incoming webhook event from a signing provider.
type WebhookEvent struct {
	// EventType is the type of event (e.g., "document.signed", "document.completed").
	EventType string

	// ProviderDocumentID is the document ID from the provider.
	ProviderDocumentID string

	// ProviderRecipientID is the recipient ID (if the event is recipient-specific).
	ProviderRecipientID string

	// DocumentStatus is the new document status (if applicable).
	DocumentStatus *entity.DocumentStatus

	// RecipientStatus is the new recipient status (if applicable).
	RecipientStatus *entity.RecipientStatus

	// Timestamp is when the event occurred.
	Timestamp time.Time

	// RawPayload is the original webhook payload for debugging.
	RawPayload []byte
}

// WebhookHandler defines the interface for processing webhook events.
type WebhookHandler interface {
	// ParseWebhook parses and validates an incoming webhook request.
	// Returns the parsed event or an error if the signature is invalid.
	ParseWebhook(ctx context.Context, req *ParseWebhookRequest) (*WebhookEvent, error)
}
