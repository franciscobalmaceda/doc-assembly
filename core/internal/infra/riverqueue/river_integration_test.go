//go:build integration

package riverqueue_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rendis/doc-assembly/core/internal/adapters/secondary/database/postgres/document_access_token_repo"
	"github.com/rendis/doc-assembly/core/internal/adapters/secondary/database/postgres/document_event_repo"
	"github.com/rendis/doc-assembly/core/internal/adapters/secondary/database/postgres/document_field_response_repo"
	"github.com/rendis/doc-assembly/core/internal/adapters/secondary/database/postgres/document_recipient_repo"
	"github.com/rendis/doc-assembly/core/internal/adapters/secondary/database/postgres/document_repo"
	"github.com/rendis/doc-assembly/core/internal/adapters/secondary/database/postgres/template_repo"
	"github.com/rendis/doc-assembly/core/internal/adapters/secondary/database/postgres/template_version_repo"
	"github.com/rendis/doc-assembly/core/internal/adapters/secondary/database/postgres/template_version_signer_role_repo"
	noopnotification "github.com/rendis/doc-assembly/core/internal/adapters/secondary/notification/noop"
	mocksigning "github.com/rendis/doc-assembly/core/internal/adapters/secondary/signing/mock"
	localstorage "github.com/rendis/doc-assembly/core/internal/adapters/secondary/storage/local"
	"github.com/rendis/doc-assembly/core/internal/core/entity"
	"github.com/rendis/doc-assembly/core/internal/core/port"
	documentsvc "github.com/rendis/doc-assembly/core/internal/core/service/document"
	documentuc "github.com/rendis/doc-assembly/core/internal/core/usecase/document"
	"github.com/rendis/doc-assembly/core/internal/infra/config"
	"github.com/rendis/doc-assembly/core/internal/infra/riverqueue"
	"github.com/rendis/doc-assembly/core/internal/testing/testhelper"
)

// ---------------------------------------------------------------------------
// Helper types
// ---------------------------------------------------------------------------

type testInfra struct {
	pool        *pgxpool.Pool
	tenantID    string
	tenantCode  string
	workspaceID string
	versionID   string
	roleIDs     [2]string
}

type docData struct {
	docID           string
	providerDocID   string
	recipientIDs    [2]string
	providerRcptIDs [2]string
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

func ptr[T any](v T) *T { return &v }

// newTestInfra creates tenant → workspace → template (published) → 2 signer
// roles → document type. Cleanup is automatic.
func newTestInfra(t *testing.T, pool *pgxpool.Pool, code string) *testInfra {
	t.Helper()

	tenantID := testhelper.CreateTestTenant(t, pool, "River "+code, code)
	workspaceID := testhelper.CreateTestWorkspace(t, pool, &tenantID, "WS", entity.WorkspaceTypeClient)
	templateID := testhelper.CreateTestTemplate(t, pool, workspaceID, "Template", nil)
	versionID := testhelper.CreateTestTemplateVersion(t, pool, templateID, 1, "v1", entity.VersionStatusDraft)
	testhelper.PublishTestVersion(t, pool, versionID)
	role1 := testhelper.CreateTestSignerRole(t, pool, versionID, "Signer", "{{SIGN_1}}", 1)
	role2 := testhelper.CreateTestSignerRole(t, pool, versionID, "Witness", "{{SIGN_2}}", 2)
	docTypeID := testhelper.CreateTestDocumentType(t, pool, tenantID, code+"_DOC", "Document")
	testhelper.SetTemplateDocumentType(t, pool, templateID, docTypeID)

	t.Cleanup(func() { testhelper.CleanupTenant(t, pool, tenantID) })

	return &testInfra{
		pool:        pool,
		tenantID:    tenantID,
		tenantCode:  code,
		workspaceID: workspaceID,
		versionID:   versionID,
		roleIDs:     [2]string{role1, role2},
	}
}

// newSandboxTestInfra creates the same as newTestInfra but with a sandbox workspace.
// The document will be created in the sandbox workspace so the worker sees is_sandbox=true.
func newSandboxTestInfra(t *testing.T, pool *pgxpool.Pool, code string) *testInfra {
	t.Helper()

	tenantID := testhelper.CreateTestTenant(t, pool, "River "+code, code)

	// Create parent (production) workspace
	parentWorkspaceID := testhelper.CreateTestWorkspace(t, pool, &tenantID, "WS", entity.WorkspaceTypeClient)

	// Create sandbox workspace
	sandboxWorkspaceID := testhelper.CreateTestSandboxWorkspace(t, pool, parentWorkspaceID)

	// Create template + version + roles in sandbox workspace
	templateID := testhelper.CreateTestTemplate(t, pool, sandboxWorkspaceID, "Template", nil)
	versionID := testhelper.CreateTestTemplateVersion(t, pool, templateID, 1, "v1", entity.VersionStatusDraft)
	testhelper.PublishTestVersion(t, pool, versionID)
	role1 := testhelper.CreateTestSignerRole(t, pool, versionID, "Signer", "{{SIGN_1}}", 1)
	role2 := testhelper.CreateTestSignerRole(t, pool, versionID, "Witness", "{{SIGN_2}}", 2)
	docTypeID := testhelper.CreateTestDocumentType(t, pool, tenantID, code+"_DOC", "Document")
	testhelper.SetTemplateDocumentType(t, pool, templateID, docTypeID)

	t.Cleanup(func() {
		testhelper.CleanupWorkspace(t, pool, sandboxWorkspaceID)
		testhelper.CleanupTenant(t, pool, tenantID)
	})

	return &testInfra{
		pool:        pool,
		tenantID:    tenantID,
		tenantCode:  code,
		workspaceID: sandboxWorkspaceID,
		versionID:   versionID,
		roleIDs:     [2]string{role1, role2},
	}
}

// newDocService creates a fully-wired DocumentService backed by real repos
// and mock adapters (signing, PDF, storage, notification).
func newDocService(t *testing.T, pool *pgxpool.Pool) *documentsvc.DocumentService {
	t.Helper()

	docRepo := documentrepo.New(pool)
	recipientRepo := documentrecipientrepo.New(pool)
	tmplRepo := templaterepo.New(pool)
	versionRepo := templateversionrepo.New(pool)
	signerRoleRepo := templateversionsignerrolerepo.New(pool)
	eventRepo := documenteventrepo.New(pool)
	accessTokenRepo := documentaccesstokenrepo.New(pool)
	fieldResponseRepo := documentfieldresponserepo.New(pool)

	mockPDF := &testhelper.MockPDFRenderer{}
	mockSigning := mocksigning.New()

	storageAdapter, err := localstorage.New(t.TempDir())
	require.NoError(t, err)

	eventEmitter := documentsvc.NewEventEmitter(eventRepo)
	notifSvc := documentsvc.NewNotificationService(
		noopnotification.New(), recipientRepo, docRepo, accessTokenRepo, "http://localhost:8080",
	)

	return documentsvc.NewDocumentService(
		docRepo, recipientRepo, tmplRepo, versionRepo, signerRoleRepo,
		mockPDF, mockSigning, storageAdapter, eventEmitter, notifSvc,
		30, accessTokenRepo, fieldResponseRepo,
		true,
	)
}

// newRiver creates a RiverService. When enabled=true it starts processing.
func newRiver(t *testing.T, pool *pgxpool.Pool, handler port.DocumentCompletedHandler, enabled bool) *riverqueue.RiverService {
	t.Helper()
	ctx := context.Background()

	cfg := config.WorkerConfig{Enabled: enabled, MaxWorkers: 2}
	svc, err := riverqueue.New(ctx, pool, cfg, handler, documentrepo.NewConcrete(pool))
	require.NoError(t, err)

	if enabled {
		require.NoError(t, svc.Start(ctx))
	}

	t.Cleanup(func() { _ = svc.Stop(ctx) })
	return svc
}

// createDocInProgress creates a document via the service (AWAITING_INPUT), then
// force-transitions it to IN_PROGRESS with mock signer info via SQL so it can
// be completed through webhook events.
func (infra *testInfra) createDocInProgress(t *testing.T, docSvc *documentsvc.DocumentService) *docData {
	t.Helper()
	ctx := context.Background()

	result, err := docSvc.CreateAndSendDocument(ctx, documentuc.CreateDocumentCommand{
		WorkspaceID:       infra.workspaceID,
		TemplateVersionID: infra.versionID,
		Title:             "Test Doc",
		Recipients: []documentuc.DocumentRecipientCommand{
			{RoleID: infra.roleIDs[0], Name: "Alice", Email: "alice@test.com"},
			{RoleID: infra.roleIDs[1], Name: "Bob", Email: "bob@test.com"},
		},
	})
	require.NoError(t, err)

	docID := result.Document.ID
	providerDocID := "prov-doc-" + docID[:8]
	pRcpt := [2]string{
		"prov-rcpt-" + result.Recipients[0].ID[:8],
		"prov-rcpt-" + result.Recipients[1].ID[:8],
	}

	// Transition doc: AWAITING_INPUT → IN_PROGRESS with signer info.
	_, err = infra.pool.Exec(ctx, `
		UPDATE execution.documents
		SET status = 'IN_PROGRESS', signer_document_id = $2, signer_provider = 'mock'
		WHERE id = $1`, docID, providerDocID)
	require.NoError(t, err)

	// Set recipients to SENT with provider IDs.
	for i, rcpt := range result.Recipients {
		_, err = infra.pool.Exec(ctx, `
			UPDATE execution.document_recipients
			SET signer_recipient_id = $2, status = 'SENT'
			WHERE id = $1`, rcpt.ID, pRcpt[i])
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		testhelper.CleanupDocument(t, infra.pool, docID)
		_, _ = infra.pool.Exec(ctx, "DELETE FROM river_job WHERE args->>'document_id' = $1", docID)
	})

	return &docData{
		docID:           docID,
		providerDocID:   providerDocID,
		recipientIDs:    [2]string{result.Recipients[0].ID, result.Recipients[1].ID},
		providerRcptIDs: pRcpt,
	}
}

// createCompletedEntity creates a document via CreateAndSendDocument, forces it
// to IN_PROGRESS via SQL, reloads it, and marks it COMPLETED in-memory so it
// can be passed to PersistAndNotify.
func (infra *testInfra) createCompletedEntity(t *testing.T, docSvc *documentsvc.DocumentService) *entity.Document {
	t.Helper()
	ctx := context.Background()

	result, err := docSvc.CreateAndSendDocument(ctx, documentuc.CreateDocumentCommand{
		WorkspaceID:       infra.workspaceID,
		TemplateVersionID: infra.versionID,
		Title:             "Test Doc",
		Recipients: []documentuc.DocumentRecipientCommand{
			{RoleID: infra.roleIDs[0], Name: "Alice", Email: "alice@test.com"},
			{RoleID: infra.roleIDs[1], Name: "Bob", Email: "bob@test.com"},
		},
	})
	require.NoError(t, err)
	docID := result.Document.ID

	// Force to IN_PROGRESS so MarkAsCompleted succeeds.
	_, err = infra.pool.Exec(ctx,
		"UPDATE execution.documents SET status = 'IN_PROGRESS' WHERE id = $1", docID)
	require.NoError(t, err)

	doc, err := documentrepo.New(infra.pool).FindByID(ctx, docID)
	require.NoError(t, err)
	require.NoError(t, doc.MarkAsCompleted())

	t.Cleanup(func() {
		testhelper.CleanupDocument(t, infra.pool, docID)
		_, _ = infra.pool.Exec(ctx, "DELETE FROM river_job WHERE args->>'document_id' = $1", docID)
	})

	return doc
}

func signedRecipientEvent(providerDocID, providerRcptID string) *port.WebhookEvent {
	return &port.WebhookEvent{
		EventType:           "recipient.signed",
		ProviderDocumentID:  providerDocID,
		ProviderRecipientID: providerRcptID,
		RecipientStatus:     ptr(entity.RecipientStatusSigned),
		Timestamp:           time.Now(),
	}
}

// waitForJob polls river_job until predicate is satisfied.
func waitForJob(t *testing.T, pool *pgxpool.Pool, docID string, pred func(state string, attempt int) bool, timeout time.Duration) {
	t.Helper()
	require.Eventually(t, func() bool {
		var state string
		var attempt int
		err := pool.QueryRow(context.Background(),
			"SELECT state, attempt FROM river_job WHERE args->>'document_id' = $1 ORDER BY id DESC LIMIT 1",
			docID,
		).Scan(&state, &attempt)
		return err == nil && pred(state, attempt)
	}, timeout, 200*time.Millisecond)
}

func jobCount(t *testing.T, pool *pgxpool.Pool, docID string) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(),
		"SELECT count(*) FROM river_job WHERE args->>'document_id' = $1", docID,
	).Scan(&n)
	require.NoError(t, err)
	return n
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestRiver_HappyPath creates a document with 2 signers, signs each via
// webhook events, and verifies the handler receives a fully-populated
// DocumentCompletedEvent with correct document, tenant, workspace, and
// recipient data.
func TestRiver_HappyPath(t *testing.T) {
	pool := testhelper.GetTestPool(t)
	infra := newTestInfra(t, pool, "RVR01")
	ctx := context.Background()

	eventCh := make(chan port.DocumentCompletedEvent, 1)
	handler := func(_ context.Context, ev port.DocumentCompletedEvent) error {
		eventCh <- ev
		return nil
	}

	docSvc := newDocService(t, pool)
	riverSvc := newRiver(t, pool, handler, true)
	docSvc.SetCompletionNotifier(riverSvc.Notifier())

	dd := infra.createDocInProgress(t, docSvc)

	// Sign recipient 1 — not all signed yet.
	require.NoError(t, docSvc.HandleWebhookEvent(ctx, signedRecipientEvent(dd.providerDocID, dd.providerRcptIDs[0])))

	select {
	case <-eventCh:
		t.Fatal("handler called prematurely after first recipient")
	case <-time.After(1 * time.Second):
		// expected
	}

	// Sign recipient 2 — all signed → completion → River job → handler.
	require.NoError(t, docSvc.HandleWebhookEvent(ctx, signedRecipientEvent(dd.providerDocID, dd.providerRcptIDs[1])))

	select {
	case ev := <-eventCh:
		assert.Equal(t, dd.docID, ev.DocumentID)
		assert.Equal(t, entity.DocumentStatusCompleted, ev.Status)
		assert.Equal(t, infra.tenantCode, ev.TenantCode)
		assert.NotEmpty(t, ev.WorkspaceCode)
		require.NotNil(t, ev.Title)
		assert.Equal(t, "Test Doc", *ev.Title)
		assert.False(t, ev.CreatedAt.IsZero())

		require.Len(t, ev.Recipients, 2)

		assert.Equal(t, "Signer", ev.Recipients[0].RoleName)
		assert.Equal(t, 1, ev.Recipients[0].SignerOrder)
		assert.Equal(t, "Alice", ev.Recipients[0].Name)
		assert.Equal(t, "alice@test.com", ev.Recipients[0].Email)
		assert.Equal(t, entity.RecipientStatusSigned, ev.Recipients[0].Status)
		assert.NotNil(t, ev.Recipients[0].SignedAt)

		assert.Equal(t, "Witness", ev.Recipients[1].RoleName)
		assert.Equal(t, 2, ev.Recipients[1].SignerOrder)
		assert.Equal(t, "Bob", ev.Recipients[1].Name)
		assert.Equal(t, "bob@test.com", ev.Recipients[1].Email)
	case <-time.After(15 * time.Second):
		t.Fatal("handler not called within timeout")
	}

	var dbStatus string
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT status FROM execution.documents WHERE id = $1", dd.docID,
	).Scan(&dbStatus))
	assert.Equal(t, "COMPLETED", dbStatus)
}

// TestRiver_TransactionalAtomicity uses insert-only mode (workers disabled)
// and verifies that PersistAndNotify atomically commits both the document
// status update and the River job in the same transaction.
func TestRiver_TransactionalAtomicity(t *testing.T) {
	pool := testhelper.GetTestPool(t)
	infra := newTestInfra(t, pool, "RVR02")
	ctx := context.Background()

	noopHandler := func(_ context.Context, _ port.DocumentCompletedEvent) error { return nil }
	docSvc := newDocService(t, pool)
	riverSvc := newRiver(t, pool, noopHandler, false) // insert-only

	doc := infra.createCompletedEntity(t, docSvc)

	require.NoError(t, riverSvc.Notifier().PersistAndNotify(ctx, doc))

	// Both must exist post-commit.
	var dbStatus string
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT status FROM execution.documents WHERE id = $1", doc.ID,
	).Scan(&dbStatus))
	assert.Equal(t, "COMPLETED", dbStatus)
	assert.Equal(t, 1, jobCount(t, pool, doc.ID))
}

// TestRiver_HandlerPanic verifies that a panicking handler does NOT crash the
// worker process. The panic is recovered and the job is retried by River.
func TestRiver_HandlerPanic(t *testing.T) {
	pool := testhelper.GetTestPool(t)
	infra := newTestInfra(t, pool, "RVR03")
	ctx := context.Background()

	var calls atomic.Int32
	handler := func(_ context.Context, _ port.DocumentCompletedEvent) error {
		calls.Add(1)
		panic("boom")
	}

	docSvc := newDocService(t, pool)
	riverSvc := newRiver(t, pool, handler, true)

	doc := infra.createCompletedEntity(t, docSvc)
	require.NoError(t, riverSvc.Notifier().PersistAndNotify(ctx, doc))

	// Wait for at least one retry (attempt >= 2 or retryable after first failure).
	waitForJob(t, pool, doc.ID, func(state string, attempt int) bool {
		return attempt >= 2 || state == "retryable"
	}, 30*time.Second)

	assert.GreaterOrEqual(t, calls.Load(), int32(1), "handler should have been invoked at least once")
}

// TestRiver_HandlerError verifies that returning an error causes River to mark
// the job as retryable and attempt it again, without losing the job.
func TestRiver_HandlerError(t *testing.T) {
	pool := testhelper.GetTestPool(t)
	infra := newTestInfra(t, pool, "RVR04")
	ctx := context.Background()

	var calls atomic.Int32
	handler := func(_ context.Context, _ port.DocumentCompletedEvent) error {
		calls.Add(1)
		return fmt.Errorf("temporary failure")
	}

	docSvc := newDocService(t, pool)
	riverSvc := newRiver(t, pool, handler, true)

	doc := infra.createCompletedEntity(t, docSvc)
	require.NoError(t, riverSvc.Notifier().PersistAndNotify(ctx, doc))

	waitForJob(t, pool, doc.ID, func(state string, attempt int) bool {
		return attempt >= 2 || state == "retryable"
	}, 30*time.Second)

	assert.GreaterOrEqual(t, calls.Load(), int32(1))

	// Job must NOT be completed.
	var state string
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT state FROM river_job WHERE args->>'document_id' = $1 ORDER BY id DESC LIMIT 1",
		doc.ID,
	).Scan(&state))
	assert.NotEqual(t, "completed", state)
}

// TestRiver_UniqueDedup calls PersistAndNotify twice for the same document and
// verifies that River's ByArgs+ByPeriod deduplication produces exactly one job.
func TestRiver_UniqueDedup(t *testing.T) {
	pool := testhelper.GetTestPool(t)
	infra := newTestInfra(t, pool, "RVR05")
	ctx := context.Background()

	noopHandler := func(_ context.Context, _ port.DocumentCompletedEvent) error { return nil }
	docSvc := newDocService(t, pool)
	riverSvc := newRiver(t, pool, noopHandler, false) // insert-only

	doc := infra.createCompletedEntity(t, docSvc)

	require.NoError(t, riverSvc.Notifier().PersistAndNotify(ctx, doc))
	require.NoError(t, riverSvc.Notifier().PersistAndNotify(ctx, doc))

	assert.Equal(t, 1, jobCount(t, pool, doc.ID), "dedup should prevent second job")
}

// TestRiver_NilNotifier verifies that without SetCompletionNotifier, a
// document still reaches COMPLETED via the fallback documentRepo.Update and
// NO River jobs are created.
func TestRiver_NilNotifier(t *testing.T) {
	pool := testhelper.GetTestPool(t)
	infra := newTestInfra(t, pool, "RVR06")
	ctx := context.Background()

	// DocumentService WITHOUT SetCompletionNotifier.
	docSvc := newDocService(t, pool)
	dd := infra.createDocInProgress(t, docSvc)

	// Sign both recipients → should complete via repo.Update (no River).
	require.NoError(t, docSvc.HandleWebhookEvent(ctx, signedRecipientEvent(dd.providerDocID, dd.providerRcptIDs[0])))
	require.NoError(t, docSvc.HandleWebhookEvent(ctx, signedRecipientEvent(dd.providerDocID, dd.providerRcptIDs[1])))

	var dbStatus string
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT status FROM execution.documents WHERE id = $1", dd.docID,
	).Scan(&dbStatus))
	assert.Equal(t, "COMPLETED", dbStatus)

	assert.Equal(t, 0, jobCount(t, pool, dd.docID), "no River jobs expected without notifier")
}

// TestRiver_ConcurrentWebhooksRace fires signing webhooks for two recipients
// simultaneously, forcing a race in AllSigned + MarkAsCompleted + PersistAndNotify.
// Verifies: document COMPLETED, exactly ONE River job (dedup), handler called.
func TestRiver_ConcurrentWebhooksRace(t *testing.T) {
	pool := testhelper.GetTestPool(t)
	infra := newTestInfra(t, pool, "RVR07")
	ctx := context.Background()

	var handlerCalls atomic.Int32
	handler := func(_ context.Context, _ port.DocumentCompletedEvent) error {
		handlerCalls.Add(1)
		return nil
	}

	docSvc := newDocService(t, pool)
	riverSvc := newRiver(t, pool, handler, true)
	docSvc.SetCompletionNotifier(riverSvc.Notifier())

	dd := infra.createDocInProgress(t, docSvc)

	// Fire both signing webhooks concurrently.
	var wg sync.WaitGroup
	errs := make([]error, 2)
	wg.Add(2)
	for i := range 2 {
		go func(idx int) {
			defer wg.Done()
			errs[idx] = docSvc.HandleWebhookEvent(ctx,
				signedRecipientEvent(dd.providerDocID, dd.providerRcptIDs[idx]))
		}(i)
	}
	wg.Wait()

	// At least one must succeed; both may succeed.
	successCount := 0
	for _, err := range errs {
		if err == nil {
			successCount++
		}
	}
	assert.GreaterOrEqual(t, successCount, 1, "at least one webhook must succeed")

	// Wait for handler.
	require.Eventually(t, func() bool {
		return handlerCalls.Load() >= 1
	}, 15*time.Second, 200*time.Millisecond)

	// Exactly one job (dedup).
	assert.Equal(t, 1, jobCount(t, pool, dd.docID))

	var dbStatus string
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT status FROM execution.documents WHERE id = $1", dd.docID,
	).Scan(&dbStatus))
	assert.Equal(t, "COMPLETED", dbStatus)
}

// TestRiver_DoubleCompletionIdempotent sends a COMPLETED document-status
// webhook twice and verifies idempotent behavior: document stays COMPLETED,
// only one River job exists.
func TestRiver_DoubleCompletionIdempotent(t *testing.T) {
	pool := testhelper.GetTestPool(t)
	infra := newTestInfra(t, pool, "RVR08")
	ctx := context.Background()

	var handlerCalls atomic.Int32
	handler := func(_ context.Context, _ port.DocumentCompletedEvent) error {
		handlerCalls.Add(1)
		return nil
	}

	docSvc := newDocService(t, pool)
	riverSvc := newRiver(t, pool, handler, true)
	docSvc.SetCompletionNotifier(riverSvc.Notifier())

	dd := infra.createDocInProgress(t, docSvc)

	completedEvent := &port.WebhookEvent{
		EventType:          "document.completed",
		ProviderDocumentID: dd.providerDocID,
		DocumentStatus:     ptr(entity.DocumentStatusCompleted),
		Timestamp:          time.Now(),
	}

	// First COMPLETED webhook → enqueues job.
	require.NoError(t, docSvc.HandleWebhookEvent(ctx, completedEvent))

	// Second COMPLETED webhook → dedup prevents second job.
	require.NoError(t, docSvc.HandleWebhookEvent(ctx, completedEvent))

	require.Eventually(t, func() bool {
		return handlerCalls.Load() >= 1
	}, 15*time.Second, 200*time.Millisecond)

	assert.Equal(t, 1, jobCount(t, pool, dd.docID), "dedup should prevent duplicate job")
}

// TestRiver_OrphanedJob enqueues a completion job, then deletes the document
// from the database BEFORE the worker processes it. Verifies the worker
// handles the missing document gracefully (error + retryable) instead of
// panicking, and the handler is never invoked.
func TestRiver_OrphanedJob(t *testing.T) {
	pool := testhelper.GetTestPool(t)
	infra := newTestInfra(t, pool, "RVR09")
	ctx := context.Background()

	var handlerCalls atomic.Int32
	handler := func(_ context.Context, _ port.DocumentCompletedEvent) error {
		handlerCalls.Add(1)
		return nil
	}

	docSvc := newDocService(t, pool)

	// Create document — manage cleanup manually since we delete it.
	result, err := docSvc.CreateAndSendDocument(ctx, documentuc.CreateDocumentCommand{
		WorkspaceID:       infra.workspaceID,
		TemplateVersionID: infra.versionID,
		Title:             "Orphaned Doc",
		Recipients: []documentuc.DocumentRecipientCommand{
			{RoleID: infra.roleIDs[0], Name: "Alice", Email: "alice@test.com"},
			{RoleID: infra.roleIDs[1], Name: "Bob", Email: "bob@test.com"},
		},
	})
	require.NoError(t, err)
	docID := result.Document.ID

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM river_job WHERE args->>'document_id' = $1", docID)
	})

	// Force IN_PROGRESS → reload → mark COMPLETED.
	_, err = pool.Exec(ctx,
		"UPDATE execution.documents SET status = 'IN_PROGRESS' WHERE id = $1", docID)
	require.NoError(t, err)

	doc, err := documentrepo.New(pool).FindByID(ctx, docID)
	require.NoError(t, err)
	require.NoError(t, doc.MarkAsCompleted())

	// Enqueue with workers DISABLED.
	disabledRiver := newRiver(t, pool, handler, false)
	require.NoError(t, disabledRiver.Notifier().PersistAndNotify(ctx, doc))
	assert.Equal(t, 1, jobCount(t, pool, docID))

	// Orphan the job: delete document + related records.
	for _, q := range []string{
		"DELETE FROM execution.document_access_tokens WHERE document_id = $1",
		"DELETE FROM execution.document_field_responses WHERE document_id = $1",
		"DELETE FROM execution.document_events WHERE document_id = $1",
		"DELETE FROM execution.document_recipients WHERE document_id = $1",
		"DELETE FROM execution.documents WHERE id = $1",
	} {
		_, _ = pool.Exec(ctx, q, docID)
	}

	// Now start workers — they will pick up the orphaned job.
	_ = newRiver(t, pool, handler, true)

	// Worker should fail (doc not found) → retryable or discarded.
	waitForJob(t, pool, docID, func(state string, _ int) bool {
		return state == "retryable" || state == "discarded"
	}, 30*time.Second)

	// Handler must NOT have been called (buildCompletedEvent fails first).
	assert.Equal(t, int32(0), handlerCalls.Load(), "handler should not run for orphaned document")
}

// TestRiver_EnvironmentDev verifies that a document in a sandbox workspace
// produces a DocumentCompletedEvent with Environment=dev.
func TestRiver_EnvironmentDev(t *testing.T) {
	pool := testhelper.GetTestPool(t)
	infra := newSandboxTestInfra(t, pool, "RENV1")
	ctx := context.Background()

	eventCh := make(chan port.DocumentCompletedEvent, 1)
	handler := func(_ context.Context, ev port.DocumentCompletedEvent) error {
		eventCh <- ev
		return nil
	}

	docSvc := newDocService(t, pool)
	riverSvc := newRiver(t, pool, handler, true)
	docSvc.SetCompletionNotifier(riverSvc.Notifier())

	dd := infra.createDocInProgress(t, docSvc)

	// Sign both recipients to trigger completion.
	require.NoError(t, docSvc.HandleWebhookEvent(ctx, signedRecipientEvent(dd.providerDocID, dd.providerRcptIDs[0])))
	require.NoError(t, docSvc.HandleWebhookEvent(ctx, signedRecipientEvent(dd.providerDocID, dd.providerRcptIDs[1])))

	select {
	case ev := <-eventCh:
		assert.Equal(t, dd.docID, ev.DocumentID)
		assert.Equal(t, entity.EnvironmentDev, ev.Environment, "sandbox workspace should produce dev environment")
		assert.Equal(t, entity.DocumentStatusCompleted, ev.Status)
	case <-time.After(15 * time.Second):
		t.Fatal("handler not called within timeout")
	}
}

// TestRiver_EnvironmentProd verifies that a document in a regular (non-sandbox)
// workspace produces a DocumentCompletedEvent with Environment=prod.
func TestRiver_EnvironmentProd(t *testing.T) {
	pool := testhelper.GetTestPool(t)
	infra := newTestInfra(t, pool, "RENV2")
	ctx := context.Background()

	eventCh := make(chan port.DocumentCompletedEvent, 1)
	handler := func(_ context.Context, ev port.DocumentCompletedEvent) error {
		eventCh <- ev
		return nil
	}

	docSvc := newDocService(t, pool)
	riverSvc := newRiver(t, pool, handler, true)
	docSvc.SetCompletionNotifier(riverSvc.Notifier())

	dd := infra.createDocInProgress(t, docSvc)

	// Sign both recipients to trigger completion.
	require.NoError(t, docSvc.HandleWebhookEvent(ctx, signedRecipientEvent(dd.providerDocID, dd.providerRcptIDs[0])))
	require.NoError(t, docSvc.HandleWebhookEvent(ctx, signedRecipientEvent(dd.providerDocID, dd.providerRcptIDs[1])))

	select {
	case ev := <-eventCh:
		assert.Equal(t, dd.docID, ev.DocumentID)
		assert.Equal(t, entity.EnvironmentProd, ev.Environment, "non-sandbox workspace should produce prod environment")
		assert.Equal(t, entity.DocumentStatusCompleted, ev.Status)
	case <-time.After(15 * time.Second):
		t.Fatal("handler not called within timeout")
	}
}
