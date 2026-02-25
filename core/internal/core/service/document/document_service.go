package document

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/rendis/doc-assembly/core/internal/core/entity"
	"github.com/rendis/doc-assembly/core/internal/core/entity/portabledoc"
	"github.com/rendis/doc-assembly/core/internal/core/port"
	documentuc "github.com/rendis/doc-assembly/core/internal/core/usecase/document"
)

// NewDocumentService creates a new document service.
func NewDocumentService(
	documentRepo port.DocumentRepository,
	recipientRepo port.DocumentRecipientRepository,
	templateRepo port.TemplateRepository,
	versionRepo port.TemplateVersionRepository,
	signerRoleRepo port.TemplateVersionSignerRoleRepository,
	pdfRenderer port.PDFRenderer,
	signingProvider port.SigningProvider,
	storageAdapter port.StorageAdapter,
	eventEmitter *EventEmitter,
	notificationSvc *NotificationService,
	expirationDays int,
	accessTokenRepo port.DocumentAccessTokenRepository,
	fieldResponseRepo port.DocumentFieldResponseRepository,
) *DocumentService {
	return &DocumentService{
		documentRepo:      documentRepo,
		recipientRepo:     recipientRepo,
		templateRepo:      templateRepo,
		versionRepo:       versionRepo,
		signerRoleRepo:    signerRoleRepo,
		pdfRenderer:       pdfRenderer,
		signingProvider:   signingProvider,
		storageAdapter:    storageAdapter,
		eventEmitter:      eventEmitter,
		notificationSvc:   notificationSvc,
		expirationDays:    expirationDays,
		accessTokenRepo:   accessTokenRepo,
		fieldResponseRepo: fieldResponseRepo,
	}
}

// DocumentService implements document business logic.
type DocumentService struct {
	documentRepo      port.DocumentRepository
	recipientRepo     port.DocumentRecipientRepository
	templateRepo      port.TemplateRepository
	versionRepo       port.TemplateVersionRepository
	signerRoleRepo    port.TemplateVersionSignerRoleRepository
	pdfRenderer       port.PDFRenderer
	signingProvider   port.SigningProvider
	storageAdapter    port.StorageAdapter
	eventEmitter      *EventEmitter
	notificationSvc   *NotificationService
	expirationDays    int
	accessTokenRepo   port.DocumentAccessTokenRepository
	fieldResponseRepo port.DocumentFieldResponseRepository

	completionNotifier port.DocumentCompletionNotifier
}

// SetCompletionNotifier sets the notifier used for transactional document
// completion processing. When set, completed documents are persisted and
// enqueued atomically.
func (s *DocumentService) SetCompletionNotifier(n port.DocumentCompletionNotifier) {
	s.completionNotifier = n
}

// persistDocUpdate persists a document update. For completed documents with
// a configured notifier, it uses PersistAndNotify for transactional atomicity.
// Otherwise it falls back to the standard repository update.
func (s *DocumentService) persistDocUpdate(ctx context.Context, doc *entity.Document) error {
	if doc.IsCompleted() && s.completionNotifier != nil {
		return s.completionNotifier.PersistAndNotify(ctx, doc)
	}
	return s.documentRepo.Update(ctx, doc)
}

// CreateAndSendDocument creates a document, generates the PDF, and sends it for signing.
// If the template has interactive fields and exactly one unsigned signer, the document
// is placed in AWAITING_INPUT status with an access token instead of rendering/uploading.
func (s *DocumentService) CreateAndSendDocument(ctx context.Context, cmd documentuc.CreateDocumentCommand) (*entity.DocumentWithRecipients, error) {
	if err := s.validateOperationType(ctx, cmd); err != nil {
		return nil, err
	}

	version, _, err := s.validateTemplateAndRoles(ctx, cmd)
	if err != nil {
		return nil, err
	}

	template, err := s.templateRepo.FindByID(ctx, version.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("finding template from version: %w", err)
	}
	if template.DocumentTypeID == nil || *template.DocumentTypeID == "" {
		return nil, entity.ErrDocumentTypeNotFound
	}

	document, err := s.createDocument(ctx, cmd, *template.DocumentTypeID)
	if err != nil {
		return nil, err
	}

	recipients, err := s.createRecipients(ctx, document.ID, cmd.Recipients)
	if err != nil {
		return nil, err
	}

	// All documents enter AWAITING_INPUT. Tokens are generated on-demand
	// via the email-verification flow (DocumentAccessService).
	if err := s.transitionToAwaitingInput(ctx, document); err != nil {
		return nil, err
	}

	s.notificationSvc.NotifyDocumentCreated(ctx, document.ID)

	return &entity.DocumentWithRecipients{
		Document:   *document,
		Recipients: recipients,
	}, nil
}

// validateTemplateAndRoles validates the template version and returns the role map.
func (s *DocumentService) validateTemplateAndRoles(ctx context.Context, cmd documentuc.CreateDocumentCommand) (*entity.TemplateVersion, map[string]*entity.TemplateVersionSignerRole, error) {
	version, err := s.versionRepo.FindByID(ctx, cmd.TemplateVersionID)
	if err != nil {
		return nil, nil, fmt.Errorf("finding template version: %w", err)
	}

	if !version.IsPublished() {
		return nil, nil, fmt.Errorf("template version is not published")
	}

	signerRoles, err := s.signerRoleRepo.FindByVersionID(ctx, cmd.TemplateVersionID)
	if err != nil {
		return nil, nil, fmt.Errorf("finding signer roles: %w", err)
	}

	if len(cmd.Recipients) != len(signerRoles) {
		return nil, nil, fmt.Errorf("recipient count (%d) does not match signer role count (%d)", len(cmd.Recipients), len(signerRoles))
	}

	roleMap := make(map[string]*entity.TemplateVersionSignerRole)
	for _, role := range signerRoles {
		roleMap[role.ID] = role
	}

	for _, r := range cmd.Recipients {
		if _, ok := roleMap[r.RoleID]; !ok {
			return nil, nil, fmt.Errorf("invalid role ID: %s", r.RoleID)
		}
	}

	return version, roleMap, nil
}

// validateOperationType validates the operation type and related document for RENEW/AMEND.
func (s *DocumentService) validateOperationType(ctx context.Context, cmd documentuc.CreateDocumentCommand) error {
	opType := cmd.OperationType
	if opType == "" {
		opType = entity.OperationCreate
	}

	if !opType.IsValid() {
		return entity.ErrInvalidOperationType
	}

	if opType == entity.OperationCreate {
		return nil
	}

	if opType != entity.OperationRenew && opType != entity.OperationAmend {
		return nil
	}

	if cmd.RelatedDocumentID == nil {
		return entity.ErrRelatedDocumentRequired
	}

	relatedDoc, err := s.documentRepo.FindByID(ctx, *cmd.RelatedDocumentID)
	if err != nil {
		return fmt.Errorf("finding related document: %w", err)
	}

	if relatedDoc.WorkspaceID != cmd.WorkspaceID {
		return entity.ErrRelatedDocumentSameWorkspace
	}

	switch opType {
	case entity.OperationRenew:
		if !relatedDoc.IsCompleted() {
			return entity.ErrDocumentNotCompleted
		}
	case entity.OperationAmend:
		if !relatedDoc.IsTerminal() {
			return entity.ErrDocumentNotTerminal
		}
	}

	return nil
}

// createDocument creates and persists the document entity.
func (s *DocumentService) createDocument(ctx context.Context, cmd documentuc.CreateDocumentCommand, documentTypeID string) (*entity.Document, error) {
	document := entity.NewDocument(cmd.WorkspaceID, cmd.TemplateVersionID)
	document.DocumentTypeID = documentTypeID
	document.ID = uuid.NewString()
	document.SetTitle(cmd.Title)

	if cmd.OperationType != "" && cmd.OperationType != entity.OperationCreate {
		document.SetOperationType(cmd.OperationType)
	}

	if cmd.RelatedDocumentID != nil {
		document.SetRelatedDocumentID(*cmd.RelatedDocumentID)
	}

	if cmd.ClientExternalReferenceID != nil {
		document.SetExternalReference(*cmd.ClientExternalReferenceID)
	}
	// Ensure transactional_id is always present for traceability and DB constraints.
	document.SetTransactionalID(uuid.NewString())

	if cmd.InjectedValues != nil {
		valuesJSON, err := json.Marshal(cmd.InjectedValues)
		if err != nil {
			return nil, fmt.Errorf("marshaling injected values: %w", err)
		}
		document.SetInjectedValues(valuesJSON)
	}

	if err := document.Validate(); err != nil {
		return nil, fmt.Errorf("validating document: %w", err)
	}

	docID, err := s.documentRepo.Create(ctx, document)
	if err != nil {
		return nil, fmt.Errorf("creating document: %w", err)
	}
	document.ID = docID

	return document, nil
}

// createRecipients creates and persists the document recipients.
func (s *DocumentService) createRecipients(ctx context.Context, docID string, cmdRecipients []documentuc.DocumentRecipientCommand) ([]*entity.DocumentRecipient, error) {
	recipients := make([]*entity.DocumentRecipient, len(cmdRecipients))

	for i, r := range cmdRecipients {
		recipient := entity.NewDocumentRecipient(docID, r.RoleID, r.Name, r.Email)
		recipient.ID = uuid.NewString()

		if err := recipient.Validate(); err != nil {
			return nil, fmt.Errorf("validating recipient: %w", err)
		}

		recipients[i] = recipient
	}

	if err := s.recipientRepo.CreateBatch(ctx, recipients); err != nil {
		return nil, fmt.Errorf("creating recipients: %w", err)
	}

	return recipients, nil
}

// signatureLineBottomRatio is the fraction of the signature box height placed above the anchor line.
// The box is positioned so the signature line (anchor point) ends up near the bottom ~30% of the box.
const signatureLineBottomRatio = 0.7

// convertFieldToProviderPosition converts raw PDF coordinates to provider-specific
// percentage positions (0-100, Y from top). Falls back to default percentages when
// raw extraction data is absent.
func convertFieldToProviderPosition(f port.SignatureField) (posX, posY float64) {
	posX, posY = f.PositionX, f.PositionY // defaults (percentages)

	if f.PDFPageW > 0 && f.PDFPageH > 0 {
		// Anchor text is centered in its grid cell, so its center X ≈ signature line center.
		// Center the box horizontally on the anchor's center position.
		anchorCenterPct := ((f.PDFPointX + f.PDFAnchorW/2) / f.PDFPageW) * 100
		posX = anchorCenterPct - f.Width/2

		// Flip Y: PDF bottom→top to provider top→bottom.
		posY = 100 - ((f.PDFPointY / f.PDFPageH) * 100)
		// Offset Y upward so signature line ends up at bottom of box.
		posY -= f.Height * signatureLineBottomRatio
	}

	// Clamp to valid range.
	posX = max(0, min(posX, 100-f.Width))
	posY = max(0, min(posY, 100-f.Height))
	return posX, posY
}

// GetDocument retrieves a document by ID.
func (s *DocumentService) GetDocument(ctx context.Context, id string) (*entity.Document, error) {
	return s.documentRepo.FindByID(ctx, id)
}

// GetDocumentWithRecipients retrieves a document with all its recipients.
func (s *DocumentService) GetDocumentWithRecipients(ctx context.Context, id string) (*entity.DocumentWithRecipients, error) {
	return s.documentRepo.FindByIDWithRecipients(ctx, id)
}

// ListDocuments lists documents in a workspace with optional filters.
func (s *DocumentService) ListDocuments(ctx context.Context, workspaceID string, filters port.DocumentFilters) ([]*entity.DocumentListItem, error) {
	return s.documentRepo.FindByWorkspace(ctx, workspaceID, filters)
}

// GetSigningURL retrieves the signing URL for a specific recipient.
func (s *DocumentService) GetSigningURL(ctx context.Context, documentID, recipientID string) (string, error) {
	doc, err := s.documentRepo.FindByID(ctx, documentID)
	if err != nil {
		return "", fmt.Errorf("finding document: %w", err)
	}

	if !doc.HasSignerInfo() {
		return "", fmt.Errorf("document has not been sent to signing provider")
	}

	recipient, err := s.recipientRepo.FindByID(ctx, recipientID)
	if err != nil {
		return "", fmt.Errorf("finding recipient: %w", err)
	}

	if recipient.DocumentID != documentID {
		return "", fmt.Errorf("recipient does not belong to this document")
	}

	if !recipient.HasSignerInfo() {
		return "", fmt.Errorf("recipient has not been registered with signing provider")
	}

	result, err := s.signingProvider.GetSigningURL(ctx, &port.GetSigningURLRequest{
		ProviderDocumentID:  *doc.SignerDocumentID,
		ProviderRecipientID: *recipient.SignerRecipientID,
	})
	if err != nil {
		return "", fmt.Errorf("getting signing URL: %w", err)
	}

	return result.SigningURL, nil
}

// RefreshDocumentStatus polls the signing provider for the latest status.
func (s *DocumentService) RefreshDocumentStatus(ctx context.Context, documentID string) (*entity.DocumentWithRecipients, error) {
	doc, err := s.documentRepo.FindByID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("finding document: %w", err)
	}

	if !doc.HasSignerInfo() {
		return nil, fmt.Errorf("document has not been sent to signing provider")
	}

	if doc.IsTerminal() {
		return s.documentRepo.FindByIDWithRecipients(ctx, documentID)
	}

	oldStatus := string(doc.Status)

	statusResult, err := s.signingProvider.GetDocumentStatus(ctx, *doc.SignerDocumentID)
	if err != nil {
		return nil, fmt.Errorf("getting document status: %w", err)
	}

	if err := s.updateDocumentFromStatus(ctx, doc, statusResult); err != nil {
		return nil, err
	}

	if err := s.updateRecipientsFromStatus(ctx, documentID, statusResult.Recipients); err != nil {
		return nil, err
	}

	s.eventEmitter.EmitDocumentEvent(ctx, documentID, entity.EventStatusRefreshed, entity.ActorSystem, "", oldStatus, string(doc.Status), nil)

	return s.documentRepo.FindByIDWithRecipients(ctx, documentID)
}

// updateDocumentFromStatus updates the document with status from the signing provider.
func (s *DocumentService) updateDocumentFromStatus(ctx context.Context, doc *entity.Document, statusResult *port.DocumentStatusResult) error {
	if err := doc.UpdateStatus(statusResult.Status); err != nil {
		slog.WarnContext(ctx, "failed to update document status", slog.String("error", err.Error()))
	}

	if statusResult.CompletedPDFURL != nil {
		doc.SetCompletedPDFURL(*statusResult.CompletedPDFURL)
	}

	if err := s.persistDocUpdate(ctx, doc); err != nil {
		return fmt.Errorf("updating document: %w", err)
	}

	if doc.IsCompleted() {
		s.downloadAndStorePDF(ctx, doc)
	}

	return nil
}

// updateRecipientsFromStatus updates recipients with their statuses from the signing provider.
func (s *DocumentService) updateRecipientsFromStatus(ctx context.Context, documentID string, recipientStatuses []port.RecipientStatusResult) error {
	recipients, err := s.recipientRepo.FindByDocumentID(ctx, documentID)
	if err != nil {
		return fmt.Errorf("finding recipients: %w", err)
	}

	for _, recipientStatus := range recipientStatuses {
		s.updateSingleRecipientFromStatus(ctx, recipients, recipientStatus)
	}

	return nil
}

// updateSingleRecipientFromStatus updates a single recipient's status.
func (s *DocumentService) updateSingleRecipientFromStatus(ctx context.Context, recipients []*entity.DocumentRecipient, status port.RecipientStatusResult) {
	for _, recipient := range recipients {
		if recipient.SignerRecipientID == nil || *recipient.SignerRecipientID != status.ProviderRecipientID {
			continue
		}

		if err := recipient.UpdateStatus(status.Status); err != nil {
			slog.WarnContext(ctx, "failed to update recipient status", slog.String("error", err.Error()))
		}
		if status.SignedAt != nil {
			recipient.SignedAt = status.SignedAt
		}
		if err := s.recipientRepo.Update(ctx, recipient); err != nil {
			slog.WarnContext(ctx, "failed to update recipient", slog.String("error", err.Error()))
		}
		break
	}
}

// CancelDocument cancels/voids a document that is pending signatures.
func (s *DocumentService) CancelDocument(ctx context.Context, documentID string) error {
	doc, err := s.documentRepo.FindByID(ctx, documentID)
	if err != nil {
		return fmt.Errorf("finding document: %w", err)
	}

	oldStatus := string(doc.Status)

	if doc.IsTerminal() {
		return fmt.Errorf("cannot cancel document in terminal state: %s", doc.Status)
	}

	if doc.HasSignerInfo() {
		if err := s.signingProvider.CancelDocument(ctx, *doc.SignerDocumentID); err != nil {
			return fmt.Errorf("canceling document with provider: %w", err)
		}
	}

	if err := doc.MarkAsVoided(); err != nil {
		return fmt.Errorf("marking document as voided: %w", err)
	}

	if err := s.documentRepo.Update(ctx, doc); err != nil {
		return fmt.Errorf("updating document: %w", err)
	}

	s.eventEmitter.EmitDocumentEvent(ctx, documentID, entity.EventDocumentCancelled, entity.ActorUser, "", oldStatus, string(entity.DocumentStatusVoided), nil)

	slog.InfoContext(ctx, "document cancelled", slog.String("document_id", documentID))

	return nil
}

// HandleWebhookEvent processes an incoming webhook event from the signing provider.
func (s *DocumentService) HandleWebhookEvent(ctx context.Context, event *port.WebhookEvent) error {
	doc, err := s.documentRepo.FindBySignerDocumentID(ctx, event.ProviderDocumentID)
	if err != nil {
		// Fallback: if externalId was set to our document UUID, try direct lookup
		doc, err = s.documentRepo.FindByID(ctx, event.ProviderDocumentID)
		if err != nil {
			return fmt.Errorf("finding document by provider ID: %w", err)
		}
	}

	slog.InfoContext(ctx, "processing webhook event",
		slog.String("event_type", event.EventType),
		slog.String("document_id", doc.ID),
		slog.String("provider_document_id", event.ProviderDocumentID),
	)

	s.eventEmitter.EmitDocumentEvent(ctx, doc.ID, entity.EventWebhookReceived, entity.ActorWebhook, "", "", "", json.RawMessage(event.RawPayload))

	oldStatus := string(doc.Status)

	if err := s.processDocumentStatusFromWebhook(ctx, doc, event); err != nil {
		return err
	}

	if event.DocumentStatus != nil && string(*event.DocumentStatus) != oldStatus {
		s.eventEmitter.EmitDocumentEvent(ctx, doc.ID, eventTypeFromDocumentStatus(*event.DocumentStatus), entity.ActorWebhook, "", oldStatus, string(*event.DocumentStatus), nil)
	}

	if event.ProviderRecipientID != "" && event.RecipientStatus != nil {
		s.processRecipientStatusFromWebhook(ctx, doc, event)
	}

	return nil
}

// processDocumentStatusFromWebhook updates document status from webhook event.
func (s *DocumentService) processDocumentStatusFromWebhook(ctx context.Context, doc *entity.Document, event *port.WebhookEvent) error {
	if event.DocumentStatus != nil {
		if err := doc.UpdateStatus(*event.DocumentStatus); err != nil {
			slog.WarnContext(ctx, "failed to update document status from webhook",
				slog.String("error", err.Error()),
				slog.String("current_status", doc.Status.String()),
				slog.String("new_status", event.DocumentStatus.String()),
			)
		}
	}

	if err := s.persistDocUpdate(ctx, doc); err != nil {
		return fmt.Errorf("updating document: %w", err)
	}

	if doc.IsCompleted() {
		s.downloadAndStorePDF(ctx, doc)
	}

	return nil
}

// processRecipientStatusFromWebhook processes recipient-specific webhook events.
func (s *DocumentService) processRecipientStatusFromWebhook(ctx context.Context, doc *entity.Document, event *port.WebhookEvent) {
	recipient, err := s.recipientRepo.FindBySignerRecipientID(ctx, event.ProviderRecipientID)
	if err != nil {
		slog.WarnContext(ctx, "recipient not found for webhook event",
			slog.String("provider_recipient_id", event.ProviderRecipientID),
		)
		return
	}

	if err := recipient.UpdateStatus(*event.RecipientStatus); err != nil {
		slog.WarnContext(ctx, "failed to update recipient status from webhook", slog.String("error", err.Error()))
	}
	if err := s.recipientRepo.Update(ctx, recipient); err != nil {
		slog.WarnContext(ctx, "failed to update recipient", slog.String("error", err.Error()))
	}

	s.updateDocumentStatusFromRecipient(ctx, doc, *event.RecipientStatus)
}

// updateDocumentStatusFromRecipient updates document status based on recipient status changes.
func (s *DocumentService) updateDocumentStatusFromRecipient(ctx context.Context, doc *entity.Document, recipientStatus entity.RecipientStatus) {
	switch recipientStatus {
	case entity.RecipientStatusSigned:
		allSigned, err := s.recipientRepo.AllSigned(ctx, doc.ID)
		if err != nil {
			slog.WarnContext(ctx, "failed to check all-signed status",
				slog.String("document_id", doc.ID),
				slog.String("error", err.Error()),
			)
			return
		}
		if !allSigned {
			return
		}
		if err := doc.MarkAsCompleted(); err != nil {
			slog.WarnContext(ctx, "failed to mark document as completed",
				slog.String("document_id", doc.ID),
				slog.String("error", err.Error()),
			)
			return
		}
		if err := s.persistDocUpdate(ctx, doc); err != nil {
			slog.WarnContext(ctx, "failed to persist completed document",
				slog.String("document_id", doc.ID),
				slog.String("error", err.Error()),
			)
			return
		}
		s.downloadAndStorePDF(ctx, doc)

	case entity.RecipientStatusDeclined:
		if err := doc.MarkAsDeclined(); err != nil {
			slog.WarnContext(ctx, "failed to mark document as declined",
				slog.String("document_id", doc.ID),
				slog.String("error", err.Error()),
			)
			return
		}
		if err := s.documentRepo.Update(ctx, doc); err != nil {
			slog.WarnContext(ctx, "failed to persist declined document",
				slog.String("document_id", doc.ID),
				slog.String("error", err.Error()),
			)
		}
	}
}

// ProcessPendingDocuments polls the signing provider for documents that need status updates.
func (s *DocumentService) ProcessPendingDocuments(ctx context.Context, limit int) error {
	docs, err := s.documentRepo.FindPendingForPolling(ctx, limit)
	if err != nil {
		return fmt.Errorf("finding pending documents: %w", err)
	}

	for _, doc := range docs {
		if _, err := s.RefreshDocumentStatus(ctx, doc.ID); err != nil {
			slog.WarnContext(ctx, "failed to refresh document status",
				slog.String("document_id", doc.ID),
				slog.String("error", err.Error()),
			)
		}
	}

	return nil
}

// ProcessPendingProviderDocuments uploads PENDING_PROVIDER documents to the signing provider.
func (s *DocumentService) ProcessPendingProviderDocuments(ctx context.Context, limit int) error {
	docs, err := s.documentRepo.FindPendingProviderForUpload(ctx, limit)
	if err != nil {
		return fmt.Errorf("finding PENDING_PROVIDER documents: %w", err)
	}

	for _, doc := range docs {
		if err := s.uploadPendingProviderDocument(ctx, doc); err != nil {
			slog.WarnContext(ctx, "failed to upload PENDING_PROVIDER document",
				slog.String("document_id", doc.ID),
				slog.String("error", err.Error()),
			)
		}
	}

	return nil
}

// uploadPendingProviderDocument uploads a single PENDING_PROVIDER document to the signing provider.
func (s *DocumentService) uploadPendingProviderDocument(ctx context.Context, doc *entity.Document) error {
	if doc.PDFStoragePath == nil || *doc.PDFStoragePath == "" {
		s.markDocError(ctx, doc)
		return fmt.Errorf("document %s has no PDF storage path", doc.ID)
	}

	pdfData, err := s.storageAdapter.Download(ctx, *doc.PDFStoragePath)
	if err != nil {
		s.markDocError(ctx, doc)
		return fmt.Errorf("downloading PDF for document %s: %w", doc.ID, err)
	}

	recipients, err := s.recipientRepo.FindByDocumentID(ctx, doc.ID)
	if err != nil {
		return fmt.Errorf("loading recipients for document %s: %w", doc.ID, err)
	}

	title := documentTitle(doc)

	result, err := s.signingProvider.UploadDocument(ctx, &port.UploadDocumentRequest{
		PDF:        pdfData,
		Title:      title,
		Recipients: buildSigningRecipients(recipients),
	})
	if err != nil {
		s.markDocError(ctx, doc)
		return fmt.Errorf("uploading document %s to signing provider: %w", doc.ID, err)
	}

	doc.SetSignerInfo(result.ProviderName, result.ProviderDocumentID)
	if markErr := doc.MarkAsPending(); markErr != nil {
		return fmt.Errorf("marking document %s as pending: %w", doc.ID, markErr)
	}
	if updateErr := s.documentRepo.Update(ctx, doc); updateErr != nil {
		return fmt.Errorf("updating document %s: %w", doc.ID, updateErr)
	}

	s.updateRecipientsFromResult(ctx, recipients, result.Recipients)

	slog.InfoContext(ctx, "PENDING_PROVIDER document uploaded to signing provider",
		slog.String("document_id", doc.ID),
		slog.String("provider_doc_id", result.ProviderDocumentID),
	)
	return nil
}

// markDocError marks a document as ERROR and persists it (best-effort).
func (s *DocumentService) markDocError(ctx context.Context, doc *entity.Document) {
	_ = doc.MarkAsError()
	_ = s.documentRepo.Update(ctx, doc)
}

// buildSigningRecipients converts entity recipients to signing provider DTOs.
func buildSigningRecipients(recipients []*entity.DocumentRecipient) []port.SigningRecipient {
	out := make([]port.SigningRecipient, len(recipients))
	for i, r := range recipients {
		out[i] = port.SigningRecipient{
			Email:       r.Email,
			Name:        r.Name,
			RoleID:      r.TemplateVersionRoleID,
			SignerOrder: i + 1,
		}
	}
	return out
}

// updateRecipientsFromResult updates recipients with signing provider result data.
func (s *DocumentService) updateRecipientsFromResult(ctx context.Context, recipients []*entity.DocumentRecipient, results []port.RecipientResult) {
	byRoleID := make(map[string]port.RecipientResult, len(results))
	for _, r := range results {
		byRoleID[r.RoleID] = r
	}
	for _, recipient := range recipients {
		pr, ok := byRoleID[recipient.TemplateVersionRoleID]
		if !ok {
			continue
		}
		recipient.SetSignerRecipientID(pr.ProviderRecipientID)
		recipient.SetSigningURL(pr.SigningURL)
		if err := s.recipientRepo.Update(ctx, recipient); err != nil {
			slog.WarnContext(ctx, "failed to update recipient signing URL",
				slog.String("recipient_id", recipient.ID),
				slog.String("error", err.Error()),
			)
		}
	}
}

// GetDocumentsByExternalRef finds documents by the client's external reference ID.
func (s *DocumentService) GetDocumentsByExternalRef(ctx context.Context, workspaceID, externalRef string) ([]*entity.Document, error) {
	return s.documentRepo.FindByClientExternalRef(ctx, workspaceID, externalRef)
}

// GetDocumentRecipients retrieves all recipients for a document with their role information.
func (s *DocumentService) GetDocumentRecipients(ctx context.Context, documentID string) ([]*entity.DocumentRecipientWithRole, error) {
	return s.recipientRepo.FindByDocumentIDWithRoles(ctx, documentID)
}

// GetDocumentStatistics returns document statistics for a workspace.
func (s *DocumentService) GetDocumentStatistics(ctx context.Context, workspaceID string) (*documentuc.DocumentStatistics, error) {
	total, err := s.documentRepo.CountByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("counting documents: %w", err)
	}

	pending, err := s.documentRepo.CountByStatus(ctx, workspaceID, entity.DocumentStatusPending)
	if err != nil {
		return nil, fmt.Errorf("counting pending documents: %w", err)
	}

	inProgress, err := s.documentRepo.CountByStatus(ctx, workspaceID, entity.DocumentStatusInProgress)
	if err != nil {
		return nil, fmt.Errorf("counting in-progress documents: %w", err)
	}

	completed, err := s.documentRepo.CountByStatus(ctx, workspaceID, entity.DocumentStatusCompleted)
	if err != nil {
		return nil, fmt.Errorf("counting completed documents: %w", err)
	}

	declined, err := s.documentRepo.CountByStatus(ctx, workspaceID, entity.DocumentStatusDeclined)
	if err != nil {
		return nil, fmt.Errorf("counting declined documents: %w", err)
	}

	return &documentuc.DocumentStatistics{
		Total:      total,
		Pending:    pending,
		InProgress: inProgress,
		Completed:  completed,
		Declined:   declined,
		ByStatus: map[string]int{
			entity.DocumentStatusPending.String():    pending,
			entity.DocumentStatusInProgress.String(): inProgress,
			entity.DocumentStatusCompleted.String():  completed,
			entity.DocumentStatusDeclined.String():   declined,
		},
	}, nil
}

// downloadAndStorePDF downloads the signed PDF from the provider and stores it locally.
func (s *DocumentService) downloadAndStorePDF(ctx context.Context, doc *entity.Document) {
	if !doc.HasSignerInfo() {
		return
	}
	if doc.PDFStoragePath != nil {
		return
	}

	pdfData, err := s.signingProvider.DownloadSignedPDF(ctx, *doc.SignerDocumentID)
	if err != nil {
		slog.WarnContext(ctx, "failed to download signed PDF",
			slog.String("document_id", doc.ID),
			slog.String("error", err.Error()),
		)
		return
	}

	storageKey := fmt.Sprintf("documents/%s/%s/signed.pdf", doc.WorkspaceID, doc.ID)

	if err := s.storageAdapter.Upload(ctx, storageKey, pdfData, "application/pdf"); err != nil {
		slog.WarnContext(ctx, "failed to store signed PDF",
			slog.String("document_id", doc.ID),
			slog.String("error", err.Error()),
		)
		return
	}

	doc.SetPDFPath(storageKey)
	if err := s.documentRepo.Update(ctx, doc); err != nil {
		slog.WarnContext(ctx, "failed to update document PDF path",
			slog.String("document_id", doc.ID),
			slog.String("error", err.Error()),
		)
	}

	slog.InfoContext(ctx, "signed PDF stored",
		slog.String("document_id", doc.ID),
		slog.String("storage_key", storageKey),
	)
}

// GetDocumentPDF returns the signed PDF for a completed document.
func (s *DocumentService) GetDocumentPDF(ctx context.Context, documentID string) ([]byte, string, error) {
	doc, err := s.documentRepo.FindByID(ctx, documentID)
	if err != nil {
		return nil, "", fmt.Errorf("finding document: %w", err)
	}

	if doc.PDFStoragePath == nil {
		return nil, "", fmt.Errorf("signed PDF not available for this document")
	}

	data, err := s.storageAdapter.Download(ctx, *doc.PDFStoragePath)
	if err != nil {
		return nil, "", fmt.Errorf("downloading PDF from storage: %w", err)
	}

	filename := fmt.Sprintf("document-%s-signed.pdf", doc.ID)
	if doc.Title != nil {
		filename = fmt.Sprintf("%s-signed.pdf", *doc.Title)
	}

	return data, filename, nil
}

// ExpireDocuments finds and expires documents that have passed their expiration time.
func (s *DocumentService) ExpireDocuments(ctx context.Context, limit int) error {
	docs, err := s.documentRepo.FindExpired(ctx, limit)
	if err != nil {
		return fmt.Errorf("finding expired documents: %w", err)
	}

	for _, doc := range docs {
		s.expireSingleDocument(ctx, doc)
	}

	if len(docs) > 0 {
		slog.InfoContext(ctx, "expired documents processed", slog.Int("count", len(docs)))
	}

	return nil
}

// expireSingleDocument expires a single document and cancels it with the provider.
func (s *DocumentService) expireSingleDocument(ctx context.Context, doc *entity.Document) {
	oldStatus := string(doc.Status)

	if doc.HasSignerInfo() {
		if err := s.signingProvider.CancelDocument(ctx, *doc.SignerDocumentID); err != nil {
			slog.WarnContext(ctx, "failed to cancel expired document with provider",
				slog.String("document_id", doc.ID),
				slog.String("error", err.Error()),
			)
		}
	}

	if err := doc.MarkAsExpired(); err != nil {
		slog.WarnContext(ctx, "failed to mark document as expired",
			slog.String("document_id", doc.ID),
			slog.String("error", err.Error()),
		)
		return
	}

	if err := s.documentRepo.Update(ctx, doc); err != nil {
		slog.WarnContext(ctx, "failed to update expired document",
			slog.String("document_id", doc.ID),
			slog.String("error", err.Error()),
		)
		return
	}

	s.eventEmitter.EmitDocumentEvent(ctx, doc.ID, entity.EventDocumentExpired, entity.ActorSystem, "", oldStatus, string(entity.DocumentStatusExpired), nil)

	slog.InfoContext(ctx, "document expired",
		slog.String("document_id", doc.ID),
		slog.String("previous_status", oldStatus),
	)
}

// RetryErrorDocuments finds ERROR documents eligible for retry and attempts recovery.
func (s *DocumentService) RetryErrorDocuments(ctx context.Context, maxRetries, limit int) error {
	docs, err := s.documentRepo.FindErrorsForRetry(ctx, maxRetries, limit)
	if err != nil {
		return fmt.Errorf("finding error documents for retry: %w", err)
	}

	for _, doc := range docs {
		s.retrySingleDocument(ctx, doc, maxRetries)
	}

	if len(docs) > 0 {
		slog.InfoContext(ctx, "retried error documents", slog.Int("count", len(docs)))
	}

	return nil
}

// retrySingleDocument attempts to recover a single error document.
func (s *DocumentService) retrySingleDocument(ctx context.Context, doc *entity.Document, maxRetries int) {
	if !doc.ScheduleRetry(maxRetries) {
		slog.WarnContext(ctx, "document exceeded max retries",
			slog.String("document_id", doc.ID),
			slog.Int("retry_count", doc.RetryCount),
		)
		return
	}

	if doc.HasSignerInfo() {
		s.retryWithStatusPoll(ctx, doc)
	} else {
		slog.InfoContext(ctx, "retry skipped for document without provider reference",
			slog.String("document_id", doc.ID),
		)
	}
}

// retryWithStatusPoll polls the signing provider for a document that already has a signer ID.
func (s *DocumentService) retryWithStatusPoll(ctx context.Context, doc *entity.Document) {
	statusResult, err := s.signingProvider.GetDocumentStatus(ctx, *doc.SignerDocumentID)
	if err != nil {
		slog.WarnContext(ctx, "retry status poll failed",
			slog.String("document_id", doc.ID),
			slog.String("error", err.Error()),
		)
		if updateErr := s.documentRepo.Update(ctx, doc); updateErr != nil {
			slog.WarnContext(ctx, "failed to update document after retry", slog.String("error", updateErr.Error()))
		}
		return
	}

	if err := s.updateDocumentFromStatus(ctx, doc, statusResult); err != nil {
		slog.WarnContext(ctx, "failed to update document from status poll",
			slog.String("document_id", doc.ID),
			slog.String("error", err.Error()),
		)
		return
	}

	if !doc.IsTerminal() {
		doc.ResetRetry()
	}
	if err := s.documentRepo.Update(ctx, doc); err != nil {
		slog.WarnContext(ctx, "failed to update document after retry recovery",
			slog.String("document_id", doc.ID),
			slog.String("error", err.Error()),
		)
	}
}

// CreateDocumentsBatch creates multiple documents in a single batch.
func (s *DocumentService) CreateDocumentsBatch(ctx context.Context, cmds []documentuc.CreateDocumentCommand) ([]documentuc.BatchDocumentResult, error) {
	results := make([]documentuc.BatchDocumentResult, len(cmds))

	for i, cmd := range cmds {
		doc, err := s.CreateAndSendDocument(ctx, cmd)
		results[i] = documentuc.BatchDocumentResult{
			Index:    i,
			Document: doc,
			Error:    err,
		}
	}

	return results, nil
}

// eventTypeFromDocumentStatus maps a document status to the corresponding event type.
func eventTypeFromDocumentStatus(status entity.DocumentStatus) string {
	switch status {
	case entity.DocumentStatusCompleted:
		return entity.EventDocumentCompleted
	case entity.DocumentStatusVoided:
		return entity.EventDocumentCancelled
	case entity.DocumentStatusExpired:
		return entity.EventDocumentExpired
	case entity.DocumentStatusError:
		return entity.EventDocumentError
	default:
		return entity.EventStatusRefreshed
	}
}

// SendReminder sends reminder notifications to pending recipients of a document.
func (s *DocumentService) SendReminder(ctx context.Context, documentID string) error {
	return s.notificationSvc.SendReminder(ctx, documentID)
}

// parsePortableDocument is a helper to parse document content.
func parsePortableDocument(content json.RawMessage) (*portabledoc.Document, error) {
	return portabledoc.Parse(content)
}

// transitionToAwaitingInput marks the document as AWAITING_INPUT.
// Tokens are now generated on-demand via the email-verification flow (DocumentAccessService).
func (s *DocumentService) transitionToAwaitingInput(
	ctx context.Context,
	document *entity.Document,
) error {
	if err := document.MarkAsAwaitingInput(); err != nil {
		return fmt.Errorf("marking document as awaiting input: %w", err)
	}

	if err := s.documentRepo.Update(ctx, document); err != nil {
		return fmt.Errorf("updating document to awaiting input: %w", err)
	}

	s.eventEmitter.EmitDocumentEvent(ctx, document.ID, entity.EventDocumentCreated, entity.ActorUser, "", "", string(entity.DocumentStatusDraft), nil)

	slog.InfoContext(ctx, "document awaiting input",
		slog.String("document_id", document.ID),
	)

	return nil
}

// generateAccessToken creates a cryptographically random hex-encoded token (128 chars).
func generateAccessToken() (string, error) {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Verify DocumentService implements DocumentUseCase
var _ documentuc.DocumentUseCase = (*DocumentService)(nil)
