# ProceedToSigning Concurrency Control

## Problem

When two signers call `ProceedToSigning` simultaneously on the same `AWAITING_INPUT` document, both could read the status, both render the PDF, and both upload to the signing provider — creating **duplicate documents in Documenso**.

Additionally, the background worker `ProcessPendingProviderDocuments` queries `PENDING_PROVIDER` documents and could race with the inline upload.

### Race Condition (Before Fix)

```mermaid
sequenceDiagram
    participant A as Signer A
    participant B as Signer B
    participant API as ProceedToSigning
    participant DB as PostgreSQL
    participant Typst as Typst Renderer
    participant Documenso as Documenso

    A->>API: POST /public/sign/{tokenA}/proceed
    B->>API: POST /public/sign/{tokenB}/proceed
    API->>DB: SELECT ... WHERE id = $1 (Signer A)
    API->>DB: SELECT ... WHERE id = $1 (Signer B)
    Note over DB: Both read status = AWAITING_INPUT
    API->>Typst: RenderPreview (Signer A)
    API->>Typst: RenderPreview (Signer B)
    Typst-->>API: PDF bytes (A)
    Typst-->>API: PDF bytes (B)
    API->>Documenso: UploadDocument (A)
    API->>Documenso: UploadDocument (B)
    Note over Documenso: Two signing documents created!
    Documenso-->>API: envelopeID_1
    Documenso-->>API: envelopeID_2
    Note over DB: Last write wins, envelopeID_1 orphaned
```

## Solution: CAS Claim + Worker Grace Period

### 1. CAS (Compare-And-Swap) Claim

An atomic `UPDATE ... WHERE status = 'AWAITING_INPUT'` ensures only one caller transitions the document to `PENDING_PROVIDER`. PostgreSQL's row-level locking guarantees mutual exclusion.

```sql
UPDATE execution.documents
SET status = 'PENDING_PROVIDER', updated_at = NOW()
WHERE id = $1 AND status = 'AWAITING_INPUT'
RETURNING ...
```

- **Winner** (rows affected = 1): Proceeds with render + upload.
- **Loser** (rows affected = 0): Receives `step: "processing"` response; frontend polls until ready.

### 2. Worker Grace Period

The background worker `ProcessPendingProviderDocuments` skips documents updated in the last 60 seconds:

```sql
WHERE status = 'PENDING_PROVIDER'
  AND updated_at < NOW() - INTERVAL '60 seconds'
```

This prevents the worker from racing with the winner's inline upload. If the winner crashes, the worker picks up the document after 60 seconds.

### Corrected Flow

```mermaid
sequenceDiagram
    participant A as Signer A
    participant B as Signer B
    participant API as ProceedToSigning
    participant DB as PostgreSQL
    participant Typst as Typst Renderer
    participant Store as Storage
    participant Documenso as Documenso

    A->>API: POST /proceed
    B->>API: POST /proceed

    API->>DB: CAS UPDATE (Signer A)
    Note over DB: status: AWAITING_INPUT -> PENDING_PROVIDER
    DB-->>API: row returned (claimed!)

    API->>DB: CAS UPDATE (Signer B)
    Note over DB: status != AWAITING_INPUT, 0 rows
    DB-->>API: no rows (not claimed)

    Note over API: Signer B reloads doc, sees PENDING_PROVIDER
    API-->>B: { step: "processing" }
    Note over B: Frontend polls every 3s

    API->>Typst: RenderPreview (Signer A only)
    Typst-->>API: PDF bytes
    API->>Store: Upload PDF
    API->>DB: UPDATE (persist PDF path)
    API->>Documenso: UploadDocument
    Documenso-->>API: envelopeID
    API->>DB: UPDATE (status=PENDING, signerDocID)
    API-->>A: { step: "signing", embeddedSigningUrl: "..." }

    Note over B: Next poll: doc is PENDING
    B->>API: POST /proceed (retry)
    API-->>B: { step: "signing", embeddedSigningUrl: "..." }
```

### Crash Recovery

```mermaid
flowchart TD
    A[CAS: AWAITING_INPUT -> PENDING_PROVIDER] --> B{Winner crashes?}

    B -->|No| C[Render PDF]
    C --> D[Store PDF + Update DB]
    D --> E[Upload to Documenso]
    E --> F[Update DB: status=PENDING]
    F --> G[Return signing URL]

    B -->|After CAS, before PDF| H[Doc: PENDING_PROVIDER, no PDF path]
    H --> I[Worker picks up after 60s]
    I --> J[No PDF path -> mark ERROR]

    B -->|After PDF stored| K[Doc: PENDING_PROVIDER + PDF path]
    K --> L[Worker picks up after 60s]
    L --> M[Download PDF, upload to Documenso]
    M --> N[Update DB: status=PENDING]
```

## Implementation

### Files Modified

| File | Change |
|---|---|
| `core/.../document_repo/queries.go` | `queryClaimForSigning` CAS query; 60s grace period on `queryFindPendingProviderForUpload` |
| `core/.../port/document_repository.go` | `ClaimForSigning(ctx, id) (*Document, bool, error)` |
| `core/.../document_repo/repo.go` | `ClaimForSigning` implementation using `scanDocument` + `pgx.ErrNoRows` |
| `core/.../usecase/document/pre_signing_usecase.go` | `StepProcessing = "processing"` |
| `core/.../service/document/pre_signing_service.go` | `claimAndRender` helper, `buildProcessingResponse`, `GetPublicSigningPage` PENDING_PROVIDER handling |
| `app/.../public-signing/types.ts` | `'processing'` added to step union |
| `app/.../public-signing/components/PublicSigningPage.tsx` | Retry loop in `handleProceed` + page load polling |

### Status Machine

```
AWAITING_INPUT ──CAS──> PENDING_PROVIDER ──upload──> PENDING ──> IN_PROGRESS ──> COMPLETED
                              │                                                    │
                              └──crash/fail──> ERROR                    DECLINED ──┘
                                                                        VOIDED
                                                                        EXPIRED
```

### Known Limitations

- If the winner crashes after the Documenso upload but before the final DB update, the background worker may upload again (duplicate in provider). This requires Documenso-side idempotency to fully solve.
- ERROR documents without `SignerDocumentID` cannot be retried by the retry worker.
