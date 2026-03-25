package galleryassetrepo

const (
	querySave = `
		INSERT INTO content.gallery_assets
			(tenant_id, workspace_id, key, filename, content_type, size, sha256, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	queryFindBySHA256 = `
		SELECT id, tenant_id, workspace_id, key, filename, content_type, size, sha256, created_by, created_at
		FROM content.gallery_assets
		WHERE workspace_id = $1 AND sha256 = $2
		LIMIT 1`

	queryFindByKey = `
		SELECT id, tenant_id, workspace_id, key, filename, content_type, size, sha256, created_by, created_at
		FROM content.gallery_assets
		WHERE workspace_id = $1 AND key = $2
		LIMIT 1`

	queryList = `
		SELECT id, tenant_id, workspace_id, key, filename, content_type, size, sha256, created_by, created_at
		FROM content.gallery_assets
		WHERE workspace_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	queryListCount = `
		SELECT COUNT(*) FROM content.gallery_assets
		WHERE workspace_id = $1`

	querySearch = `
		SELECT id, tenant_id, workspace_id, key, filename, content_type, size, sha256, created_by, created_at
		FROM content.gallery_assets
		WHERE workspace_id = $1
		  AND filename ILIKE $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	querySearchCount = `
		SELECT COUNT(*) FROM content.gallery_assets
		WHERE workspace_id = $1
		  AND filename ILIKE $2`

	queryDelete = `
		DELETE FROM content.gallery_assets
		WHERE workspace_id = $1 AND key = $2`
)
