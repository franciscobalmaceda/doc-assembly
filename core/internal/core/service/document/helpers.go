package document

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rendis/doc-assembly/core/internal/core/entity"
	"github.com/rendis/doc-assembly/core/internal/core/entity/portabledoc"
	"github.com/rendis/doc-assembly/core/internal/core/port"
)

// loadFieldResponseMap loads saved field responses as a map for PDF rendering.
func loadFieldResponseMap(ctx context.Context, repo port.DocumentFieldResponseRepository, documentID string) map[string]json.RawMessage {
	responses, err := repo.FindByDocumentID(ctx, documentID)
	if err != nil || len(responses) == 0 {
		return nil
	}
	m := make(map[string]json.RawMessage, len(responses))
	for _, resp := range responses {
		m[resp.FieldID] = resp.Response
	}
	return m
}

// buildSignerRoleValues maps recipient data to portable doc role IDs for PDF rendering.
func buildSignerRoleValues(
	recipients []*entity.DocumentRecipient,
	dbSignerRoles []*entity.TemplateVersionSignerRole,
	portableDocRoles []portabledoc.SignerRole,
) map[string]port.SignerRoleValue {
	dbRoleToAnchor := make(map[string]string, len(dbSignerRoles))
	for _, role := range dbSignerRoles {
		dbRoleToAnchor[role.ID] = role.AnchorString
	}

	anchorToPortableID := make(map[string]string, len(portableDocRoles))
	for _, role := range portableDocRoles {
		anchor := portabledoc.GenerateAnchorString(role.Label)
		anchorToPortableID[anchor] = role.ID
	}

	values := make(map[string]port.SignerRoleValue, len(recipients))
	for _, r := range recipients {
		anchor := dbRoleToAnchor[r.TemplateVersionRoleID]
		portableDocRoleID := anchorToPortableID[anchor]
		if portableDocRoleID != "" {
			values[portableDocRoleID] = port.SignerRoleValue{
				Name:  r.Name,
				Email: r.Email,
			}
		}
	}
	return values
}

// buildAnchorToRoleIDMap builds a map from anchor string to DB role ID.
func buildAnchorToRoleIDMap(signerRoles []*entity.TemplateVersionSignerRole) map[string]string {
	m := make(map[string]string, len(signerRoles))
	for _, role := range signerRoles {
		if role.AnchorString != "" {
			m[role.AnchorString] = role.ID
		}
	}
	return m
}

// mapSignatureFieldPositions converts PDF renderer signature fields to signing provider positions.
// It resolves portable doc role IDs to DB role IDs via anchor string matching.
// Pass portableDocRoles=nil when fields already carry the anchor string directly.
func mapSignatureFieldPositions(
	fields []port.SignatureField,
	dbSignerRoles []*entity.TemplateVersionSignerRole,
	portableDocRoles []portabledoc.SignerRole,
) []port.SignatureFieldPosition {
	if len(fields) == 0 {
		return nil
	}

	anchorToDBRoleID := buildAnchorToRoleIDMap(dbSignerRoles)

	var portableIDToAnchor map[string]string
	if len(portableDocRoles) > 0 {
		portableIDToAnchor = make(map[string]string, len(portableDocRoles))
		for _, role := range portableDocRoles {
			portableIDToAnchor[role.ID] = portabledoc.GenerateAnchorString(role.Label)
		}
	}

	positions := make([]port.SignatureFieldPosition, 0, len(fields))
	for _, f := range fields {
		anchor := f.AnchorString
		if portableIDToAnchor != nil {
			if a := portableIDToAnchor[f.RoleID]; a != "" {
				anchor = a
			}
		}

		dbRoleID := anchorToDBRoleID[anchor]
		if dbRoleID == "" {
			continue
		}

		posX, posY := convertFieldToProviderPosition(f)
		positions = append(positions, port.SignatureFieldPosition{
			RoleID:    dbRoleID,
			Page:      f.Page,
			PositionX: posX,
			PositionY: posY,
			Width:     f.Width,
			Height:    f.Height,
		})
	}

	return positions
}

// documentTitle returns the document title, falling back to the document ID.
func documentTitle(doc *entity.Document) string {
	if doc.Title != nil {
		return *doc.Title
	}
	return doc.ID
}

func hasStoredPDFPath(doc *entity.Document) bool {
	return doc != nil && doc.PDFStoragePath != nil && *doc.PDFStoragePath != ""
}

func resolvePreProviderRecoveryTarget(
	ctx context.Context,
	storageAdapter port.StorageAdapter,
	storageEnabled bool,
	doc *entity.Document,
) (entity.DocumentStatus, bool) {
	if !storageEnabled {
		return entity.DocumentStatusAwaitingInput, false
	}

	if !hasStoredPDFPath(doc) {
		return entity.DocumentStatusAwaitingInput, false
	}

	if storageAdapter == nil {
		return entity.DocumentStatusAwaitingInput, true
	}

	if ctx == nil {
		ctx = context.Background()
	}

	exists, err := storageAdapter.Exists(ctx, &port.StorageRequest{Key: *doc.PDFStoragePath})
	if err != nil || !exists {
		return entity.DocumentStatusAwaitingInput, true
	}

	return entity.DocumentStatusPendingProvider, false
}

func signedDocumentFilename(doc *entity.Document) string {
	if doc != nil && doc.Title != nil {
		return fmt.Sprintf("%s-signed.pdf", *doc.Title)
	}
	if doc != nil {
		return fmt.Sprintf("document-%s-signed.pdf", doc.ID)
	}
	return "document-signed.pdf"
}
