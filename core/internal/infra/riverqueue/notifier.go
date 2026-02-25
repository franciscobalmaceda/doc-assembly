package riverqueue

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/rendis/doc-assembly/core/internal/core/entity"
)

// documentUpdater is a local interface for updating documents within a transaction.
// Defined here (consumer side) following Go's interface-at-consumer idiom.
type documentUpdater interface {
	UpdateTx(ctx context.Context, tx pgx.Tx, doc *entity.Document) error
}

// Notifier implements port.DocumentCompletionNotifier using River.
// It persists the document update and enqueues the completion job
// atomically within the same PostgreSQL transaction.
type Notifier struct {
	pool       *pgxpool.Pool
	client     *river.Client[pgx.Tx]
	docUpdater documentUpdater
}

// PersistAndNotify updates the document and enqueues a completion job in a single transaction.
func (n *Notifier) PersistAndNotify(ctx context.Context, doc *entity.Document) error {
	tx, err := n.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	if err := n.docUpdater.UpdateTx(ctx, tx, doc); err != nil {
		return fmt.Errorf("update document in tx: %w", err)
	}

	_, err = n.client.InsertTx(ctx, tx, DocumentCompletedArgs{DocumentID: doc.ID}, nil)
	if err != nil {
		return fmt.Errorf("enqueue completion job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
