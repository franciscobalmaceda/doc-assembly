// Package riverqueue implements background job processing for document
// completion events using River, a PostgreSQL-native job queue.
// See docs/backend/worker-queue-guide.md for architecture and flow diagrams.
package riverqueue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/rendis/doc-assembly/core/internal/core/port"
	"github.com/rendis/doc-assembly/core/internal/infra/config"
)

// RiverService manages the River client lifecycle and exposes the notifier.
type RiverService struct {
	client   *river.Client[pgx.Tx]
	notifier *Notifier
}

// New creates a RiverService: runs migrations, registers the worker, and
// builds the River client. When cfg.Enabled is false the client operates in
// insert-only mode (no queue processing).
func New(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg config.WorkerConfig,
	handler port.DocumentCompletedHandler,
	docUpdater documentUpdater,
) (*RiverService, error) {
	driver := riverpgxv5.New(pool)

	// Run River schema migrations programmatically.
	migrator, err := rivermigrate.New(driver, nil)
	if err != nil {
		return nil, fmt.Errorf("creating river migrator: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return nil, fmt.Errorf("running river migrations: %w", err)
	}

	// Register workers.
	workers := river.NewWorkers()
	river.AddWorker(workers, &DocumentCompletedWorker{
		handler: handler,
		pool:    pool,
	})

	// Build River config. When disabled, omit Queues so the client is insert-only.
	riverCfg := &river.Config{
		Workers: workers,
	}
	if cfg.Enabled {
		riverCfg.Queues = map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: cfg.MaxWorkersOrDefault()},
		}
	}

	client, err := river.NewClient(driver, riverCfg)
	if err != nil {
		return nil, fmt.Errorf("creating river client: %w", err)
	}

	notifier := &Notifier{
		pool:       pool,
		client:     client,
		docUpdater: docUpdater,
	}

	slog.InfoContext(ctx, "river queue initialized",
		slog.Bool("workers_enabled", cfg.Enabled),
		slog.Int("max_workers", cfg.MaxWorkersOrDefault()),
	)

	return &RiverService{
		client:   client,
		notifier: notifier,
	}, nil
}

// Notifier returns the DocumentCompletionNotifier for use by the document service.
func (r *RiverService) Notifier() port.DocumentCompletionNotifier {
	return r.notifier
}

// Start begins processing jobs. Only meaningful when workers are enabled.
func (r *RiverService) Start(ctx context.Context) error {
	return r.client.Start(ctx)
}

// Stop gracefully shuts down the River client.
func (r *RiverService) Stop(ctx context.Context) error {
	return r.client.Stop(ctx)
}
