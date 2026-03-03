package document

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/rendis/doc-assembly/core/internal/core/entity"
	"github.com/rendis/doc-assembly/core/internal/core/port"
	documentuc "github.com/rendis/doc-assembly/core/internal/core/usecase/document"
)

// InternalDocumentService implements usecase.InternalDocumentUseCase.
type InternalDocumentService struct {
	generator       *DocumentGenerator
	documentRepo    port.DocumentRepository
	tenantRepo      port.TenantRepository
	workspaceRepo   port.WorkspaceRepository
	docTypeRepo     port.DocumentTypeRepository
	templateRepo    port.TemplateRepository
	versionRepo     port.TemplateVersionRepository
	customResolver  port.TemplateResolver
	defaultResolver port.TemplateResolver
	searchAdapter   port.TemplateVersionSearchAdapter
}

// NewInternalDocumentService creates a new InternalDocumentService.
func NewInternalDocumentService(
	generator *DocumentGenerator,
	documentRepo port.DocumentRepository,
	tenantRepo port.TenantRepository,
	workspaceRepo port.WorkspaceRepository,
	docTypeRepo port.DocumentTypeRepository,
	templateRepo port.TemplateRepository,
	versionRepo port.TemplateVersionRepository,
	customResolver port.TemplateResolver,
) documentuc.InternalDocumentUseCase {
	return &InternalDocumentService{
		generator:       generator,
		documentRepo:    documentRepo,
		tenantRepo:      tenantRepo,
		workspaceRepo:   workspaceRepo,
		docTypeRepo:     docTypeRepo,
		templateRepo:    templateRepo,
		versionRepo:     versionRepo,
		customResolver:  customResolver,
		defaultResolver: NewDefaultTemplateResolver(),
		searchAdapter: NewTemplateVersionSearchAdapter(
			tenantRepo,
			workspaceRepo,
			docTypeRepo,
			templateRepo,
			versionRepo,
		),
	}
}

// CreateDocument creates or replays a document using the extension system.
//
//nolint:funlen // Service orchestration is intentionally linear for traceability.
func (s *InternalDocumentService) CreateDocument(
	ctx context.Context,
	cmd documentuc.InternalCreateCommand,
) (*documentuc.InternalCreateResult, error) {
	slog.InfoContext(ctx, "creating document via internal API",
		"tenantCode", cmd.TenantCode,
		"workspaceCode", cmd.WorkspaceCode,
		"documentType", cmd.DocumentType,
		"externalID", cmd.ExternalID,
	)

	resolved, err := s.resolveTemplateContext(ctx, cmd)
	if err != nil {
		return nil, err
	}

	mapCtx := &port.MapperContext{
		ExternalID:        cmd.ExternalID,
		TemplateID:        resolved.template.ID,
		TemplateVersionID: resolved.version.ID,
		DocumentTypeID:    resolved.documentType.ID,
		TenantCode:        cmd.TenantCode,
		WorkspaceCode:     cmd.WorkspaceCode,
		DocumentTypeCode:  cmd.DocumentType,
		TransactionalID:   cmd.TransactionalID,
		Operation:         entity.OperationCreate,
		Environment:       cmd.Environment,
		Headers:           cmd.Headers,
		RawBody:           cmd.PayloadRaw,
	}

	prepared, err := s.generator.PrepareDocument(ctx, mapCtx)
	if err != nil {
		return nil, fmt.Errorf("preparing internal document: %w", err)
	}

	doc, err := s.buildInternalDocument(cmd, resolved, prepared)
	if err != nil {
		return nil, err
	}

	txResult, err := s.documentRepo.InternalCreateOrReplay(ctx, &port.InternalCreateRequest{
		WorkspaceID:     resolved.workspace.ID,
		DocumentTypeID:  resolved.documentType.ID,
		ExternalID:      cmd.ExternalID,
		TransactionalID: cmd.TransactionalID,
		ForceCreate:     cmd.ForceCreate,
		SupersedeReason: cmd.SupersedeReason,
		Document:        doc,
		Recipients:      prepared.Recipients,
	})
	if err != nil {
		return nil, fmt.Errorf("persisting internal document: %w", err)
	}

	stored, err := s.documentRepo.FindByIDWithRecipients(ctx, txResult.DocumentID)
	if err != nil {
		return nil, fmt.Errorf("loading internal document after persistence: %w", err)
	}

	return &documentuc.InternalCreateResult{
		Document:                     stored,
		IdempotentReplay:             txResult.IdempotentReplay,
		SupersededPreviousDocumentID: txResult.SupersededPreviousDocumentID,
	}, nil
}

type internalResolvedContext struct {
	tenant       *entity.Tenant
	workspace    *entity.Workspace
	documentType *entity.DocumentType
	template     *entity.Template
	version      *entity.TemplateVersionWithDetails
}

//nolint:funlen // Resolution has explicit fallback and validation stages.
func (s *InternalDocumentService) resolveTemplateContext(
	ctx context.Context,
	cmd documentuc.InternalCreateCommand,
) (*internalResolvedContext, error) {
	tenantCode := strings.ToUpper(strings.TrimSpace(cmd.TenantCode))
	workspaceCode := strings.ToUpper(strings.TrimSpace(cmd.WorkspaceCode))
	documentTypeCode := strings.ToUpper(strings.TrimSpace(cmd.DocumentType))

	tenant, err := s.tenantRepo.FindByCode(ctx, tenantCode)
	if err != nil {
		return nil, fmt.Errorf("resolving tenant by code: %w", err)
	}
	slog.InfoContext(ctx, "resolved tenant", slog.String("tenantID", tenant.ID), slog.String("tenantCode", tenantCode))

	workspace, err := s.workspaceRepo.FindByCode(ctx, tenant.ID, workspaceCode)
	if err != nil {
		if !errors.Is(err, entity.ErrWorkspaceNotFound) || s.customResolver == nil {
			return nil, fmt.Errorf("resolving workspace by code: %w", err)
		}
		slog.InfoContext(ctx, "workspace not found, deferring to template resolver",
			slog.String("workspaceCode", workspaceCode))
	} else {
		slog.InfoContext(ctx, "resolved workspace", slog.String("workspaceID", workspace.ID), slog.String("workspaceCode", workspaceCode))
	}

	docType, err := s.docTypeRepo.FindByCodeWithGlobalFallback(ctx, tenant.ID, documentTypeCode)
	if err != nil {
		return nil, fmt.Errorf("resolving document type by code: %w", err)
	}
	slog.InfoContext(ctx, "resolved document type", slog.String("docTypeID", docType.ID), slog.String("docTypeCode", documentTypeCode))

	process := strings.ToUpper(strings.TrimSpace(cmd.Process))
	if process == "" {
		process = entity.DefaultProcess
	}
	processType := cmd.ProcessType
	if processType == "" {
		processType = string(entity.DefaultProcessType)
	}

	resolverReq := &port.TemplateResolverRequest{
		TenantCode:      tenantCode,
		WorkspaceCode:   workspaceCode,
		DocumentType:    documentTypeCode,
		Process:         process,
		ProcessType:     processType,
		ExternalID:      cmd.ExternalID,
		TransactionalID: cmd.TransactionalID,
		ForceCreate:     cmd.ForceCreate,
		SupersedeReason: cmd.SupersedeReason,
		Headers:         cmd.Headers,
		RawBody:         cmd.PayloadRaw,
		Environment:     cmd.Environment,
	}

	if workspace != nil {
		if err := s.applySandboxWorkspaceCode(ctx, resolverReq, workspace.ID, cmd.Environment); err != nil {
			return nil, err
		}
	}

	versionID, err := s.resolveTemplateVersionID(ctx, resolverReq)
	if err != nil {
		return nil, err
	}

	version, err := s.versionRepo.FindByIDWithDetails(ctx, *versionID)
	if err != nil {
		return nil, fmt.Errorf("loading resolved template version: %w", err)
	}
	if !version.IsPublished() {
		return nil, entity.ErrInternalTemplateResolutionNotFound
	}

	template, err := s.templateRepo.FindByID(ctx, version.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("loading resolved template: %w", err)
	}
	if template.DocumentTypeID == nil || *template.DocumentTypeID != docType.ID {
		return nil, entity.ErrInternalTemplateResolutionNotFound
	}

	if workspace == nil {
		workspace, err = s.workspaceRepo.FindByID(ctx, template.WorkspaceID)
		if err != nil {
			return nil, fmt.Errorf("resolving workspace from resolved template: %w", err)
		}
	}

	return &internalResolvedContext{
		tenant:       tenant,
		workspace:    workspace,
		documentType: docType,
		template:     template,
		version:      version,
	}, nil
}

// applySandboxWorkspaceCode populates SandboxWorkspaceCode when environment is dev.
func (s *InternalDocumentService) applySandboxWorkspaceCode(
	ctx context.Context,
	req *port.TemplateResolverRequest,
	workspaceID string,
	env entity.Environment,
) error {
	if env != entity.EnvironmentDev {
		return nil
	}
	sandbox, err := s.workspaceRepo.FindSandboxByParentID(ctx, workspaceID)
	if err == nil {
		req.SandboxWorkspaceCode = sandbox.Code
		return nil
	}
	if errors.Is(err, entity.ErrSandboxNotFound) {
		return nil
	}
	return fmt.Errorf("resolving sandbox workspace: %w", err)
}

// resolveTemplateVersionID runs custom then default resolver to find a template version.
func (s *InternalDocumentService) resolveTemplateVersionID(
	ctx context.Context,
	req *port.TemplateResolverRequest,
) (*string, error) {
	var versionID *string
	if s.customResolver != nil {
		vID, err := s.customResolver.Resolve(ctx, req, s.searchAdapter)
		if err != nil {
			slog.ErrorContext(ctx, "custom template resolver error", slog.Any("error", err))
			return nil, fmt.Errorf("custom template resolver failed: %w", err)
		}
		if vID != nil {
			slog.InfoContext(ctx, "custom resolver hit", slog.String("versionID", *vID))
		} else {
			slog.InfoContext(ctx, "custom resolver miss, falling back to default")
		}
		versionID = vID
	}

	if versionID == nil {
		vID, err := s.defaultResolver.Resolve(ctx, req, s.searchAdapter)
		if err != nil {
			slog.ErrorContext(ctx, "default template resolver error", slog.Any("error", err))
			return nil, err
		}
		if vID != nil {
			slog.InfoContext(ctx, "default resolver hit", slog.String("versionID", *vID))
		} else {
			slog.WarnContext(ctx, "default resolver miss, no template version found")
		}
		versionID = vID
	}
	if versionID == nil || *versionID == "" {
		return nil, entity.ErrInternalTemplateResolutionNotFound
	}
	return versionID, nil
}

func (s *InternalDocumentService) buildInternalDocument(
	cmd documentuc.InternalCreateCommand,
	resolved *internalResolvedContext,
	prepared *PreparedDocumentData,
) (*entity.Document, error) {
	doc := entity.NewDocument(resolved.workspace.ID, resolved.version.ID)
	doc.DocumentTypeID = resolved.documentType.ID
	doc.SetOperationType(entity.OperationCreate)
	doc.SetExternalReference(cmd.ExternalID)
	doc.SetTransactionalID(cmd.TransactionalID)

	if len(cmd.Metadata) > 0 {
		if err := doc.SetMetadata(cmd.Metadata); err != nil {
			return nil, fmt.Errorf("setting metadata: %w", err)
		}
	}

	if err := doc.SetInjectedValuesSnapshot(prepared.ResolvedValues); err != nil {
		return nil, fmt.Errorf("setting injected values snapshot: %w", err)
	}

	if len(prepared.Recipients) > 0 {
		if err := doc.MarkAsAwaitingInput(); err != nil {
			return nil, fmt.Errorf("marking internal document as awaiting input: %w", err)
		}
	}

	if err := doc.Validate(); err != nil {
		return nil, fmt.Errorf("validating internal document: %w", err)
	}

	return doc, nil
}
