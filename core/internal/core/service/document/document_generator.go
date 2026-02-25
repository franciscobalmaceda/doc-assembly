package document

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/rendis/doc-assembly/core/internal/core/entity"
	portable_doc "github.com/rendis/doc-assembly/core/internal/core/entity/portabledoc"
	"github.com/rendis/doc-assembly/core/internal/core/port"
	injectable_svc "github.com/rendis/doc-assembly/core/internal/core/service/injectable"
	injectable_uc "github.com/rendis/doc-assembly/core/internal/core/usecase/injectable"
	"github.com/rendis/doc-assembly/core/internal/core/validation"
)

// DocumentGenerationResult contains the result of document generation.
type DocumentGenerationResult struct {
	Document       *entity.Document
	Recipients     []*entity.DocumentRecipient
	Version        *entity.TemplateVersionWithDetails
	PortableDoc    *portable_doc.Document
	ResolvedValues map[string]any
}

// PreparedDocumentData contains generation output before persistence.
type PreparedDocumentData struct {
	WorkspaceID    string
	Version        *entity.TemplateVersionWithDetails
	PortableDoc    *portable_doc.Document
	ResolvedValues map[string]any
	Recipients     []*entity.DocumentRecipient
}

// DocumentGenerator is the centralized service for document generation.
// It orchestrates the entire flow: validation, mapping, injection, and creation.
// This service is reusable by CREATE, RENEW, AMEND operations.
type DocumentGenerator struct {
	templateRepo   port.TemplateRepository
	versionRepo    port.TemplateVersionRepository
	documentRepo   port.DocumentRepository
	recipientRepo  port.DocumentRecipientRepository
	injectableUC   injectable_uc.InjectableUseCase
	mapperRegistry port.MapperRegistry
	resolver       *injectable_svc.InjectableResolverService
}

// NewDocumentGenerator creates a new DocumentGenerator instance.
func NewDocumentGenerator(
	templateRepo port.TemplateRepository,
	versionRepo port.TemplateVersionRepository,
	documentRepo port.DocumentRepository,
	recipientRepo port.DocumentRecipientRepository,
	injectableUC injectable_uc.InjectableUseCase,
	mapperRegistry port.MapperRegistry,
	resolver *injectable_svc.InjectableResolverService,
) *DocumentGenerator {
	return &DocumentGenerator{
		templateRepo:   templateRepo,
		versionRepo:    versionRepo,
		documentRepo:   documentRepo,
		recipientRepo:  recipientRepo,
		injectableUC:   injectableUC,
		mapperRegistry: mapperRegistry,
		resolver:       resolver,
	}
}

// generationContext holds intermediate state during document generation.
type generationContext struct {
	workspaceID     string
	injectables     []*entity.InjectableDefinition
	version         *entity.TemplateVersionWithDetails
	portableDoc     *portable_doc.Document
	referencedCodes []string
	payload         any
	resolvedValues  map[string]any
	recipients      []*entity.DocumentRecipient
}

// GenerateDocument is the centralized method for document generation.
// It handles the complete flow from template lookup through document creation.
// Note: PDF rendering and signing provider upload are NOT handled here.
// The caller (InternalDocumentService) handles those steps after generation.
func (g *DocumentGenerator) GenerateDocument(
	ctx context.Context,
	mapCtx *port.MapperContext,
) (*DocumentGenerationResult, error) {
	prepared, err := g.PrepareDocument(ctx, mapCtx)
	if err != nil {
		return nil, err
	}

	doc, err := g.persistDocumentAndRecipients(
		ctx,
		prepared.WorkspaceID,
		prepared.Version.ID,
		mapCtx,
		prepared.ResolvedValues,
		prepared.Recipients,
	)
	if err != nil {
		return nil, err
	}

	return &DocumentGenerationResult{
		Document:       doc,
		Recipients:     prepared.Recipients,
		Version:        prepared.Version,
		PortableDoc:    prepared.PortableDoc,
		ResolvedValues: prepared.ResolvedValues,
	}, nil
}

// PrepareDocument resolves template data, injectables and recipients without persistence.
func (g *DocumentGenerator) PrepareDocument(
	ctx context.Context,
	mapCtx *port.MapperContext,
) (*PreparedDocumentData, error) {
	genCtx, err := g.prepareGenerationContext(ctx, mapCtx)
	if err != nil {
		return nil, err
	}

	return &PreparedDocumentData{
		WorkspaceID:    genCtx.workspaceID,
		Version:        genCtx.version,
		PortableDoc:    genCtx.portableDoc,
		ResolvedValues: genCtx.resolvedValues,
		Recipients:     genCtx.recipients,
	}, nil
}

// prepareGenerationContext fetches all required data and resolves injectables.
func (g *DocumentGenerator) prepareGenerationContext(
	ctx context.Context,
	mapCtx *port.MapperContext,
) (*generationContext, error) {
	genCtx, err := g.fetchTemplateData(ctx, mapCtx)
	if err != nil {
		return nil, err
	}

	if err := g.resolveAndBuildRecipients(ctx, mapCtx, genCtx); err != nil {
		return nil, err
	}

	return genCtx, nil
}

// fetchTemplateData fetches workspace, injectables, version, and parses the portable document.
func (g *DocumentGenerator) fetchTemplateData(ctx context.Context, mapCtx *port.MapperContext) (*generationContext, error) {
	genCtx := &generationContext{}
	var err error

	templateID := mapCtx.TemplateID
	if mapCtx.TemplateVersionID != "" {
		genCtx.version, err = g.findVersionByID(ctx, mapCtx.TemplateVersionID)
		if err != nil {
			return nil, err
		}
		if !genCtx.version.IsPublished() {
			return nil, entity.ErrInternalTemplateResolutionNotFound
		}
		templateID = genCtx.version.TemplateID
		mapCtx.TemplateID = templateID
	}

	genCtx.workspaceID, err = g.findWorkspaceID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	// Build InjectorContext for provider integration during injectable listing
	injCtx := entity.NewInjectorContext(
		mapCtx.ExternalID,
		mapCtx.TemplateID,
		mapCtx.TransactionalID,
		string(mapCtx.Operation),
		mapCtx.Environment,
		mapCtx.Headers,
		nil,
	)

	genCtx.injectables, err = g.fetchAvailableInjectables(ctx, genCtx.workspaceID, injCtx)
	if err != nil {
		return nil, err
	}

	if genCtx.version == nil {
		genCtx.version, err = g.findPublishedVersion(ctx, templateID)
		if err != nil {
			return nil, err
		}
	}

	genCtx.portableDoc, err = g.parseContentStructure(genCtx.version.ContentStructure)
	if err != nil {
		return nil, fmt.Errorf("parsing content structure: %w", err)
	}

	genCtx.referencedCodes = g.collectReferencedCodes(genCtx.version.Injectables, genCtx.portableDoc.SignerRoles)
	slog.DebugContext(ctx, "collected referenced codes", "codes", genCtx.referencedCodes)

	if err := g.validateRequiredInjectables(ctx, genCtx.referencedCodes, genCtx.injectables); err != nil {
		return nil, err
	}

	return genCtx, nil
}

func (g *DocumentGenerator) findVersionByID(
	ctx context.Context,
	versionID string,
) (*entity.TemplateVersionWithDetails, error) {
	version, err := g.versionRepo.FindByIDWithDetails(ctx, versionID)
	if err != nil {
		return nil, fmt.Errorf("finding template version by id: %w", err)
	}
	return version, nil
}

// resolveAndBuildRecipients executes the mapper, resolves injectables, and builds recipients.
func (g *DocumentGenerator) resolveAndBuildRecipients(
	ctx context.Context,
	mapCtx *port.MapperContext,
	genCtx *generationContext,
) error {
	var err error

	genCtx.payload, err = g.executeMapper(ctx, mapCtx)
	if err != nil {
		return err
	}

	genCtx.resolvedValues, err = g.resolveInjectables(ctx, mapCtx, genCtx.payload, genCtx.referencedCodes)
	if err != nil {
		return err
	}

	genCtx.recipients, err = g.buildRecipientsFromSignerRoles(ctx, genCtx.portableDoc.SignerRoles, genCtx.version.SignerRoles, genCtx.resolvedValues)
	if err != nil {
		slog.ErrorContext(ctx, "recipient validation failed", "error", err)
		return err
	}

	return nil
}

// findWorkspaceID retrieves the workspace ID from the template.
func (g *DocumentGenerator) findWorkspaceID(ctx context.Context, templateID string) (string, error) {
	if templateID == "" {
		return "", fmt.Errorf("template id is required to resolve workspace")
	}

	template, err := g.templateRepo.FindByID(ctx, templateID)
	if err != nil {
		return "", fmt.Errorf("finding template: %w", err)
	}

	slog.DebugContext(ctx, "found template", "workspaceID", template.WorkspaceID)
	return template.WorkspaceID, nil
}

// fetchAvailableInjectables retrieves all injectables available for the workspace.
func (g *DocumentGenerator) fetchAvailableInjectables(
	ctx context.Context,
	workspaceID string,
	_ *entity.InjectorContext,
) ([]*entity.InjectableDefinition, error) {
	result, err := g.injectableUC.ListInjectables(ctx, &injectable_uc.ListInjectablesRequest{
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("listing available injectables: %w", err)
	}

	slog.DebugContext(ctx, "found available injectables", "count", len(result.Injectables))
	return result.Injectables, nil
}

// findPublishedVersion retrieves the published version for the template.
func (g *DocumentGenerator) findPublishedVersion(
	ctx context.Context,
	templateID string,
) (*entity.TemplateVersionWithDetails, error) {
	version, err := g.versionRepo.FindPublishedByTemplateIDWithDetails(ctx, templateID)
	if err != nil {
		return nil, entity.ErrNoPublishedVersion
	}

	slog.DebugContext(ctx, "found published version",
		"versionID", version.ID,
		"versionNumber", version.VersionNumber,
	)
	return version, nil
}

// parseContentStructure parses the JSON ContentStructure into portable_doc.Document.
func (g *DocumentGenerator) parseContentStructure(content json.RawMessage) (*portable_doc.Document, error) {
	if content == nil {
		return nil, entity.ErrMissingRequiredContent
	}

	var doc portable_doc.Document
	if err := json.Unmarshal(content, &doc); err != nil {
		return nil, fmt.Errorf("unmarshaling content structure: %w", err)
	}

	return &doc, nil
}

// collectReferencedCodes collects all injectable codes referenced by the version.
// This includes codes from version injectables and codes referenced in SignerRoles.
func (g *DocumentGenerator) collectReferencedCodes(
	versionInjectables []*entity.VersionInjectableWithDefinition,
	signerRoles []portable_doc.SignerRole,
) []string {
	codeSet := make(map[string]bool)

	for _, vi := range versionInjectables {
		if vi.Definition != nil {
			codeSet[vi.Definition.Key] = true
		} else if vi.SystemInjectableKey != nil {
			codeSet[*vi.SystemInjectableKey] = true
		}
	}

	for _, sr := range signerRoles {
		if sr.Name.IsInjectable() && sr.Name.Value != "" {
			codeSet[sr.Name.Value] = true
		}
		if sr.Email.IsInjectable() && sr.Email.Value != "" {
			codeSet[sr.Email.Value] = true
		}
	}

	codes := make([]string, 0, len(codeSet))
	for code := range codeSet {
		codes = append(codes, code)
	}

	return codes
}

// validateRequiredInjectables validates that all required injectable codes are available.
func (g *DocumentGenerator) validateRequiredInjectables(
	ctx context.Context,
	requiredCodes []string,
	availableInjectables []*entity.InjectableDefinition,
) error {
	availableKeys := make(map[string]bool, len(availableInjectables))
	for _, inj := range availableInjectables {
		availableKeys[inj.Key] = true
	}

	var missingCodes []string
	for _, code := range requiredCodes {
		if !availableKeys[code] {
			missingCodes = append(missingCodes, code)
		}
	}

	if len(missingCodes) == 0 {
		return nil
	}

	slog.WarnContext(ctx, "missing required injectables",
		"missingCodes", missingCodes,
		"requiredCount", len(requiredCodes),
		"availableCount", len(availableInjectables),
	)
	return &entity.MissingInjectablesError{MissingCodes: missingCodes}
}

// executeMapper runs the registered mapper to transform the request.
func (g *DocumentGenerator) executeMapper(
	ctx context.Context,
	mapCtx *port.MapperContext,
) (any, error) {
	mapper, ok := g.mapperRegistry.Get()
	if !ok {
		return nil, entity.ErrNoMapperRegistered
	}

	payload, err := mapper.Map(ctx, mapCtx)
	if err != nil {
		return nil, fmt.Errorf("mapper failed: %w", err)
	}

	slog.DebugContext(ctx, "mapper executed successfully")
	return payload, nil
}

// resolveInjectables executes injectors and returns resolved values.
func (g *DocumentGenerator) resolveInjectables(
	ctx context.Context,
	mapCtx *port.MapperContext,
	payload any,
	referencedCodes []string,
) (map[string]any, error) {
	var injCtx *entity.InjectorContext
	if mapCtx.TenantCode != "" || mapCtx.WorkspaceCode != "" {
		injCtx = entity.NewInjectorContextWithCodes(
			mapCtx.ExternalID,
			mapCtx.TemplateID,
			mapCtx.TransactionalID,
			string(mapCtx.Operation),
			mapCtx.TenantCode,
			mapCtx.WorkspaceCode,
			mapCtx.Environment,
			mapCtx.Headers,
			payload,
		)
	} else {
		injCtx = entity.NewInjectorContext(
			mapCtx.ExternalID,
			mapCtx.TemplateID,
			mapCtx.TransactionalID,
			string(mapCtx.Operation),
			mapCtx.Environment,
			mapCtx.Headers,
			payload,
		)
	}

	resolveResult, err := g.resolver.Resolve(ctx, injCtx, referencedCodes)
	if err != nil {
		return nil, fmt.Errorf("resolving injectors: %w", err)
	}

	slog.DebugContext(ctx, "injectors resolved", "resolvedCount", len(resolveResult.Values))

	resolvedValues := make(map[string]any, len(resolveResult.Values))
	for code, val := range resolveResult.Values {
		resolvedValues[code] = val.AsAny()
	}

	return resolvedValues, nil
}

// buildRecipientsFromSignerRoles builds and validates DocumentRecipient entities from portable_doc SignerRoles.
func (g *DocumentGenerator) buildRecipientsFromSignerRoles(
	ctx context.Context,
	portableSignerRoles []portable_doc.SignerRole,
	dbSignerRoles []*entity.TemplateVersionSignerRole,
	resolvedValues map[string]any,
) ([]*entity.DocumentRecipient, error) {
	roleByAnchor := make(map[string]*entity.TemplateVersionSignerRole, len(dbSignerRoles))
	for _, r := range dbSignerRoles {
		roleByAnchor[r.AnchorString] = r
	}

	var validationErrors []string
	recipients := make([]*entity.DocumentRecipient, 0, len(portableSignerRoles))

	for _, sr := range portableSignerRoles {
		recipient, err := g.buildAndValidateRecipient(sr, roleByAnchor, resolvedValues)
		if err != nil {
			validationErrors = append(validationErrors, err.Error())
			continue
		}
		recipients = append(recipients, recipient)
	}

	if len(validationErrors) > 0 {
		return nil, &entity.RecipientValidationError{Errors: validationErrors}
	}

	return recipients, nil
}

// buildAndValidateRecipient creates and validates a single DocumentRecipient from a SignerRole.
func (g *DocumentGenerator) buildAndValidateRecipient(
	sr portable_doc.SignerRole,
	roleByAnchor map[string]*entity.TemplateVersionSignerRole,
	resolvedValues map[string]any,
) (*entity.DocumentRecipient, error) {
	name := validation.NormalizeName(g.resolveFieldValue(sr.Name, resolvedValues))
	email := strings.TrimSpace(g.resolveFieldValue(sr.Email, resolvedValues))
	anchor := portable_doc.GenerateAnchorString(sr.Label)
	dbRole, found := roleByAnchor[anchor]

	// Validate with descriptive error messages
	if !found {
		return nil, fmt.Errorf("role '%s': no matching signature anchor found", sr.Label)
	}
	if name == "" {
		return nil, fmt.Errorf("role '%s': name is empty after resolution", sr.Label)
	}
	if email == "" {
		return nil, fmt.Errorf("role '%s': email is empty after resolution", sr.Label)
	}
	if !validation.IsValidEmail(email) {
		return nil, fmt.Errorf("role '%s': invalid email format '%s'", sr.Label, email)
	}

	return &entity.DocumentRecipient{
		ID:                    uuid.NewString(),
		TemplateVersionRoleID: dbRole.ID,
		Name:                  name,
		Email:                 email,
		Status:                entity.RecipientStatusPending,
	}, nil
}

// resolveFieldValue resolves a FieldValue to its actual string value.
func (g *DocumentGenerator) resolveFieldValue(
	field portable_doc.FieldValue,
	resolvedValues map[string]any,
) string {
	if field.IsText() {
		return field.Value
	}

	if !field.IsInjectable() {
		return ""
	}

	val, ok := resolvedValues[field.Value]
	if !ok {
		return ""
	}

	if strVal, ok := val.(string); ok {
		return strVal
	}

	return fmt.Sprintf("%v", val)
}

// createDocument creates and persists the document entity.
func (g *DocumentGenerator) createDocument(
	ctx context.Context,
	workspaceID string,
	versionID string,
	mapCtx *port.MapperContext,
	resolvedValues map[string]any,
) (*entity.Document, error) {
	doc := entity.NewDocument(workspaceID, versionID)
	doc.DocumentTypeID = mapCtx.DocumentTypeID
	doc.SetOperationType(mapCtx.Operation)
	doc.SetExternalReference(mapCtx.ExternalID)
	doc.SetTransactionalID(mapCtx.TransactionalID)

	if err := doc.SetInjectedValuesSnapshot(resolvedValues); err != nil {
		return nil, fmt.Errorf("setting injected values snapshot: %w", err)
	}

	docID, err := g.documentRepo.Create(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("saving document: %w", err)
	}

	doc.ID = docID
	slog.DebugContext(ctx, "document saved", "documentID", docID)

	return doc, nil
}

// persistDocumentAndRecipients creates and saves the document with its recipients.
func (g *DocumentGenerator) persistDocumentAndRecipients(
	ctx context.Context,
	workspaceID, versionID string,
	mapCtx *port.MapperContext,
	resolvedValues map[string]any,
	recipients []*entity.DocumentRecipient,
) (*entity.Document, error) {
	doc, err := g.createDocument(ctx, workspaceID, versionID, mapCtx, resolvedValues)
	if err != nil {
		return nil, err
	}

	if err := g.saveRecipients(ctx, doc.ID, recipients); err != nil {
		return nil, err
	}

	return doc, nil
}

// saveRecipients persists the document recipients.
func (g *DocumentGenerator) saveRecipients(
	ctx context.Context,
	documentID string,
	recipients []*entity.DocumentRecipient,
) error {
	if len(recipients) == 0 {
		return nil
	}

	for _, r := range recipients {
		r.DocumentID = documentID
	}

	if err := g.recipientRepo.CreateBatch(ctx, recipients); err != nil {
		return fmt.Errorf("saving recipients: %w", err)
	}

	slog.DebugContext(ctx, "recipients saved", "count", len(recipients))
	return nil
}
