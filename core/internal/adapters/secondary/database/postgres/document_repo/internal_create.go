package documentrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/rendis/doc-assembly/core/internal/core/entity"
	"github.com/rendis/doc-assembly/core/internal/core/port"
)

const (
	uniqueViolationCode              = "23505"
	constraintWorkspaceTransactional = "uq_documents_workspace_transactional_id"
	constraintActiveLogicalKey       = "uq_documents_active_logical_key"
)

const (
	queryAdvisoryLockInternalCreate = `
		SELECT pg_advisory_xact_lock(hashtextextended($1, 0))
	`

	queryFindByWorkspaceTransactionalForUpdate = `
		SELECT id
		FROM execution.documents
		WHERE workspace_id = $1 AND transactional_id = $2
		ORDER BY created_at ASC, id ASC
		LIMIT 1
		FOR UPDATE
	`

	queryFindActiveByLogicalKeyForUpdate = `
		SELECT id
		FROM execution.documents
		WHERE workspace_id = $1
		  AND document_type_id = $2
		  AND client_external_reference_id = $3
		  AND is_active = TRUE
		ORDER BY created_at DESC, id DESC
		LIMIT 1
		FOR UPDATE
	`

	queryMarkSuperseded = `
		UPDATE execution.documents
		SET is_active = FALSE,
		    superseded_at = NOW(),
		    supersede_reason = $2,
		    updated_at = NOW()
		WHERE id = $1
	`

	querySetSupersededByDocument = `
		UPDATE execution.documents
		SET superseded_by_document_id = $2,
		    updated_at = NOW()
		WHERE id = $1
	`

	queryCreateInternalDocument = `
		INSERT INTO execution.documents (
			workspace_id, template_version_id, document_type_id, title, client_external_reference_id,
			transactional_id, operation_type, related_document_id,
			signer_document_id, signer_provider, status, injected_values_snapshot,
			pdf_storage_path, completed_pdf_url, is_active, superseded_at,
			superseded_by_document_id, supersede_reason, expires_at, metadata, created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15, $16,
			$17, $18, $19, $20, $21
		)
		RETURNING id
	`

	queryCreateInternalRecipient = `
		INSERT INTO execution.document_recipients (
			document_id, template_version_role_id, name, email,
			signer_recipient_id, signing_url, status, signed_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	queryFindByWorkspaceTransactional = `
		SELECT id
		FROM execution.documents
		WHERE workspace_id = $1 AND transactional_id = $2
		ORDER BY created_at ASC, id ASC
		LIMIT 1
	`

	queryFindActiveByLogicalKey = `
		SELECT id
		FROM execution.documents
		WHERE workspace_id = $1
		  AND document_type_id = $2
		  AND client_external_reference_id = $3
		  AND is_active = TRUE
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`
)

// InternalCreateOrReplay executes transactional create/replay/supersede logic for internal create.
//
//nolint:nestif // Conflict recovery branches are explicit and easier to audit.
func (r *Repository) InternalCreateOrReplay(ctx context.Context, req *port.InternalCreateRequest) (*port.InternalCreateResult, error) {
	if req == nil || req.Document == nil {
		return nil, fmt.Errorf("invalid internal create request")
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("starting internal create transaction: %w", err)
	}

	result, err := r.internalCreateOrReplayTx(ctx, tx, req)
	if err != nil {
		_ = tx.Rollback(ctx)

		if isConstraintViolation(err, constraintWorkspaceTransactional) {
			docID, findErr := r.findByWorkspaceTransactional(ctx, req.WorkspaceID, req.TransactionalID)
			if findErr != nil {
				return nil, fmt.Errorf("recovering transactional replay after conflict: %w", findErr)
			}
			return &port.InternalCreateResult{DocumentID: docID, IdempotentReplay: true}, nil
		}

		if isConstraintViolation(err, constraintActiveLogicalKey) {
			docID, findErr := r.findActiveByLogicalKey(ctx, req.WorkspaceID, req.DocumentTypeID, req.ExternalID)
			if findErr != nil {
				return nil, fmt.Errorf("recovering active replay after conflict: %w", findErr)
			}
			return &port.InternalCreateResult{DocumentID: docID, IdempotentReplay: true}, nil
		}

		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing internal create transaction: %w", err)
	}

	return result, nil
}

//nolint:funlen,gocognit,gocyclo,nestif // Transaction flow keeps conflict/replay branches explicit.
func (r *Repository) internalCreateOrReplayTx(ctx context.Context, tx pgx.Tx, req *port.InternalCreateRequest) (*port.InternalCreateResult, error) {
	lockKey := fmt.Sprintf("%s|%s|%s", req.WorkspaceID, req.ExternalID, req.DocumentTypeID)
	if _, err := tx.Exec(ctx, queryAdvisoryLockInternalCreate, lockKey); err != nil {
		return nil, fmt.Errorf("acquiring advisory lock: %w", err)
	}

	if replayDocID, ok, err := r.findTransactionalReplayTx(ctx, tx, req.WorkspaceID, req.TransactionalID); err != nil {
		return nil, err
	} else if ok {
		return &port.InternalCreateResult{
			DocumentID:       replayDocID,
			IdempotentReplay: true,
		}, nil
	}

	activeDocID, hasActive, err := r.findActiveByLogicalKeyTx(ctx, tx, req.WorkspaceID, req.DocumentTypeID, req.ExternalID)
	if err != nil {
		return nil, err
	}

	if hasActive && !req.ForceCreate {
		return &port.InternalCreateResult{
			DocumentID:       activeDocID,
			IdempotentReplay: true,
		}, nil
	}

	if hasActive {
		if _, err := tx.Exec(ctx, queryMarkSuperseded, activeDocID, req.SupersedeReason); err != nil {
			return nil, fmt.Errorf("marking active document superseded: %w", err)
		}
	}

	doc := req.Document
	doc.DocumentTypeID = req.DocumentTypeID
	doc.IsActive = true
	if doc.ClientExternalReferenceID == nil {
		doc.SetExternalReference(req.ExternalID)
	}
	if doc.TransactionalID == nil {
		doc.SetTransactionalID(req.TransactionalID)
	}
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now().UTC()
	}

	var createdDocID string
	if err := tx.QueryRow(ctx, queryCreateInternalDocument,
		doc.WorkspaceID,
		doc.TemplateVersionID,
		doc.DocumentTypeID,
		doc.Title,
		doc.ClientExternalReferenceID,
		doc.TransactionalID,
		doc.OperationType,
		doc.RelatedDocumentID,
		doc.SignerDocumentID,
		doc.SignerProvider,
		doc.Status,
		doc.InjectedValuesSnapshot,
		doc.PDFStoragePath,
		doc.CompletedPDFURL,
		doc.IsActive,
		doc.SupersededAt,
		doc.SupersededByDocumentID,
		doc.SupersedeReason,
		doc.ExpiresAt,
		doc.Metadata,
		doc.CreatedAt,
	).Scan(&createdDocID); err != nil {
		return nil, fmt.Errorf("creating internal document: %w", err)
	}

	for _, recipient := range req.Recipients {
		if recipient == nil {
			continue
		}
		recipient.DocumentID = createdDocID
		if recipient.CreatedAt.IsZero() {
			recipient.CreatedAt = time.Now().UTC()
		}
		if !recipient.Status.IsValid() {
			recipient.Status = entity.RecipientStatusPending
		}

		if _, err := tx.Exec(ctx, queryCreateInternalRecipient,
			recipient.DocumentID,
			recipient.TemplateVersionRoleID,
			recipient.Name,
			recipient.Email,
			recipient.SignerRecipientID,
			recipient.SigningURL,
			recipient.Status,
			recipient.SignedAt,
			recipient.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("creating internal recipient: %w", err)
		}
	}

	if hasActive {
		if _, err := tx.Exec(ctx, querySetSupersededByDocument, activeDocID, createdDocID); err != nil {
			return nil, fmt.Errorf("linking superseded document: %w", err)
		}
	}

	result := &port.InternalCreateResult{DocumentID: createdDocID}
	if hasActive {
		result.SupersededPreviousDocumentID = &activeDocID
	}

	return result, nil
}

func (r *Repository) findTransactionalReplayTx(ctx context.Context, tx pgx.Tx, workspaceID, transactionalID string) (string, bool, error) {
	var docID string
	err := tx.QueryRow(ctx, queryFindByWorkspaceTransactionalForUpdate, workspaceID, transactionalID).Scan(&docID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("querying transactional replay: %w", err)
	}
	return docID, true, nil
}

func (r *Repository) findActiveByLogicalKeyTx(ctx context.Context, tx pgx.Tx, workspaceID, documentTypeID, externalID string) (string, bool, error) {
	var docID string
	err := tx.QueryRow(ctx, queryFindActiveByLogicalKeyForUpdate, workspaceID, documentTypeID, externalID).Scan(&docID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("querying active document by logical key: %w", err)
	}
	return docID, true, nil
}

func (r *Repository) findByWorkspaceTransactional(ctx context.Context, workspaceID, transactionalID string) (string, error) {
	var docID string
	err := r.pool.QueryRow(ctx, queryFindByWorkspaceTransactional, workspaceID, transactionalID).Scan(&docID)
	if err != nil {
		return "", err
	}
	return docID, nil
}

func (r *Repository) findActiveByLogicalKey(ctx context.Context, workspaceID, documentTypeID, externalID string) (string, error) {
	var docID string
	err := r.pool.QueryRow(ctx, queryFindActiveByLogicalKey, workspaceID, documentTypeID, externalID).Scan(&docID)
	if err != nil {
		return "", err
	}
	return docID, nil
}

func isConstraintViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == uniqueViolationCode && pgErr.ConstraintName == constraint
}
