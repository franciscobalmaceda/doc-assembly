package dto

import (
	"encoding/json"

	"github.com/rendis/doc-assembly/core/internal/core/entity"
)

// InternalCreateDocumentRequest is the new contract for internal create.
type InternalCreateDocumentRequest struct {
	ForceCreate     *bool             `json:"forceCreate,omitempty"`
	SupersedeReason *string           `json:"supersedeReason,omitempty"`
	Payload         json.RawMessage   `json:"payload"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// InternalCreateDocumentResponse is the response for document creation via internal API.
type InternalCreateDocumentResponse struct {
	ID                           string  `json:"id"`
	WorkspaceID                  string  `json:"workspaceId"`
	TemplateVersionID            string  `json:"templateVersionId"`
	ExternalID                   string  `json:"externalId"`
	TransactionalID              string  `json:"transactionalId"`
	Status                       string  `json:"status"`
	IdempotentReplay             bool    `json:"idempotentReplay"`
	SupersededPreviousDocumentID *string `json:"supersededPreviousDocumentId,omitempty"`
}

// InternalDocumentRecipientResponse represents a recipient in the internal API response.
type InternalDocumentRecipientResponse struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Email      string  `json:"email"`
	SigningURL *string `json:"signingUrl,omitempty"`
}

// InternalCreateDocumentWithRecipientsResponse includes recipients in the response.
type InternalCreateDocumentWithRecipientsResponse struct {
	InternalCreateDocumentResponse
	Recipients []InternalDocumentRecipientResponse `json:"recipients,omitempty"`
}

// InternalErrorResponse is the error response for internal API.
type InternalErrorResponse struct {
	Error   string   `json:"error"`
	Code    string   `json:"code,omitempty"`
	Details []string `json:"details,omitempty"`
}

// MissingInjectablesErrorResponse is the response for missing injectables error.
type MissingInjectablesErrorResponse struct {
	Error        string   `json:"error"`
	Code         string   `json:"code"`
	MissingCodes []string `json:"missingCodes"`
}

// NewMissingInjectablesErrorResponse creates a response for missing injectables error.
func NewMissingInjectablesErrorResponse(err *entity.MissingInjectablesError) MissingInjectablesErrorResponse {
	return MissingInjectablesErrorResponse{
		Error:        err.Error(),
		Code:         "MISSING_INJECTABLES",
		MissingCodes: err.MissingCodes,
	}
}
