package document

import (
	"context"
	"fmt"
	"strings"

	"github.com/rendis/doc-assembly/core/internal/core/entity"
	"github.com/rendis/doc-assembly/core/internal/core/port"
)

// TemplateVersionSearchAdapter provides read-only template version search for custom resolvers.
type TemplateVersionSearchAdapter struct {
	tenantRepo    port.TenantRepository
	workspaceRepo port.WorkspaceRepository
	docTypeRepo   port.DocumentTypeRepository
	templateRepo  port.TemplateRepository
	versionRepo   port.TemplateVersionRepository
}

// NewTemplateVersionSearchAdapter builds a new read-only search adapter.
func NewTemplateVersionSearchAdapter(
	tenantRepo port.TenantRepository,
	workspaceRepo port.WorkspaceRepository,
	docTypeRepo port.DocumentTypeRepository,
	templateRepo port.TemplateRepository,
	versionRepo port.TemplateVersionRepository,
) port.TemplateVersionSearchAdapter {
	return &TemplateVersionSearchAdapter{
		tenantRepo:    tenantRepo,
		workspaceRepo: workspaceRepo,
		docTypeRepo:   docTypeRepo,
		templateRepo:  templateRepo,
		versionRepo:   versionRepo,
	}
}

// SearchTemplateVersions returns deterministic candidates by tenant/workspace/document type.
//
//nolint:funlen,gocognit,gocyclo // Search logic encodes business fallback/filtering rules explicitly.
func (a *TemplateVersionSearchAdapter) SearchTemplateVersions(ctx context.Context, params port.TemplateVersionSearchParams) ([]port.TemplateVersionSearchItem, error) {
	if params.TenantCode == "" {
		return nil, fmt.Errorf("tenantCode is required")
	}
	if len(params.WorkspaceCodes) == 0 {
		return nil, fmt.Errorf("workspaceCodes is required")
	}
	if params.DocumentType == "" {
		return nil, fmt.Errorf("documentType is required")
	}

	tenant, err := a.tenantRepo.FindByCode(ctx, strings.ToUpper(strings.TrimSpace(params.TenantCode)))
	if err != nil {
		if err == entity.ErrTenantNotFound {
			return []port.TemplateVersionSearchItem{}, nil
		}
		return nil, fmt.Errorf("finding tenant by code: %w", err)
	}

	docType, err := a.docTypeRepo.FindByCodeWithGlobalFallback(ctx, tenant.ID, strings.ToUpper(strings.TrimSpace(params.DocumentType)))
	if err != nil {
		if err == entity.ErrDocumentTypeNotFound {
			return []port.TemplateVersionSearchItem{}, nil
		}
		return nil, fmt.Errorf("finding document type by code: %w", err)
	}

	wantPublished := true
	if params.Published != nil {
		wantPublished = *params.Published
	}

	results := make([]port.TemplateVersionSearchItem, 0, len(params.WorkspaceCodes))
	for _, workspaceCode := range params.WorkspaceCodes {
		workspaceCode = strings.ToUpper(strings.TrimSpace(workspaceCode))
		workspace, err := a.workspaceRepo.FindByCode(ctx, tenant.ID, workspaceCode)
		if err != nil {
			if err == entity.ErrWorkspaceNotFound {
				continue
			}
			return nil, fmt.Errorf("finding workspace by code: %w", err)
		}

		process := strings.ToUpper(strings.TrimSpace(params.Process))
		if process == "" {
			process = entity.DefaultProcess
		}
		tmpl, err := a.templateRepo.FindByDocumentType(ctx, workspace.ID, docType.ID, process)
		if err != nil {
			return nil, fmt.Errorf("finding template by document type: %w", err)
		}
		if tmpl == nil && !strings.EqualFold(process, entity.DefaultProcess) {
			tmpl, err = a.templateRepo.FindByDocumentType(ctx, workspace.ID, docType.ID, entity.DefaultProcess)
			if err != nil {
				return nil, fmt.Errorf("finding template by document type (default fallback): %w", err)
			}
		}
		if tmpl == nil {
			continue
		}

		tags, err := a.loadTemplateTags(ctx, tmpl.ID)
		if err != nil {
			return nil, err
		}
		if !tagsMatchAny(params.Tags, tags) {
			continue
		}

		if wantPublished {
			version, err := a.versionRepo.FindPublishedByTemplateID(ctx, tmpl.ID)
			if err != nil {
				if err == entity.ErrVersionNotFound || err == entity.ErrNoPublishedVersion {
					continue
				}
				continue
			}

			results = append(results, port.TemplateVersionSearchItem{
				Published:     true,
				TenantCode:    tenant.Code,
				WorkspaceCode: workspace.Code,
				VersionID:     version.ID,
				Tags:          tags,
			})
			continue
		}

		versions, err := a.versionRepo.FindByTemplateID(ctx, tmpl.ID)
		if err != nil {
			return nil, fmt.Errorf("finding versions by template: %w", err)
		}

		for _, v := range versions {
			if v.IsPublished() {
				continue
			}
			results = append(results, port.TemplateVersionSearchItem{
				Published:     false,
				TenantCode:    tenant.Code,
				WorkspaceCode: workspace.Code,
				VersionID:     v.ID,
				Tags:          tags,
			})
		}
	}

	return results, nil
}

func (a *TemplateVersionSearchAdapter) loadTemplateTags(ctx context.Context, templateID string) ([]string, error) {
	tmpl, err := a.templateRepo.FindByIDWithDetails(ctx, templateID)
	if err != nil {
		if err == entity.ErrTemplateNotFound {
			return []string{}, nil
		}
		return nil, fmt.Errorf("loading template details for tags: %w", err)
	}

	tags := make([]string, 0, len(tmpl.Tags))
	for _, tag := range tmpl.Tags {
		if tag == nil {
			continue
		}
		tags = append(tags, strings.ToUpper(strings.TrimSpace(tag.Name)))
	}
	return tags, nil
}

func tagsMatchAny(required, candidate []string) bool {
	if len(required) == 0 {
		return true
	}

	set := make(map[string]struct{}, len(candidate))
	for _, t := range candidate {
		t = strings.ToUpper(strings.TrimSpace(t))
		if t != "" {
			set[t] = struct{}{}
		}
	}

	for _, t := range required {
		t = strings.ToUpper(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if _, ok := set[t]; ok {
			return true
		}
	}

	return false
}
