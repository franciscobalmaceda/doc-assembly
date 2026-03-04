package documentaccesstokenrepo

const (
	queryCreate = `
		INSERT INTO execution.document_access_tokens (
			document_id, recipient_id, token, token_type, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	queryFindByAccessValue = ` -- nolint:gosec // not a credential, SQL query param
		SELECT id, document_id, recipient_id, token, token_type, expires_at, used_at, created_at
		FROM execution.document_access_tokens
		WHERE token = $1
	`

	queryFindActiveByRecipientAndType = `
		SELECT id, document_id, recipient_id, token, token_type, expires_at, used_at, created_at
		FROM execution.document_access_tokens
		WHERE recipient_id = $1
		  AND token_type = $2
		  AND used_at IS NULL
		  AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`

	queryFindActiveByDocumentRecipientAndType = `
		SELECT id, document_id, recipient_id, token, token_type, expires_at, used_at, created_at
		FROM execution.document_access_tokens
		WHERE document_id = $1
		  AND recipient_id = $2
		  AND token_type = $3
		  AND used_at IS NULL
		  AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`

	queryMarkAsUsed = `
		UPDATE execution.document_access_tokens
		SET used_at = NOW()
		WHERE id = $1
	`

	queryInvalidateByDocumentID = `
		UPDATE execution.document_access_tokens
		SET used_at = NOW()
		WHERE document_id = $1
		  AND used_at IS NULL
	`

	queryCountRecentByDocAndRecipient = `
		SELECT COUNT(*)
		FROM execution.document_access_tokens
		WHERE document_id = $1
		  AND recipient_id = $2
		  AND created_at >= $3
	`
)
