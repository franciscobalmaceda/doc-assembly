package documentrepo

const (
	queryCreate = `
		INSERT INTO execution.documents (
			workspace_id, template_version_id, document_type_id, title, client_external_reference_id,
			transactional_id, operation_type, related_document_id,
			signer_document_id, signer_provider, status, injected_values_snapshot,
			pdf_storage_path, completed_pdf_url, is_active, superseded_at,
			superseded_by_document_id, supersede_reason, expires_at, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
		RETURNING id
	`

	queryFindByID = `
		SELECT id, workspace_id, template_version_id, document_type_id, title, client_external_reference_id,
			   transactional_id, operation_type, related_document_id,
			   signer_document_id, signer_provider, status, injected_values_snapshot,
			   pdf_storage_path, completed_pdf_url, is_active, superseded_at,
			   superseded_by_document_id, supersede_reason, expires_at, metadata,
			   retry_count, last_retry_at, next_retry_at,
			   created_at, updated_at
		FROM execution.documents
		WHERE id = $1
	`

	queryFindBySignerDocumentID = `
		SELECT id, workspace_id, template_version_id, document_type_id, title, client_external_reference_id,
			   transactional_id, operation_type, related_document_id,
			   signer_document_id, signer_provider, status, injected_values_snapshot,
			   pdf_storage_path, completed_pdf_url, is_active, superseded_at,
			   superseded_by_document_id, supersede_reason, expires_at, metadata,
			   retry_count, last_retry_at, next_retry_at,
			   created_at, updated_at
		FROM execution.documents
		WHERE signer_document_id = $1
	`

	queryFindByClientExternalRef = `
		SELECT id, workspace_id, template_version_id, document_type_id, title, client_external_reference_id,
			   transactional_id, operation_type, related_document_id,
			   signer_document_id, signer_provider, status, injected_values_snapshot,
			   pdf_storage_path, completed_pdf_url, is_active, superseded_at,
			   superseded_by_document_id, supersede_reason, expires_at, metadata,
			   retry_count, last_retry_at, next_retry_at,
			   created_at, updated_at
		FROM execution.documents
		WHERE workspace_id = $1 AND client_external_reference_id = $2
		ORDER BY created_at DESC
	`

	queryFindByTemplateVersion = `
		SELECT id, workspace_id, template_version_id, title, client_external_reference_id,
			   signer_provider, status, created_at, updated_at
		FROM execution.documents
		WHERE template_version_id = $1
		ORDER BY created_at DESC
	`

	queryFindPendingForPolling = `
		SELECT id, workspace_id, template_version_id, document_type_id, title, client_external_reference_id,
			   transactional_id, operation_type, related_document_id,
			   signer_document_id, signer_provider, status, injected_values_snapshot,
			   pdf_storage_path, completed_pdf_url, is_active, superseded_at,
			   superseded_by_document_id, supersede_reason, expires_at, metadata,
			   retry_count, last_retry_at, next_retry_at,
			   created_at, updated_at
		FROM execution.documents
		WHERE status IN ('PENDING', 'IN_PROGRESS')
		  AND signer_document_id IS NOT NULL
		ORDER BY updated_at ASC NULLS FIRST, created_at ASC
		LIMIT $1
	`

	queryFindErrorsForRetry = `
		SELECT id, workspace_id, template_version_id, document_type_id, title, client_external_reference_id,
			   transactional_id, operation_type, related_document_id,
			   signer_document_id, signer_provider, status, injected_values_snapshot,
			   pdf_storage_path, completed_pdf_url, is_active, superseded_at,
			   superseded_by_document_id, supersede_reason, expires_at, metadata,
			   retry_count, last_retry_at, next_retry_at,
			   created_at, updated_at
		FROM execution.documents
		WHERE status = 'ERROR'
		  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		  AND retry_count < $1
		ORDER BY next_retry_at ASC NULLS FIRST, created_at ASC
		LIMIT $2
	`

	queryFindPendingProviderForUpload = `
		SELECT id, workspace_id, template_version_id, document_type_id, title, client_external_reference_id,
			   transactional_id, operation_type, related_document_id,
			   signer_document_id, signer_provider, status, injected_values_snapshot,
			   pdf_storage_path, completed_pdf_url, is_active, superseded_at,
			   superseded_by_document_id, supersede_reason, expires_at, metadata,
			   retry_count, last_retry_at, next_retry_at,
			   created_at, updated_at
		FROM execution.documents
		WHERE status = 'PENDING_PROVIDER'
		  AND updated_at < NOW() - INTERVAL '60 seconds'
		ORDER BY created_at ASC
		LIMIT $1
	`

	queryFindExpired = `
		SELECT id, workspace_id, template_version_id, document_type_id, title, client_external_reference_id,
			   transactional_id, operation_type, related_document_id,
			   signer_document_id, signer_provider, status, injected_values_snapshot,
			   pdf_storage_path, completed_pdf_url, is_active, superseded_at,
			   superseded_by_document_id, supersede_reason, expires_at, metadata,
			   retry_count, last_retry_at, next_retry_at,
			   created_at, updated_at
		FROM execution.documents
		WHERE status IN ('PENDING', 'IN_PROGRESS')
		  AND expires_at IS NOT NULL
		  AND expires_at <= NOW()
		ORDER BY expires_at ASC
		LIMIT $1
	`

	queryFindByWorkspaceBase = `
		SELECT id, workspace_id, template_version_id, title, client_external_reference_id,
			   signer_provider, status, created_at, updated_at
		FROM execution.documents
		WHERE workspace_id = $1
	`

	queryUpdate = `
		UPDATE execution.documents
		SET document_type_id = $2, title = $3, client_external_reference_id = $4,
			transactional_id = $5, operation_type = $6, related_document_id = $7,
			signer_document_id = $8, signer_provider = $9, status = $10,
			injected_values_snapshot = $11, pdf_storage_path = $12,
			completed_pdf_url = $13, is_active = $14, superseded_at = $15,
			superseded_by_document_id = $16, supersede_reason = $17, expires_at = $18,
			metadata = $19,
			retry_count = $20, last_retry_at = $21, next_retry_at = $22,
			updated_at = $23
		WHERE id = $1
	`

	queryUpdateStatus = `
		UPDATE execution.documents
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`

	// queryClaimForSigning atomically transitions AWAITING_INPUT → PENDING_PROVIDER.
	// Only one concurrent caller succeeds; losers get zero rows.
	queryClaimForSigning = `
		UPDATE execution.documents
		SET status = 'PENDING_PROVIDER', updated_at = NOW()
		WHERE id = $1 AND status = 'AWAITING_INPUT'
		RETURNING id, workspace_id, template_version_id, document_type_id, title,
				  client_external_reference_id, transactional_id, operation_type,
				  related_document_id, signer_document_id, signer_provider, status,
				  injected_values_snapshot, pdf_storage_path, completed_pdf_url,
				  is_active, superseded_at, superseded_by_document_id, supersede_reason,
				  expires_at, metadata, retry_count, last_retry_at, next_retry_at,
				  created_at, updated_at
	`

	queryDelete = `
		DELETE FROM execution.documents
		WHERE id = $1
	`

	queryCountByWorkspace = `
		SELECT COUNT(*)
		FROM execution.documents
		WHERE workspace_id = $1
	`

	queryCountByStatus = `
		SELECT COUNT(*)
		FROM execution.documents
		WHERE workspace_id = $1 AND status = $2
	`
)
