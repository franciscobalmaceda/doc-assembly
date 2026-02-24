package document

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/rendis/doc-assembly/core/internal/core/entity"
	"github.com/rendis/doc-assembly/core/internal/core/entity/portabledoc"
	"github.com/rendis/doc-assembly/core/internal/core/port"
	documentuc "github.com/rendis/doc-assembly/core/internal/core/usecase/document"
)

// PreSigningService implements the public signing use case.
type PreSigningService struct {
	accessTokenRepo   port.DocumentAccessTokenRepository
	fieldResponseRepo port.DocumentFieldResponseRepository
	documentRepo      port.DocumentRepository
	recipientRepo     port.DocumentRecipientRepository
	versionRepo       port.TemplateVersionRepository
	signerRoleRepo    port.TemplateVersionSignerRoleRepository
	pdfRenderer       port.PDFRenderer
	signingProvider   port.SigningProvider
	storageAdapter    port.StorageAdapter
	eventEmitter      *EventEmitter
	publicURL         string
}

// NewPreSigningService creates a new PreSigningService.
func NewPreSigningService(
	accessTokenRepo port.DocumentAccessTokenRepository,
	fieldResponseRepo port.DocumentFieldResponseRepository,
	documentRepo port.DocumentRepository,
	recipientRepo port.DocumentRecipientRepository,
	versionRepo port.TemplateVersionRepository,
	signerRoleRepo port.TemplateVersionSignerRoleRepository,
	pdfRenderer port.PDFRenderer,
	signingProvider port.SigningProvider,
	storageAdapter port.StorageAdapter,
	eventEmitter *EventEmitter,
	publicURL string,
) documentuc.PreSigningUseCase {
	return &PreSigningService{
		accessTokenRepo:   accessTokenRepo,
		fieldResponseRepo: fieldResponseRepo,
		documentRepo:      documentRepo,
		recipientRepo:     recipientRepo,
		versionRepo:       versionRepo,
		signerRoleRepo:    signerRoleRepo,
		pdfRenderer:       pdfRenderer,
		signingProvider:   signingProvider,
		storageAdapter:    storageAdapter,
		eventEmitter:      eventEmitter,
		publicURL:         publicURL,
	}
}

// GetPublicSigningPage returns the current signing page state based on document status and token type.
func (s *PreSigningService) GetPublicSigningPage(ctx context.Context, token string) (*documentuc.PublicSigningResponse, error) {
	accessToken, wasUsed, err := s.validateTokenAllowUsed(ctx, token)
	if err != nil {
		return nil, err
	}

	doc, err := s.documentRepo.FindByID(ctx, accessToken.DocumentID)
	if err != nil {
		return nil, fmt.Errorf("finding document: %w", err)
	}

	recipient, err := s.recipientRepo.FindByID(ctx, accessToken.RecipientID)
	if err != nil {
		return nil, fmt.Errorf("finding recipient: %w", err)
	}

	title := documentTitle(doc)

	// Terminal states — always accessible, even with used tokens.
	if doc.IsCompleted() {
		resp := &documentuc.PublicSigningResponse{
			Step:          documentuc.StepCompleted,
			DocumentTitle: title,
			RecipientName: recipient.Name,
		}
		s.applyAccessFlags(resp, doc, recipient, accessToken.Token)
		return resp, nil
	}
	if doc.IsDeclined() {
		resp := &documentuc.PublicSigningResponse{
			Step:          documentuc.StepDeclined,
			DocumentTitle: title,
			RecipientName: recipient.Name,
		}
		s.applyAccessFlags(resp, doc, recipient, accessToken.Token)
		return resp, nil
	}

	// Non-terminal states require an unused token.
	if wasUsed {
		return nil, fmt.Errorf("access token has already been used")
	}

	// Path B: AWAITING_INPUT + PRE_SIGNING token.
	if doc.IsAwaitingInput() && accessToken.IsPreSigning() {
		// If field responses already saved → show PDF preview; otherwise show form.
		if s.hasFieldResponses(ctx, doc.ID) {
			return s.buildPreviewPDFResponse(doc, recipient, title, accessToken.Token)
		}
		return s.buildPreviewFormResponse(ctx, doc, recipient, title, accessToken.Token)
	}

	// Path A: AWAITING_INPUT + SIGNING token → show PDF preview.
	if doc.IsAwaitingInput() && accessToken.IsSigning() {
		return s.buildPreviewPDFResponse(doc, recipient, title, accessToken.Token)
	}

	// Doc already sent to provider → show signing iframe.
	if doc.IsPending() || doc.IsInProgress() {
		return s.buildSigningResponse(ctx, doc, recipient, accessToken, title)
	}

	// Doc is being rendered/uploaded by another goroutine → tell the signer to retry.
	if doc.IsPendingProvider() {
		return s.buildProcessingResponse(doc, recipient, accessToken.Token), nil
	}

	return nil, fmt.Errorf("document is not in a valid state for signing")
}

// SubmitPreSigningForm validates responses, saves them, renders PDF, sends to provider,
// and returns the signing page state with embedded URL.
func (s *PreSigningService) SubmitPreSigningForm(
	ctx context.Context,
	token string,
	responses []documentuc.FieldResponseInput,
) (*documentuc.PublicSigningResponse, error) {
	accessToken, err := s.validateToken(ctx, token)
	if err != nil {
		return nil, err
	}

	doc, recipient, version, portableDoc, err := s.loadSubmissionContext(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	// Map DB role to portable doc role for field extraction.
	portableRoleID := s.findPortableRoleID(ctx, version, portableDoc, recipient.TemplateVersionRoleID)
	fieldDefs := extractInteractiveFieldsForRole(portableDoc, portableRoleID)

	// Validate responses against field definitions.
	if err := s.validateResponses(responses, fieldDefs); err != nil {
		return nil, err
	}

	// Save responses (no render/upload yet — deferred to ProceedToSigning).
	if err := s.saveFieldResponses(ctx, doc.ID, recipient.ID, responses, fieldDefs); err != nil {
		return nil, fmt.Errorf("saving field responses: %w", err)
	}

	slog.InfoContext(ctx, "pre-signing form submitted, responses saved",
		slog.String("document_id", doc.ID),
		slog.String("recipient_id", recipient.ID),
	)

	// Return PDF preview step so the signer can review before proceeding.
	title := documentTitle(doc)
	return s.buildPreviewPDFResponse(doc, recipient, title, accessToken.Token)
}

// ProceedToSigning renders the PDF, uploads to the signing provider, and returns the
// embedded signing URL. Accepts both SIGNING (Path A) and PRE_SIGNING (Path B) tokens.
//
//nolint:gocognit,nestif // Flow is sequential by token/document state to keep behavior explicit.
func (s *PreSigningService) ProceedToSigning(ctx context.Context, token string) (*documentuc.PublicSigningResponse, error) {
	accessToken, err := s.validateToken(ctx, token)
	if err != nil {
		return nil, err
	}

	doc, err := s.documentRepo.FindByID(ctx, accessToken.DocumentID)
	if err != nil {
		return nil, fmt.Errorf("finding document: %w", err)
	}

	recipient, err := s.recipientRepo.FindByID(ctx, accessToken.RecipientID)
	if err != nil {
		return nil, fmt.Errorf("finding recipient: %w", err)
	}

	// If document is still AWAITING_INPUT, atomically claim and render + upload.
	// Only one concurrent caller wins the CAS; losers reload and fall through.
	if doc.IsAwaitingInput() {
		var claimErr error
		doc, recipient, claimErr = s.claimAndRender(ctx, doc, accessToken)
		if claimErr != nil {
			return nil, claimErr
		}
	}

	// Document is being rendered/uploaded by another goroutine — tell the signer to retry.
	if doc.IsPendingProvider() && !doc.HasSignerInfo() {
		return s.buildProcessingResponse(doc, recipient, accessToken.Token), nil
	}

	if !doc.IsPending() && !doc.IsInProgress() && !doc.IsPendingProvider() {
		return nil, fmt.Errorf("document is not pending signing")
	}

	// Check signing order.
	if waitResp := s.checkSigningOrder(ctx, doc, recipient, accessToken.Token); waitResp != nil {
		return waitResp, nil
	}

	title := documentTitle(doc)
	return s.buildSigningResponse(ctx, doc, recipient, accessToken, title)
}

// claimAndRender atomically claims the document for rendering via CAS, then renders and
// uploads to the signing provider. If another goroutine already claimed it, reloads the
// current document state. Returns the refreshed document and recipient.
func (s *PreSigningService) claimAndRender(
	ctx context.Context,
	doc *entity.Document,
	accessToken *entity.DocumentAccessToken,
) (*entity.Document, *entity.DocumentRecipient, error) {
	claimedDoc, claimed, err := s.documentRepo.ClaimForSigning(ctx, doc.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("claiming document for signing: %w", err)
	}

	if !claimed {
		// Another goroutine claimed it. Reload current state.
		reloaded, err := s.documentRepo.FindByID(ctx, accessToken.DocumentID)
		if err != nil {
			return nil, nil, fmt.Errorf("reloading document after concurrent claim: %w", err)
		}
		recipient, err := s.recipientRepo.FindByID(ctx, accessToken.RecipientID)
		if err != nil {
			return nil, nil, fmt.Errorf("reloading recipient after concurrent claim: %w", err)
		}
		return reloaded, recipient, nil
	}

	doc = claimedDoc

	version, err := s.versionRepo.FindByID(ctx, doc.TemplateVersionID)
	if err != nil {
		return nil, nil, fmt.Errorf("finding template version: %w", err)
	}
	portableDoc, err := parsePortableDocument(version.ContentStructure)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing document content: %w", err)
	}
	fieldResponses := loadFieldResponseMap(ctx, s.fieldResponseRepo, doc.ID)

	if err := s.renderAndSendToProvider(ctx, doc, version, portableDoc, fieldResponses); err != nil {
		return nil, nil, err
	}

	// Refresh entities after upload since provider identifiers are persisted.
	doc, err = s.documentRepo.FindByID(ctx, accessToken.DocumentID)
	if err != nil {
		return nil, nil, fmt.Errorf("refreshing document after provider upload: %w", err)
	}
	recipient, err := s.recipientRepo.FindByID(ctx, accessToken.RecipientID)
	if err != nil {
		return nil, nil, fmt.Errorf("refreshing recipient after provider upload: %w", err)
	}

	slog.InfoContext(ctx, "document rendered and sent to provider via ProceedToSigning",
		slog.String("document_id", doc.ID),
		slog.String("recipient_id", recipient.ID),
	)

	return doc, recipient, nil
}

// CompleteEmbeddedSigning marks the token as used after embedded signing is completed.
func (s *PreSigningService) CompleteEmbeddedSigning(ctx context.Context, token string) error {
	accessToken, err := s.validateToken(ctx, token)
	if err != nil {
		return err
	}

	if err := s.accessTokenRepo.MarkAsUsed(ctx, accessToken.ID); err != nil {
		return fmt.Errorf("marking token as used: %w", err)
	}

	slog.InfoContext(ctx, "embedded signing completed, token marked as used",
		slog.String("document_id", accessToken.DocumentID),
		slog.String("recipient_id", accessToken.RecipientID),
	)

	return nil
}

// RefreshEmbeddedURL refreshes an expired embedded signing URL.
func (s *PreSigningService) RefreshEmbeddedURL(ctx context.Context, token string) (*documentuc.PublicSigningResponse, error) {
	accessToken, err := s.validateToken(ctx, token)
	if err != nil {
		return nil, err
	}

	doc, err := s.documentRepo.FindByID(ctx, accessToken.DocumentID)
	if err != nil {
		return nil, fmt.Errorf("finding document: %w", err)
	}

	recipient, err := s.recipientRepo.FindByID(ctx, accessToken.RecipientID)
	if err != nil {
		return nil, fmt.Errorf("finding recipient: %w", err)
	}

	title := documentTitle(doc)
	return s.buildSigningResponse(ctx, doc, recipient, accessToken, title)
}

// InvalidateTokens invalidates all active tokens for a document in AWAITING_INPUT status.
func (s *PreSigningService) InvalidateTokens(ctx context.Context, documentID string) error {
	doc, err := s.documentRepo.FindByID(ctx, documentID)
	if err != nil {
		return fmt.Errorf("finding document: %w", err)
	}
	if !doc.IsAwaitingInput() {
		return fmt.Errorf("document is not in AWAITING_INPUT status")
	}

	if err := s.accessTokenRepo.InvalidateByDocumentID(ctx, documentID); err != nil {
		return fmt.Errorf("invalidating existing tokens: %w", err)
	}

	slog.InfoContext(ctx, "access tokens invalidated",
		slog.String("document_id", documentID),
	)

	return nil
}

// RenderPreviewPDF renders the document PDF on-demand for preview without storing it.
func (s *PreSigningService) RenderPreviewPDF(ctx context.Context, token string) ([]byte, error) {
	accessToken, wasUsed, err := s.validateTokenAllowUsed(ctx, token)
	if err != nil {
		return nil, err
	}

	doc, err := s.documentRepo.FindByID(ctx, accessToken.DocumentID)
	if err != nil {
		return nil, fmt.Errorf("finding document: %w", err)
	}
	if wasUsed && !doc.IsCompleted() && !doc.IsDeclined() {
		return nil, fmt.Errorf("access token has already been used")
	}

	version, err := s.versionRepo.FindByID(ctx, doc.TemplateVersionID)
	if err != nil {
		return nil, fmt.Errorf("finding template version: %w", err)
	}

	portableDoc, err := parsePortableDocument(version.ContentStructure)
	if err != nil {
		return nil, fmt.Errorf("parsing document content: %w", err)
	}
	if portableDoc == nil {
		return nil, fmt.Errorf("document has no content")
	}

	recipients, err := s.recipientRepo.FindByDocumentID(ctx, doc.ID)
	if err != nil {
		return nil, fmt.Errorf("loading recipients: %w", err)
	}

	signerRoles, err := s.signerRoleRepo.FindByVersionID(ctx, doc.TemplateVersionID)
	if err != nil {
		return nil, fmt.Errorf("loading signer roles: %w", err)
	}

	signerRoleValues := buildSignerRoleValues(recipients, signerRoles, portableDoc.SignerRoles)

	var injectables map[string]any
	if doc.InjectedValuesSnapshot != nil {
		_ = json.Unmarshal(doc.InjectedValuesSnapshot, &injectables)
	}

	fieldResponses := loadFieldResponseMap(ctx, s.fieldResponseRepo, doc.ID)

	renderResult, err := s.pdfRenderer.RenderPreview(ctx, &port.RenderPreviewRequest{
		Document:         portableDoc,
		Injectables:      injectables,
		SignerRoleValues: signerRoleValues,
		FieldResponses:   fieldResponses,
	})
	if err != nil {
		return nil, fmt.Errorf("rendering preview PDF: %w", err)
	}

	return renderResult.PDF, nil
}

// DownloadCompletedPDF returns the signed PDF for completed documents when the
// token recipient is authorized.
//
//nolint:gocognit,nestif // Explicit guard flow preserves security/state checks.
func (s *PreSigningService) DownloadCompletedPDF(ctx context.Context, token string) ([]byte, string, error) {
	accessToken, _, err := s.validateTokenAllowUsed(ctx, token)
	if err != nil {
		return nil, "", err
	}

	doc, err := s.documentRepo.FindByID(ctx, accessToken.DocumentID)
	if err != nil {
		return nil, "", fmt.Errorf("finding document: %w", err)
	}

	if !doc.IsCompleted() {
		return nil, "", fmt.Errorf("document is not completed")
	}

	recipient, err := s.recipientRepo.FindByID(ctx, accessToken.RecipientID)
	if err != nil {
		return nil, "", fmt.Errorf("finding recipient: %w", err)
	}

	if !recipient.IsSigned() {
		return nil, "", fmt.Errorf("completed PDF is not available for this recipient")
	}

	if doc.PDFStoragePath == nil || *doc.PDFStoragePath == "" {
		if !doc.HasSignerInfo() {
			return nil, "", fmt.Errorf("completed PDF is not available")
		}

		pdfData, err := s.signingProvider.DownloadSignedPDF(ctx, *doc.SignerDocumentID)
		if err != nil {
			return nil, "", fmt.Errorf("downloading completed PDF: %w", err)
		}

		storageKey := fmt.Sprintf("documents/%s/%s/signed.pdf", doc.WorkspaceID, doc.ID)
		if err := s.storageAdapter.Upload(ctx, storageKey, pdfData, "application/pdf"); err != nil {
			return nil, "", fmt.Errorf("storing completed PDF: %w", err)
		}

		doc.SetPDFPath(storageKey)
		if err := s.documentRepo.Update(ctx, doc); err != nil {
			slog.WarnContext(ctx, "failed to persist completed PDF storage path",
				slog.String("document_id", doc.ID),
				slog.String("error", err.Error()),
			)
		}
	}

	pdfData, err := s.storageAdapter.Download(ctx, *doc.PDFStoragePath)
	if err != nil {
		return nil, "", fmt.Errorf("downloading completed PDF: %w", err)
	}

	filename := fmt.Sprintf("document-%s-signed.pdf", doc.ID)
	if doc.Title != nil {
		filename = fmt.Sprintf("%s-signed.pdf", *doc.Title)
	}

	return pdfData, filename, nil
}

// hasFieldResponses checks whether field responses have been saved for a document.
func (s *PreSigningService) hasFieldResponses(ctx context.Context, documentID string) bool {
	responses, err := s.fieldResponseRepo.FindByDocumentID(ctx, documentID)
	if err != nil {
		return false
	}
	return len(responses) > 0
}

// --- Response builders ---

// buildPreviewFormResponse builds a preview response with the pre-signing form (Path B).
func (s *PreSigningService) buildPreviewFormResponse(
	ctx context.Context,
	doc *entity.Document,
	recipient *entity.DocumentRecipient,
	title string,
	token string,
) (*documentuc.PublicSigningResponse, error) {
	version, err := s.versionRepo.FindByID(ctx, doc.TemplateVersionID)
	if err != nil {
		return nil, fmt.Errorf("finding template version: %w", err)
	}

	portableDoc, err := parsePortableDocument(version.ContentStructure)
	if err != nil {
		return nil, fmt.Errorf("parsing document content: %w", err)
	}
	if portableDoc == nil {
		return nil, fmt.Errorf("document has no content")
	}

	content, err := s.resolveContent(portableDoc, doc)
	if err != nil {
		return nil, fmt.Errorf("resolving content: %w", err)
	}

	portableRoleID := s.findPortableRoleID(ctx, version, portableDoc, recipient.TemplateVersionRoleID)
	fields := extractInteractiveFieldsForRole(portableDoc, portableRoleID)

	resp := &documentuc.PublicSigningResponse{
		Step:          documentuc.StepPreview,
		DocumentTitle: title,
		RecipientName: recipient.Name,
		Form: &documentuc.PreSigningFormDTO{
			DocumentTitle:  title,
			DocumentStatus: string(doc.Status),
			RecipientName:  recipient.Name,
			RecipientEmail: recipient.Email,
			RoleID:         recipient.TemplateVersionRoleID,
			Content:        content,
			Fields:         fields,
		},
	}
	s.applyAccessFlags(resp, doc, recipient, token)
	return resp, nil
}

// buildProcessingResponse returns a "processing" step when the document is being
// rendered/uploaded and signer info is not yet available.
func (s *PreSigningService) buildProcessingResponse(
	doc *entity.Document,
	recipient *entity.DocumentRecipient,
	token string,
) *documentuc.PublicSigningResponse {
	title := documentTitle(doc)
	resp := &documentuc.PublicSigningResponse{
		Step:          documentuc.StepProcessing,
		DocumentTitle: title,
		RecipientName: recipient.Name,
	}
	s.applyAccessFlags(resp, doc, recipient, token)
	return resp
}

// buildPreviewPDFResponse builds a preview response with the PDF URL for on-demand rendering.
func (s *PreSigningService) buildPreviewPDFResponse(
	doc *entity.Document,
	recipient *entity.DocumentRecipient,
	title string,
	token string,
) (*documentuc.PublicSigningResponse, error) {
	resp := &documentuc.PublicSigningResponse{
		Step:          documentuc.StepPreview,
		DocumentTitle: title,
		RecipientName: recipient.Name,
		PdfURL:        fmt.Sprintf("/public/sign/%s/pdf", token),
	}
	s.applyAccessFlags(resp, doc, recipient, token)
	return resp, nil
}

// buildSigningResponse builds a signing response with the embedded signing URL.
func (s *PreSigningService) buildSigningResponse(
	ctx context.Context,
	doc *entity.Document,
	recipient *entity.DocumentRecipient,
	accessToken *entity.DocumentAccessToken,
	title string,
) (*documentuc.PublicSigningResponse, error) {
	if recipient.SignerRecipientID == nil || doc.SignerDocumentID == nil {
		return nil, fmt.Errorf("document is not registered with a signing provider")
	}

	callbackURL := s.buildCallbackURL(accessToken.Token)

	embeddedResult, err := s.signingProvider.GetEmbeddedSigningURL(ctx, &port.GetEmbeddedSigningURLRequest{
		ProviderDocumentID:  *doc.SignerDocumentID,
		ProviderRecipientID: *recipient.SignerRecipientID,
		CallbackURL:         callbackURL,
	})
	if err != nil {
		// Fallback: if embedding not supported, return direct URL.
		if recipient.SigningURL != nil {
			resp := &documentuc.PublicSigningResponse{
				Step:          documentuc.StepSigning,
				DocumentTitle: title,
				RecipientName: recipient.Name,
				FallbackURL:   *recipient.SigningURL,
			}
			s.applyAccessFlags(resp, doc, recipient, accessToken.Token)
			return resp, nil
		}
		return nil, fmt.Errorf("getting embedded signing URL: %w", err)
	}

	resp := &documentuc.PublicSigningResponse{
		Step:               documentuc.StepSigning,
		DocumentTitle:      title,
		RecipientName:      recipient.Name,
		EmbeddedSigningURL: embeddedResult.EmbeddedURL,
	}
	s.applyAccessFlags(resp, doc, recipient, accessToken.Token)
	return resp, nil
}

// checkSigningOrder checks if previous signers have completed. Returns a waiting response or nil.
func (s *PreSigningService) checkSigningOrder(
	ctx context.Context,
	doc *entity.Document,
	recipient *entity.DocumentRecipient,
	token string,
) *documentuc.PublicSigningResponse {
	recipientsWithRoles, err := s.recipientRepo.FindByDocumentIDWithRoles(ctx, doc.ID)
	if err != nil {
		return nil
	}

	// Find this recipient's order.
	var myOrder int
	for _, r := range recipientsWithRoles {
		if r.ID == recipient.ID {
			myOrder = r.SignerOrder
			break
		}
	}

	// Order 0 or 1 means first signer — no need to wait.
	if myOrder <= 1 {
		return nil
	}

	for _, r := range recipientsWithRoles {
		if r.SignerOrder < myOrder && !r.IsSigned() {
			title := documentTitle(doc)
			resp := &documentuc.PublicSigningResponse{
				Step:               documentuc.StepWaiting,
				DocumentTitle:      title,
				RecipientName:      recipient.Name,
				WaitingForPrevious: true,
				SigningPosition:    myOrder,
				TotalSigners:       len(recipientsWithRoles),
			}
			s.applyAccessFlags(resp, doc, recipient, token)
			resp.CanSign = false
			return resp
		}
	}

	return nil
}

func (s *PreSigningService) applyAccessFlags(
	resp *documentuc.PublicSigningResponse,
	doc *entity.Document,
	recipient *entity.DocumentRecipient,
	token string,
) {
	resp.DocumentStatus = string(doc.Status)
	resp.HasCurrentUserSigned = recipient.IsSigned()
	resp.CanSign = doc.IsAwaitingInput() || doc.IsPending() || doc.IsInProgress()
	resp.CanDownload = doc.IsCompleted() && recipient.IsSigned()
	if resp.CanDownload {
		resp.DownloadURL = fmt.Sprintf("/public/sign/%s/download", token)
	}
}

// buildCallbackURL constructs the signing callback bridge URL.
func (s *PreSigningService) buildCallbackURL(token string) string {
	return fmt.Sprintf("%s/public/sign/%s/signing-callback", s.publicURL, token)
}

// --- Existing internal helpers (unchanged logic) ---

// loadSubmissionContext loads the document, recipient, version, and portable doc for a form submission.
func (s *PreSigningService) loadSubmissionContext(
	ctx context.Context,
	accessToken *entity.DocumentAccessToken,
) (*entity.Document, *entity.DocumentRecipient, *entity.TemplateVersion, *portabledoc.Document, error) {
	doc, err := s.documentRepo.FindByID(ctx, accessToken.DocumentID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("finding document: %w", err)
	}
	if !doc.IsAwaitingInput() {
		return nil, nil, nil, nil, fmt.Errorf("document is not awaiting input")
	}

	recipient, err := s.recipientRepo.FindByID(ctx, accessToken.RecipientID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("finding recipient: %w", err)
	}

	version, err := s.versionRepo.FindByID(ctx, doc.TemplateVersionID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("finding template version: %w", err)
	}

	portableDoc, err := parsePortableDocument(version.ContentStructure)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("parsing document content: %w", err)
	}
	if portableDoc == nil {
		return nil, nil, nil, nil, fmt.Errorf("document has no content")
	}

	return doc, recipient, version, portableDoc, nil
}

// validateToken checks that a token exists, is not expired, and is not used.
func (s *PreSigningService) validateToken(ctx context.Context, token string) (*entity.DocumentAccessToken, error) {
	if token == "" {
		return nil, entity.ErrMissingToken
	}

	accessToken, err := s.accessTokenRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, entity.ErrInvalidToken
	}

	if accessToken.UsedAt != nil {
		return nil, fmt.Errorf("access token has already been used")
	}

	if time.Now().UTC().After(accessToken.ExpiresAt) {
		return nil, entity.ErrTokenExpired
	}

	return accessToken, nil
}

// validateTokenAllowUsed is like validateToken but permits already-used tokens.
// Returns the token and whether it was already used. Expiry is only enforced
// for unused tokens — a used token can always proceed so GetPublicSigningPage
// can display terminal states (completed/declined).
func (s *PreSigningService) validateTokenAllowUsed(ctx context.Context, token string) (*entity.DocumentAccessToken, bool, error) {
	if token == "" {
		return nil, false, entity.ErrMissingToken
	}

	accessToken, err := s.accessTokenRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, false, entity.ErrInvalidToken
	}

	wasUsed := accessToken.UsedAt != nil

	if !wasUsed && time.Now().UTC().After(accessToken.ExpiresAt) {
		return nil, false, entity.ErrTokenExpired
	}

	return accessToken, wasUsed, nil
}

// resolveContent serializes the portable doc content with injected values applied.
func (s *PreSigningService) resolveContent(portableDoc *portabledoc.Document, doc *entity.Document) (json.RawMessage, error) {
	if portableDoc.Content == nil {
		return nil, fmt.Errorf("document has no ProseMirror content")
	}
	return json.Marshal(portableDoc.Content)
}

// findPortableRoleID maps a DB signer role ID to the portable doc role ID.
func (s *PreSigningService) findPortableRoleID(
	ctx context.Context,
	version *entity.TemplateVersion,
	portableDoc *portabledoc.Document,
	dbRoleID string,
) string {
	signerRoles, err := s.signerRoleRepo.FindByVersionID(ctx, version.ID)
	if err != nil {
		return ""
	}

	var anchor string
	for _, role := range signerRoles {
		if role.ID == dbRoleID {
			anchor = role.AnchorString
			break
		}
	}
	if anchor == "" {
		return ""
	}

	for _, sr := range portableDoc.SignerRoles {
		if portabledoc.GenerateAnchorString(sr.Label) == anchor {
			return sr.ID
		}
	}

	return ""
}

// extractInteractiveFieldsForRole extracts interactive fields for a specific role.
func extractInteractiveFieldsForRole(doc *portabledoc.Document, roleID string) []documentuc.InteractiveFieldDTO {
	fields := make([]documentuc.InteractiveFieldDTO, 0, 8)

	for _, node := range doc.CollectNodesOfType(portabledoc.NodeTypeInteractiveField) {
		attrs, err := portabledoc.ParseInteractiveFieldAttrs(node.Attrs)
		if err != nil {
			continue
		}

		if roleID != "" && attrs.RoleID != roleID {
			continue
		}

		fields = append(fields, documentuc.InteractiveFieldDTO{
			ID:          attrs.ID,
			FieldType:   attrs.FieldType,
			Label:       attrs.Label,
			Required:    attrs.Required,
			Options:     attrs.Options,
			Placeholder: attrs.Placeholder,
			MaxLength:   attrs.MaxLength,
		})
	}

	return fields
}

// validateResponses validates all field responses against field definitions.
func (s *PreSigningService) validateResponses(
	responses []documentuc.FieldResponseInput,
	fieldDefs []documentuc.InteractiveFieldDTO,
) error {
	defByID := make(map[string]documentuc.InteractiveFieldDTO, len(fieldDefs))
	for _, def := range fieldDefs {
		defByID[def.ID] = def
	}

	submittedIDs := make(map[string]bool, len(responses))
	for _, resp := range responses {
		submittedIDs[resp.FieldID] = true
	}

	for _, def := range fieldDefs {
		if def.Required && !submittedIDs[def.ID] {
			return fmt.Errorf("required field %q (%s) is missing", def.Label, def.ID)
		}
	}

	for _, resp := range responses {
		def, ok := defByID[resp.FieldID]
		if !ok {
			return fmt.Errorf("unknown field ID: %s", resp.FieldID)
		}

		if err := validateSingleResponse(resp, def); err != nil {
			return fmt.Errorf("field %q (%s): %w", def.Label, def.ID, err)
		}
	}

	return nil
}

// validateSingleResponse validates a single field response against its definition.
func validateSingleResponse(resp documentuc.FieldResponseInput, def documentuc.InteractiveFieldDTO) error {
	switch def.FieldType {
	case portabledoc.InteractiveFieldTypeCheckbox, portabledoc.InteractiveFieldTypeRadio:
		return validateSelectionResponse(resp.Response, def)
	case portabledoc.InteractiveFieldTypeText:
		return validateTextResponse(resp.Response, def)
	default:
		return fmt.Errorf("unsupported field type: %s", def.FieldType)
	}
}

type selectionResponse struct {
	SelectedOptionIDs []string `json:"selectedOptionIds"`
}

type textResponse struct {
	Text string `json:"text"`
}

func validateSelectionResponse(responseJSON json.RawMessage, def documentuc.InteractiveFieldDTO) error {
	var resp selectionResponse
	if err := json.Unmarshal(responseJSON, &resp); err != nil {
		return fmt.Errorf("invalid response format: expected {\"selectedOptionIds\":[...]}")
	}

	if def.FieldType == portabledoc.InteractiveFieldTypeRadio && len(resp.SelectedOptionIDs) > 1 {
		return fmt.Errorf("radio field must have at most one selected option")
	}

	validIDs := make(map[string]bool, len(def.Options))
	for _, opt := range def.Options {
		validIDs[opt.ID] = true
	}

	for _, id := range resp.SelectedOptionIDs {
		if !validIDs[id] {
			return fmt.Errorf("invalid option ID: %s", id)
		}
	}

	return nil
}

func validateTextResponse(responseJSON json.RawMessage, def documentuc.InteractiveFieldDTO) error {
	var resp textResponse
	if err := json.Unmarshal(responseJSON, &resp); err != nil {
		return fmt.Errorf("invalid response format: expected {\"text\":\"...\"}")
	}

	if def.MaxLength > 0 && len(resp.Text) > def.MaxLength {
		return fmt.Errorf("text exceeds maximum length of %d characters", def.MaxLength)
	}

	return nil
}

func (s *PreSigningService) saveFieldResponses(
	ctx context.Context,
	documentID, recipientID string,
	responses []documentuc.FieldResponseInput,
	fieldDefs []documentuc.InteractiveFieldDTO,
) error {
	typeByID := make(map[string]string, len(fieldDefs))
	for _, def := range fieldDefs {
		typeByID[def.ID] = def.FieldType
	}

	for _, resp := range responses {
		fieldResponse := &entity.DocumentFieldResponse{
			ID:          uuid.NewString(),
			DocumentID:  documentID,
			RecipientID: recipientID,
			FieldID:     resp.FieldID,
			FieldType:   typeByID[resp.FieldID],
			Response:    resp.Response,
			CreatedAt:   time.Now().UTC(),
		}

		if err := s.fieldResponseRepo.Create(ctx, fieldResponse); err != nil {
			return fmt.Errorf("creating field response for field %s: %w", resp.FieldID, err)
		}
	}

	return nil
}

// renderAndSendToProvider renders the PDF with field responses and sends to the signing provider.
func (s *PreSigningService) renderAndSendToProvider(
	ctx context.Context,
	doc *entity.Document,
	version *entity.TemplateVersion,
	portableDoc *portabledoc.Document,
	fieldResponses map[string]json.RawMessage,
) error {
	recipients, err := s.recipientRepo.FindByDocumentID(ctx, doc.ID)
	if err != nil {
		return fmt.Errorf("loading recipients: %w", err)
	}

	signerRoles, err := s.signerRoleRepo.FindByVersionID(ctx, doc.TemplateVersionID)
	if err != nil {
		return fmt.Errorf("loading signer roles: %w", err)
	}

	signerRoleValues := buildSignerRoleValues(recipients, signerRoles, portableDoc.SignerRoles)

	var injectables map[string]any
	if doc.InjectedValuesSnapshot != nil {
		_ = json.Unmarshal(doc.InjectedValuesSnapshot, &injectables)
	}

	renderResult, err := s.pdfRenderer.RenderPreview(ctx, &port.RenderPreviewRequest{
		Document:         portableDoc,
		Injectables:      injectables,
		SignerRoleValues: signerRoleValues,
		FieldResponses:   fieldResponses,
	})
	if err != nil {
		return fmt.Errorf("rendering PDF: %w", err)
	}

	storagePath := fmt.Sprintf("documents/%s/%s/pre-signed.pdf", doc.WorkspaceID, doc.ID)
	if err := s.storageAdapter.Upload(ctx, storagePath, renderResult.PDF, "application/pdf"); err != nil {
		return fmt.Errorf("storing PDF: %w", err)
	}
	doc.SetPDFPath(storagePath)

	// Status is already PENDING_PROVIDER (set by ClaimForSigning CAS).
	// Persist the PDF storage path as a durable checkpoint for the background worker.
	if err := s.documentRepo.Update(ctx, doc); err != nil {
		return fmt.Errorf("persisting PDF storage path: %w", err)
	}

	return s.uploadToProvider(ctx, doc, recipients, signerRoles, portableDoc, renderResult)
}

// uploadToProvider uploads the rendered PDF to the signing provider and finalizes the document.
func (s *PreSigningService) uploadToProvider(
	ctx context.Context,
	doc *entity.Document,
	recipients []*entity.DocumentRecipient,
	signerRoles []*entity.TemplateVersionSignerRole,
	portableDoc *portabledoc.Document,
	renderResult *port.RenderPreviewResult,
) error {
	title := documentTitle(doc)

	sigFields := mapSignatureFieldPositions(renderResult.SignatureFields, signerRoles, portableDoc.SignerRoles)
	if len(sigFields) == 0 {
		sigFields = buildDefaultSignatureFieldPositions(recipients)
	}

	uploadResult, err := s.signingProvider.UploadDocument(ctx, &port.UploadDocumentRequest{
		PDF:             renderResult.PDF,
		Title:           title,
		Recipients:      buildSigningRecipients(recipients),
		ExternalRef:     doc.ID,
		SignatureFields: sigFields,
	})
	if err != nil {
		_ = doc.MarkAsError()
		_ = s.documentRepo.Update(ctx, doc)
		return fmt.Errorf("uploading to signing provider: %w", err)
	}

	doc.SetSignerInfo(uploadResult.ProviderName, uploadResult.ProviderDocumentID)
	if err := doc.MarkAsPending(); err != nil {
		return fmt.Errorf("marking document as pending: %w", err)
	}
	if err := s.documentRepo.Update(ctx, doc); err != nil {
		return fmt.Errorf("updating document: %w", err)
	}

	s.updateRecipientsFromResult(ctx, recipients, uploadResult.Recipients)

	s.eventEmitter.EmitDocumentEvent(ctx, doc.ID, entity.EventDocumentSent, entity.ActorSystem, "",
		string(entity.DocumentStatusAwaitingInput), string(entity.DocumentStatusPending), nil)

	return nil
}

// buildDefaultSignatureFieldPositions generates fallback signature field positions when
// the renderer doesn't extract explicit positions. Each recipient gets a field on page 1.
func buildDefaultSignatureFieldPositions(recipients []*entity.DocumentRecipient) []port.SignatureFieldPosition {
	fields := make([]port.SignatureFieldPosition, 0, len(recipients))
	for i, r := range recipients {
		fields = append(fields, port.SignatureFieldPosition{
			RoleID:    r.TemplateVersionRoleID,
			Page:      1,
			PositionX: 10,
			PositionY: float64(70 + i*12),
			Width:     30,
			Height:    5,
		})
	}
	return fields
}

// updateRecipientsFromResult updates recipients with provider IDs and signing URLs.
func (s *PreSigningService) updateRecipientsFromResult(
	ctx context.Context,
	recipients []*entity.DocumentRecipient,
	results []port.RecipientResult,
) {
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

		if err := recipient.MarkAsSent(); err != nil {
			slog.WarnContext(ctx, "failed to mark recipient as sent",
				slog.String("recipient_id", recipient.ID),
				slog.String("error", err.Error()),
			)
		}

		if err := s.recipientRepo.Update(ctx, recipient); err != nil {
			slog.WarnContext(ctx, "failed to update recipient",
				slog.String("recipient_id", recipient.ID),
				slog.String("error", err.Error()),
			)
		}
	}
}

// Verify PreSigningService implements PreSigningUseCase.
var _ documentuc.PreSigningUseCase = (*PreSigningService)(nil)
