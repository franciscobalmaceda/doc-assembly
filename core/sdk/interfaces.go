package sdk

import (
	"github.com/rendis/doc-assembly/core/internal/core/entity"
	"github.com/rendis/doc-assembly/core/internal/core/port"
)

// --- Core Extensibility Interfaces ---

// Injector resolves injectable values during document rendering.
type Injector = port.Injector

// InitFunc is called once before all injectors on each render request.
type InitFunc = port.InitFunc

// ResolveFunc resolves a single injectable value.
type ResolveFunc = port.ResolveFunc

// RequestMapper transforms incoming render requests before processing.
type RequestMapper = port.RequestMapper

// MapperContext provides request context to the mapper.
type MapperContext = port.MapperContext

// TemplateResolver allows custom internal template version selection.
type TemplateResolver = port.TemplateResolver

// TemplateResolverRequest provides context for custom template resolution.
type TemplateResolverRequest = port.TemplateResolverRequest

// TemplateVersionSearchAdapter is the read-only adapter passed to custom resolvers.
type TemplateVersionSearchAdapter = port.TemplateVersionSearchAdapter

// TemplateVersionSearchParams are filters for searching candidate versions.
type TemplateVersionSearchParams = port.TemplateVersionSearchParams

// TemplateVersionSearchItem is one search result item.
type TemplateVersionSearchItem = port.TemplateVersionSearchItem

// --- Provider Interfaces ---

// WorkspaceInjectableProvider supplies workspace-specific injectable definitions.
type WorkspaceInjectableProvider = port.WorkspaceInjectableProvider

// PublicDocumentAccessAuthenticator provides custom auth for /public/doc/:documentId.
type PublicDocumentAccessAuthenticator = port.PublicDocumentAccessAuthenticator

// SigningProvider handles document signing via an external provider.
type SigningProvider = port.SigningProvider

// WebhookHandler parses incoming webhook events from a signing provider.
type WebhookHandler = port.WebhookHandler

// StorageAdapter handles file storage (local, S3, etc.).
type StorageAdapter = port.StorageAdapter

// NotificationProvider sends notifications (email, etc.).
type NotificationProvider = port.NotificationProvider

// --- Optional Schema Interfaces ---

// TableSchemaProvider can be implemented by Injector to expose table column schema.
type TableSchemaProvider = port.TableSchemaProvider

// ListSchemaProvider can be implemented by Injector to expose list schema.
type ListSchemaProvider = port.ListSchemaProvider

// --- Provider DTOs ---

// WorkspaceInjectableProvider types
type (
	GetInjectablesResult      = port.GetInjectablesResult
	ProviderInjectable        = port.ProviderInjectable
	ProviderFormat            = port.ProviderFormat
	ProviderGroup             = port.ProviderGroup
	ResolveInjectablesRequest = port.ResolveInjectablesRequest
	ResolveInjectablesResult  = port.ResolveInjectablesResult
)

// PublicDocumentAccessAuthenticator types
type PublicDocumentAccessClaims = port.PublicDocumentAccessClaims

// SigningProvider types
type (
	UploadDocumentRequest  = port.UploadDocumentRequest
	UploadDocumentResult   = port.UploadDocumentResult
	SigningRecipient       = port.SigningRecipient
	SignatureFieldPosition = port.SignatureFieldPosition
	RecipientResult        = port.RecipientResult
	GetSigningURLRequest   = port.GetSigningURLRequest
	GetSigningURLResult    = port.GetSigningURLResult
	DocumentStatusResult   = port.DocumentStatusResult
	RecipientStatusResult  = port.RecipientStatusResult
	WebhookEvent           = port.WebhookEvent
)

// NotificationProvider types
type (
	NotificationRequest    = port.NotificationRequest
	NotificationAttachment = port.NotificationAttachment
)

// PDF Renderer types
type (
	SignatureField  = port.SignatureField
	SignerRoleValue = port.SignerRoleValue
)

// Registry types
type GroupConfig = port.GroupConfig

// Environment types
type Environment = entity.Environment

// Environment constants and helpers
var (
	EnvironmentProd        = entity.EnvironmentProd
	EnvironmentDev         = entity.EnvironmentDev
	ParseEnvironment       = entity.ParseEnvironment
	EnvironmentFromSandbox = entity.EnvironmentFromSandbox
)

// SigningProvider request types
type (
	GetDocumentStatusRequest     = port.GetDocumentStatusRequest
	CancelDocumentRequest        = port.CancelDocumentRequest
	DownloadSignedPDFRequest     = port.DownloadSignedPDFRequest
	GetEmbeddedSigningURLRequest = port.GetEmbeddedSigningURLRequest
	ParseWebhookRequest          = port.ParseWebhookRequest
)

// StorageAdapter request types
type (
	StorageUploadRequest = port.StorageUploadRequest
	StorageRequest       = port.StorageRequest
)

// PublicDocumentAccessAuthenticator request type
type AuthenticateRequest = port.AuthenticateRequest
