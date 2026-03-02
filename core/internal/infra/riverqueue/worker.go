package riverqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/rendis/doc-assembly/core/internal/core/entity"
	"github.com/rendis/doc-assembly/core/internal/core/port"
)

// DocumentCompletedWorker processes document completion jobs.
type DocumentCompletedWorker struct {
	river.WorkerDefaults[DocumentCompletedArgs]
	handler port.DocumentCompletedHandler
	pool    *pgxpool.Pool
}

// Work executes the document completion handler with defensive panic recovery.
func (w *DocumentCompletedWorker) Work(ctx context.Context, job *river.Job[DocumentCompletedArgs]) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "document completion handler panicked",
				slog.String("document_id", job.Args.DocumentID),
				slog.String("panic", fmt.Sprintf("%v", r)),
			)
			retErr = fmt.Errorf("handler panicked: %v", r)
		}
	}()

	event, err := buildCompletedEvent(ctx, w.pool, job.Args.DocumentID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to build completion event",
			slog.String("document_id", job.Args.DocumentID),
			slog.String("error", err.Error()),
		)
		return err
	}

	return w.handler(ctx, event)
}

// Timeout returns the maximum duration for processing a single job.
func (w *DocumentCompletedWorker) Timeout(_ *river.Job[DocumentCompletedArgs]) time.Duration {
	return 30 * time.Second
}

// buildCompletedEvent loads fresh document data from the database and constructs the event.
func buildCompletedEvent(ctx context.Context, pool *pgxpool.Pool, documentID string) (port.DocumentCompletedEvent, error) {
	var event port.DocumentCompletedEvent

	// Fetch document with workspace and tenant codes.
	var isSandbox bool
	var rawMetadata json.RawMessage
	err := pool.QueryRow(ctx, `
		SELECT d.id, d.status, d.client_external_reference_id, d.title,
		       d.created_at, d.updated_at, d.expires_at, d.metadata,
		       w.code AS workspace_code, w.is_sandbox, t.code AS tenant_code
		FROM execution.documents d
		JOIN tenancy.workspaces w ON w.id = d.workspace_id
		JOIN tenancy.tenants t ON t.id = w.tenant_id
		WHERE d.id = $1
	`, documentID).Scan(
		&event.DocumentID,
		&event.Status,
		&event.ExternalID,
		&event.Title,
		&event.CreatedAt,
		&event.UpdatedAt,
		&event.ExpiresAt,
		&rawMetadata,
		&event.WorkspaceCode,
		&isSandbox,
		&event.TenantCode,
	)
	if err != nil {
		return event, fmt.Errorf("querying document %s: %w", documentID, err)
	}
	event.Environment = entity.EnvironmentFromSandbox(isSandbox)

	if len(rawMetadata) > 0 {
		if err := json.Unmarshal(rawMetadata, &event.Metadata); err != nil {
			return event, fmt.Errorf("unmarshalling metadata for document %s: %w", documentID, err)
		}
	}

	// Fetch recipients with role information.
	rows, err := pool.Query(ctx, `
		SELECT sr.role_name, sr.signer_order, dr.name, dr.email, dr.status, dr.signed_at
		FROM execution.document_recipients dr
		JOIN content.template_version_signer_roles sr ON sr.id = dr.template_version_role_id
		WHERE dr.document_id = $1
		ORDER BY sr.signer_order ASC
	`, documentID)
	if err != nil {
		return event, fmt.Errorf("querying recipients for document %s: %w", documentID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var r port.CompletedRecipient
		if err := rows.Scan(&r.RoleName, &r.SignerOrder, &r.Name, &r.Email, &r.Status, &r.SignedAt); err != nil {
			return event, fmt.Errorf("scanning recipient: %w", err)
		}
		event.Recipients = append(event.Recipients, r)
	}
	if err := rows.Err(); err != nil {
		return event, fmt.Errorf("iterating recipients: %w", err)
	}

	return event, nil
}

// Verify DocumentCompletedWorker satisfies the Worker interface at compile time.
var _ river.Worker[DocumentCompletedArgs] = (*DocumentCompletedWorker)(nil)
