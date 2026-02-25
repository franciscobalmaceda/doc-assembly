package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/rendis/doc-assembly/core/internal/core/entity"
	"github.com/rendis/doc-assembly/core/internal/core/port"
)

const providerName = "mock"

// mockDocument represents a document stored in the mock adapter.
type mockDocument struct {
	ID         string
	Title      string
	Status     string // PENDING, COMPLETED, VOIDED
	Recipients []string
	PDFData    []byte
}

// mockRecipient represents a recipient stored in the mock adapter.
type mockRecipient struct {
	ID         string
	DocumentID string
	RoleID     string
	Email      string
	Name       string
	Status     string // SENT, SIGNED, DECLINED
	SignedAt   *time.Time
}

// Adapter implements port.SigningProvider and port.WebhookHandler for testing.
type Adapter struct {
	mu         sync.RWMutex
	documents  map[string]*mockDocument
	recipients map[string]*mockRecipient
}

// New creates a new mock signing adapter.
func New() *Adapter {
	return &Adapter{
		documents:  make(map[string]*mockDocument),
		recipients: make(map[string]*mockRecipient),
	}
}

// ProviderName returns the name of this signing provider.
func (a *Adapter) ProviderName() string {
	return providerName
}

// UploadDocument stores a mock document and generates IDs for recipients.
func (a *Adapter) UploadDocument(_ context.Context, req *port.UploadDocumentRequest) (*port.UploadDocumentResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	docID := uuid.New().String()

	recipientIDs := make([]string, 0, len(req.Recipients))
	recipientResults := make([]port.RecipientResult, 0, len(req.Recipients))

	for _, r := range req.Recipients {
		recipientID := uuid.New().String()
		recipientIDs = append(recipientIDs, recipientID)

		a.recipients[recipientID] = &mockRecipient{
			ID:         recipientID,
			DocumentID: docID,
			RoleID:     r.RoleID,
			Email:      r.Email,
			Name:       r.Name,
			Status:     "SENT",
		}

		recipientResults = append(recipientResults, port.RecipientResult{
			RoleID:              r.RoleID,
			ProviderRecipientID: recipientID,
			SigningURL:          fmt.Sprintf("http://mock-signing/sign/%s", recipientID),
			Status:              entity.RecipientStatusSent,
		})
	}

	a.documents[docID] = &mockDocument{
		ID:         docID,
		Title:      req.Title,
		Status:     "PENDING",
		Recipients: recipientIDs,
		PDFData:    req.PDF,
	}

	return &port.UploadDocumentResult{
		ProviderDocumentID: docID,
		ProviderName:       providerName,
		Status:             entity.DocumentStatusPending,
		Recipients:         recipientResults,
	}, nil
}

// GetSigningURL returns a mock signing URL for the recipient.
func (a *Adapter) GetSigningURL(_ context.Context, req *port.GetSigningURLRequest) (*port.GetSigningURLResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if _, ok := a.recipients[req.ProviderRecipientID]; !ok {
		return nil, fmt.Errorf("mock: recipient %s not found", req.ProviderRecipientID)
	}

	return &port.GetSigningURLResult{
		SigningURL: fmt.Sprintf("http://mock-signing/sign/%s", req.ProviderRecipientID),
	}, nil
}

// GetEmbeddedSigningURL returns a mock embeddable signing URL.
func (a *Adapter) GetEmbeddedSigningURL(_ context.Context, req *port.GetEmbeddedSigningURLRequest) (*port.GetEmbeddedSigningURLResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if _, ok := a.recipients[req.ProviderRecipientID]; !ok {
		return nil, fmt.Errorf("mock: recipient %s not found", req.ProviderRecipientID)
	}

	return &port.GetEmbeddedSigningURLResult{
		EmbeddedURL:    fmt.Sprintf("http://mock-signing/embed/%s", req.ProviderRecipientID),
		FrameSrcDomain: "http://mock-signing",
	}, nil
}

// GetDocumentStatus returns the current status of a mock document.
func (a *Adapter) GetDocumentStatus(_ context.Context, req *port.GetDocumentStatusRequest) (*port.DocumentStatusResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	doc, ok := a.documents[req.ProviderDocumentID]
	if !ok {
		return nil, fmt.Errorf("mock: document %s not found", req.ProviderDocumentID)
	}

	recipientResults := make([]port.RecipientStatusResult, 0, len(doc.Recipients))
	allSigned := true

	for _, rid := range doc.Recipients {
		r := a.recipients[rid]
		status := mapRecipientStatus(r.Status)
		if status != entity.RecipientStatusSigned {
			allSigned = false
		}

		recipientResults = append(recipientResults, port.RecipientStatusResult{
			ProviderRecipientID: r.ID,
			Status:              status,
			SignedAt:            r.SignedAt,
			ProviderStatus:      r.Status,
		})
	}

	docStatus := mapDocumentStatus(doc.Status)
	if allSigned && len(doc.Recipients) > 0 {
		docStatus = entity.DocumentStatusCompleted
	}

	return &port.DocumentStatusResult{
		Status:         docStatus,
		Recipients:     recipientResults,
		ProviderStatus: doc.Status,
	}, nil
}

// CancelDocument sets the mock document status to VOIDED.
func (a *Adapter) CancelDocument(_ context.Context, req *port.CancelDocumentRequest) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	doc, ok := a.documents[req.ProviderDocumentID]
	if !ok {
		return fmt.Errorf("mock: document %s not found", req.ProviderDocumentID)
	}

	doc.Status = "VOIDED"
	return nil
}

// DownloadSignedPDF returns mock PDF bytes.
func (a *Adapter) DownloadSignedPDF(_ context.Context, req *port.DownloadSignedPDFRequest) ([]byte, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	doc, ok := a.documents[req.ProviderDocumentID]
	if !ok {
		return nil, fmt.Errorf("mock: document %s not found", req.ProviderDocumentID)
	}

	if doc.Status != "COMPLETED" {
		return nil, fmt.Errorf("mock: document %s not completed (status: %s)", req.ProviderDocumentID, doc.Status)
	}

	// Return stored PDF or a minimal valid PDF
	if len(doc.PDFData) > 0 {
		return doc.PDFData, nil
	}

	return minimalPDF(), nil
}

// ParseWebhook parses a webhook body into a WebhookEvent (no HMAC validation).
func (a *Adapter) ParseWebhook(_ context.Context, req *port.ParseWebhookRequest) (*port.WebhookEvent, error) {
	var event port.WebhookEvent
	if err := json.Unmarshal(req.Body, &event); err != nil {
		return nil, fmt.Errorf("mock: parsing webhook: %w", err)
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	event.RawPayload = req.Body

	return &event, nil
}

// --- Test Helper Methods ---

// SimulateSign sets a recipient's status to SIGNED with the current timestamp.
func (a *Adapter) SimulateSign(recipientID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	r, ok := a.recipients[recipientID]
	if !ok {
		return
	}

	now := time.Now()
	r.Status = "SIGNED"
	r.SignedAt = &now
}

// SimulateComplete sets all recipients to SIGNED and the document to COMPLETED.
func (a *Adapter) SimulateComplete(documentID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	doc, ok := a.documents[documentID]
	if !ok {
		return
	}

	now := time.Now()
	for _, rid := range doc.Recipients {
		r := a.recipients[rid]
		r.Status = "SIGNED"
		r.SignedAt = &now
	}
	doc.Status = "COMPLETED"
}

// GetMockDocument returns the internal mock document for assertions.
func (a *Adapter) GetMockDocument(documentID string) *mockDocument {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.documents[documentID]
}

// GetMockRecipient returns the internal mock recipient for assertions.
func (a *Adapter) GetMockRecipient(recipientID string) *mockRecipient {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.recipients[recipientID]
}

// Reset clears all stored documents and recipients.
func (a *Adapter) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.documents = make(map[string]*mockDocument)
	a.recipients = make(map[string]*mockRecipient)
}

// DocumentCount returns the number of documents stored in the mock adapter.
func (a *Adapter) DocumentCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.documents)
}

// --- Internal helpers ---

func mapRecipientStatus(status string) entity.RecipientStatus {
	switch status {
	case "SENT":
		return entity.RecipientStatusSent
	case "SIGNED":
		return entity.RecipientStatusSigned
	case "DECLINED":
		return entity.RecipientStatusDeclined
	case "DELIVERED":
		return entity.RecipientStatusDelivered
	default:
		return entity.RecipientStatusPending
	}
}

func mapDocumentStatus(status string) entity.DocumentStatus {
	switch status {
	case "PENDING":
		return entity.DocumentStatusPending
	case "COMPLETED":
		return entity.DocumentStatusCompleted
	case "VOIDED":
		return entity.DocumentStatusVoided
	default:
		return entity.DocumentStatusPending
	}
}

func minimalPDF() []byte {
	return []byte(`%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>
endobj
xref
0 4
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000115 00000 n
trailer
<< /Size 4 /Root 1 0 R >>
startxref
190
%%EOF`)
}

// Compile-time interface checks.
var (
	_ port.SigningProvider = (*Adapter)(nil)
	_ port.WebhookHandler  = (*Adapter)(nil)
)
