# AGENTS.md

This file provides guidance to AI Agents when working with code in this repository.

## Project Overview

**doc-assembly** is a multi-tenant document template builder with digital signature delegation to external providers (PandaDoc, Documenso, DocuSign).

**Stack**: Go 1.25 + React 19 + PostgreSQL 16 + Keycloak

## Monorepo Structure

```plaintext
core/       → Go backend (Hexagonal Architecture, Gin, Wire DI)
app/        → React SPA (TanStack Router, Zustand, TipTap)
db/         → Liquibase migrations (PostgreSQL)
docs/       → All project documentation
scripts/    → Tooling reutilizable por agents y CI
```

## Build and Development Commands

### Backend (`core/`)

```bash
make build            # Build (runs wire, swagger, lint, then compiles)
make run              # Run the service
make test             # Run unit tests with coverage
make test-integration # Run integration tests (Docker required)
make lint             # Run linter (golangci-lint)
make wire             # Generate Wire DI code
make swagger          # Generate Swagger docs
make gen              # Generate all (Wire + Swagger + Extensions)
make dev              # Hot reload development (requires air)
```

### Frontend (`app/`)

```bash
pnpm dev              # Start dev server (Vite with rolldown)
pnpm build            # Type-check (tsc -b) then build
pnpm lint             # ESLint for TS/TSX files
pnpm preview          # Preview production build
```

### Integration Tests

Integration tests require Docker and use Testcontainers (PostgreSQL + Liquibase):

```bash
# Run all integration tests
go test -C core -tags=integration -v -timeout 5m ./internal/adapters/secondary/database/postgres/...

# Run specific repository tests
go test -C core -tags=integration -v -run TestTenantRepo ./internal/adapters/secondary/database/postgres/...

# Run River worker tests
go test -C core -tags=integration -run TestRiver -v -count=1 ./internal/infra/riverqueue/
```

## Architecture (Cross-Component)

### Request Flow (Hexagonal)

```plaintext
HTTP Request
  → Middleware (JWT auth, tenant/workspace context, operation ID)
    → Controller (parse request DTO, validate)
      → UseCase interface → Service (business logic)
        → Port interface → Repository (SQL via pgx)
          → PostgreSQL
```

### Multi-Tenant Data Flow

1. Frontend Zustand stores hold current tenant/workspace selection
2. Axios interceptor in `api-client.ts` auto-attaches `Authorization`, `X-Tenant-ID`, `X-Workspace-ID` headers — never set these manually
3. Backend middleware extracts headers into request context
4. Services and repositories receive scoped context throughout the call chain

### RBAC (Three Levels)

1. **System**: SUPERADMIN, PLATFORM_ADMIN (global)
2. **Tenant**: OWNER, ADMIN
3. **Workspace**: OWNER, ADMIN, EDITOR, OPERATOR, VIEWER

SUPERADMIN auto-elevates to OWNER in any workspace/tenant.

### Public Signing Flow (No Auth)

Public endpoints (`/public/*`) require NO authentication. Two flows:

- **Email verification gate**: `/public/doc/{id}` → enter email → receive token via email
- **Token-based signing**: `/public/sign/{token}` → preview PDF → sign via embedded iframe

Token types: `SIGNING` (direct sign, no form) vs `PRE_SIGNING` (fill form first).
Anti-enumeration: `RequestAccess` always returns 200 regardless of email match.
Admin can invalidate all tokens via `POST /documents/{id}/invalidate-tokens`.

**Documentation**: [`docs/backend/public-signing-flow.md`](docs/backend/public-signing-flow.md)

### OpenAPI Spec

When working with API contracts, prefer using `mcp__doc-engine-api__*` tools to query the swagger interactively. Fallback: read `core/docs/swagger.yaml` directly (large file, ~3000+ lines).

---

## Backend (Go) — `core/`

### Layer Structure

- **`internal/core/entity/`** — Domain entities and value objects (flat structure)
  - `portabledoc/` — PDF document format types
  - Entity files by domain: document, template, organization, injectable, catalog, signing, access, shared
- **`internal/core/port/`** — Output port interfaces (repository contracts)
- **`internal/core/usecase/`** — Input port interfaces organized by domain: `document/`, `template/`, `organization/`, `injectable/`, `catalog/`, `access/`
- **`internal/core/service/`** — Business logic organized by domain (matching usecase folders)
- **`internal/adapters/primary/http/`** — Driving adapter (Gin HTTP handlers): `controller/`, `dto/`, `mapper/`, `middleware/`
- **`internal/adapters/secondary/database/postgres/`** — Driven adapter (each repo in own subpackage)
- **`internal/infra/`** — Infrastructure (config, DI, server bootstrap)

### Repository Structure

Each repository lives in its own subpackage under `postgres/`:

```
postgres/
├── client.go                    # Connection pool creation
├── tenantrepo/
│   ├── repo.go                  # Repository implementation
│   └── queries.go               # SQL queries
├── workspacerepo/
└── ...
```

### Wire DI

`internal/infra/di.go` defines ProviderSet → `cmd/api/wire.go` declares build → `cmd/api/wire_gen.go` auto-generated. Always run `make wire` after adding/changing services or repositories.

### Adding a New Feature

1. Define entity in `internal/core/entity/`
2. Create repository interface in `internal/core/port/`
3. Define use case interface with command structs in `internal/core/usecase/<domain>/`
4. Implement service in `internal/core/service/<domain>/`
5. Create PostgreSQL repository in `internal/adapters/secondary/database/postgres/<name>repo/`
6. Add DTOs in `internal/adapters/primary/http/dto/`
7. Create mapper in `internal/adapters/primary/http/mapper/`
8. Add controller in `internal/adapters/primary/http/controller/`
9. Register all in `internal/infra/di.go` with Wire bindings
10. Run `make wire` to regenerate DI

**Domain folders:** `document`, `template`, `organization`, `injectable`, `catalog`, `access`

### Integration Tests

Files with `//go:build integration` tag. Tests use `testhelper.GetTestPool(t)` from `internal/testing/testhelper/` which starts PostgreSQL 16 via Testcontainers, runs Liquibase migrations, and uses singleton pattern.

Test pattern:

```go
//go:build integration

package myrepo_test

func TestMyRepo_Operation(t *testing.T) {
    pool := testhelper.GetTestPool(t)
    repo := myrepo.New(pool)
    ctx := context.Background()
    // Setup, create, defer cleanup, assert
}
```

### Background Workers (River)

Document completion events are processed via [River](https://riverqueue.com), a PostgreSQL-native job queue that runs inside the same Go process. No external broker needed.

**Transactional guarantee:** The document status update (`COMPLETED`) and job enqueue happen in a **single PostgreSQL transaction** via `PersistAndNotify`. This prevents orphaned states on crashes.

**Flow:**
```
Webhook → DocumentService.persistDocUpdate()
  → completionNotifier != nil && doc.IsCompleted()
    → Notifier.PersistAndNotify(ctx, doc)
      → BEGIN TX → UPDATE doc status → INSERT river_job → COMMIT
  → else: plain documentRepo.Update(ctx, doc)
```

**Deduplication:** `ByArgs` + `ByPeriod(1h)` — same document_id produces at most 1 job per hour.
**Error handling:** Handler errors → exponential backoff retries. Panics → recovered, treated as error.

**SDK handler example:**
```go
handler := func(ctx context.Context, ev sdk.DocumentCompletedEvent) error {
    log.Printf("Doc %s completed: %d recipients", ev.DocumentID, len(ev.Recipients))
    return nil // return error to retry
}
```

**Key files:**

- `internal/infra/riverqueue/` — River client, notifier, worker, job args
  - `client.go` — `RiverService` lifecycle (New, Start, Stop, Notifier)
  - `notifier.go` — `PersistAndNotify` transactional enqueue
  - `worker.go` — `DocumentCompletedWorker` builds event from DB, calls handler
  - `args.go` — `DocumentCompletedArgs` with Kind() and dedup InsertOpts()
- `internal/core/port/document_completion.go` — `DocumentCompletionNotifier`, `DocumentCompletedHandler`
- `sdk/worker.go` — Re-exported types for SDK consumers

**Integration tests:**

```bash
go test -C core -tags=integration -run TestRiver -v -count=1 ./internal/infra/riverqueue/
```

9 tests cover: happy path, transactional atomicity, panic/error recovery, handler error retry, dedup, nil notifier fallback, concurrent webhook race, double completion idempotency, orphaned job.

**Documentation:** [`docs/backend/worker-queue-guide.md`](docs/backend/worker-queue-guide.md)

### Configuration

Config loaded from `settings/app.yaml`, overridden via `DOC_ENGINE_` prefixed env vars.

Key variables:

- `DOC_ENGINE_DATABASE_HOST/PORT/USER/PASSWORD/NAME` — PostgreSQL connection
- `DOC_ENGINE_AUTH_JWKS_URL` — Keycloak JWKS endpoint
- `DOC_ENGINE_AUTH_ISSUER` — JWT issuer validation
- `DOC_ENGINE_WORKER_ENABLED` — Enable River job queue workers (default: `false`)
- `DOC_ENGINE_WORKER_MAX_WORKERS` — Max concurrent worker goroutines (default: `10`)

### Logging Guidelines

Uses `log/slog` with a **ContextHandler** that automatically extracts attributes from `context.Context`.

```go
// ALWAYS use context-aware functions
slog.InfoContext(ctx, "user created", "user_id", user.ID)
slog.ErrorContext(ctx, "operation failed", "error", err)
ctx = logging.WithAttrs(ctx, slog.String("tenant_id", tenantID))
```

**Do NOT:** Inject `*slog.Logger` as dependency, call `slog.Default()`, use `slog.Info()` without context, log sensitive data.

**Documentation:** [`docs/backend/logging-guide.md`](docs/backend/logging-guide.md)

### Go Best Practices

**Documentation:** [`docs/backend/go-best-practices.md`](docs/backend/go-best-practices.md)

**Reference when:** Writing functions, designing APIs, handling errors, working with concurrency, or reviewing code.

### Extensibility System

Custom injectors, mappers, and initialization logic.

- `//docengine:injector` — Mark struct as injector (multiple allowed)
- `//docengine:mapper` — Mark struct as mapper (ONE only)
- `//docengine:init` — Mark function as init (ONE only)
- `make gen` — Regenerate `internal/extensions/registry_gen.go`

**Key files:** `internal/extensions/injectors/`, `internal/extensions/mappers/`, `internal/extensions/init.go`, `settings/injectors.i18n.yaml`

**Documentation:** [`docs/backend/extensibility-guide.md`](docs/backend/extensibility-guide.md)

### Public Signing Flow (Backend)

**Key services:**

- `internal/core/service/document/document_access_service.go` — `RequestAccess()`, email gate
- `internal/core/service/document/pre_signing_service.go` — `GetPublicSigningPage()`, `SubmitPreSigningForm()`, `ProceedToSigning()`, `InvalidateTokens()`
- `internal/core/service/document/notification_service.go` — `NotifyDocumentCreated()`, `SendAccessLink()`

**Key controllers:**

- `internal/adapters/primary/http/controller/public_document_access_controller.go` — `/public/doc/*`
- `internal/adapters/primary/http/controller/public_signing_controller.go` — `/public/sign/*`

**Patterns:**

- Anti-enumeration: `RequestAccess` returns nil (200) for invalid emails, missing docs, rate limits
- Token types: `SIGNING` (no interactive fields) vs `PRE_SIGNING` (has interactive fields)
- Tokens: 128-char hex, single-use (`used_at`), expiring (configurable TTL)
- Rate limiting: per document+recipient pair, configurable in `settings/app.yaml` → `public_access`
- `buildSigningURL()` fallback: active token → `/public/doc/{docID}`

### Mandatory Documentation Updates

#### Authorization Matrix (`docs/backend/authorization-matrix.md`)

**MUST update** when: New endpoint, permission change, new role, header requirement change, new controller, authorization middleware modification.

#### Extensibility Guide (`docs/backend/extensibility-guide.md`)

**MUST update** when: Changes to `port.Injector`, `port.RequestMapper`, `InitFunc`, `InjectorContext`, formatters, code markers, extensions directory, or code generation.

#### Go Best Practices (`docs/backend/go-best-practices.md`)

**SHOULD update** when: New patterns, project conventions, modern Go features, anti-patterns discovered.

### Mandatory Verification Checklist

**BEFORE considering any complex development work as complete**, agents MUST verify:

| Command                                         | Expected Result                        |
| ----------------------------------------------- | -------------------------------------- |
| `make wire` (in `core/`)                        | Regenerated successfully               |
| `make build` (in `core/`)                       | Compiled without errors                |
| `make test` (in `core/`)                        | All unit tests passed                  |
| `make lint` (in `core/`)                        | No lint errors                         |
| `go build -tags=integration ./...` (in `core/`) | Integration tests compile              |
| `make test-integration` (in `core/`)            | All E2E tests passed (requires Docker) |

> **IMPORTANT:** Files with `//go:build integration` tag are NOT compiled by `make test` — they require `-tags=integration` flag.

---

## Frontend (React) — `app/`

React 19 + TypeScript SPA for a multi-tenant document assembly platform. Uses Vite (rolldown-vite) for bundling.

**Full architecture guide:** [`docs/frontend/architecture.md`](docs/frontend/architecture.md)

### Routing

- **TanStack Router** with file-based routing in `src/routes/`
- Routes auto-generated to `src/routeTree.gen.ts` by `@tanstack/router-vite-plugin`
- Root route (`__root.tsx`) enforces tenant selection before navigation

### State Management

- **Zustand** stores with persistence:
  - `auth-store.ts`: JWT token and system roles
  - `app-context-store.ts`: Current tenant and workspace context
  - `theme-store.ts`: Light/dark theme preference

### Authentication & Authorization

- **Keycloak** integration via `keycloak-js` (mock with `VITE_USE_MOCK_AUTH=true`)
- **RBAC system** in `src/features/auth/rbac/`:
  - Three role levels: System, Tenant, Workspace
  - `usePermission()` hook and `<PermissionGuard>` component
- **Authorization matrix:** [`docs/backend/authorization-matrix.md`](docs/backend/authorization-matrix.md) — **ALWAYS** consult before implementing permission checks.

### API Layer

- Axios client (`src/lib/api-client.ts`) auto-attaches `Authorization`, `X-Tenant-ID`, `X-Workspace-ID` headers
- Backend expected at `VITE_API_URL` (default: `http://localhost:8080/api/v1`)
- **OpenAPI spec:** Prefer `mcp__doc-engine-api__*` MCP tools. Setup: [`docs/frontend/mcp-setup.md`](docs/frontend/mcp-setup.md). Fallback: `core/docs/swagger.yaml`.

### Feature Structure

Features organized in `src/features/` with `api/`, `components/`, `hooks/`, `types/` subfolders.
Current features: `auth`, `tenants`, `workspaces`, `documents`, `editor`, `signing`, `public-signing`

### Public Routes (No Auth)

Routes under `src/features/public-signing/`:

- `PublicDocumentAccessPage` — email verification gate (`/public/doc/{id}`)
- `PublicSigningPage` — token-based signing (`/public/sign/{token}`)
- `EmbeddedSigningFrame` — signing provider iframe
- `PDFPreview` — on-demand PDF rendering

These use a separate axios instance without auth interceptors.

### Styling

- **Tailwind CSS** with shadcn/ui-style CSS variables, dark mode via `class` strategy
- Colors defined as HSL CSS variables in `index.css`
- **Design System:** [`docs/frontend/design-system.md`](docs/frontend/design-system.md) — **ALWAYS** consult before UI changes.

### Rich Text Editor

**TipTap** editor with StarterKit in `src/features/editor/`. Prose styling via `@tailwindcss/typography`.

### i18n

**i18next** with browser detection. Translations in `public/locales/{lng}/translation.json`. Supports: `en`, `es`.

### Environment Variables

```plaintext
VITE_API_URL              # Backend API base URL
VITE_KEYCLOAK_URL         # Keycloak server URL
VITE_KEYCLOAK_REALM       # Keycloak realm name
VITE_KEYCLOAK_CLIENT_ID   # Keycloak client ID
VITE_USE_MOCK_AUTH        # Set to "true" to skip Keycloak (dev only)
VITE_BASE_PATH            # Base path for public URLs (default: empty)
```

### Path Aliases

`@/` maps to `./src/` (configured in vite.config.ts)

---

## Database Schema

Managed by Liquibase in `db/`. **Agents must NEVER modify `db/src/` files directly** — only read for context and suggest changes to the user. See `db/DATABASE.md` for full schema docs.

```
db/
├── changelog.master.xml          # Master changelog
├── liquibase-*.properties        # Environment configurations
├── src/                          # Changesets by domain
│   ├── schemas/, types/, tables/, indexes/, constraints/, triggers/, content/
└── DATABASE.md                   # Model documentation
```

**Pitfalls:**

- Forgetting `splitStatements="false"` for PL/pgSQL functions
- Wrong changeset ID format (use `{table}:{operation}[:{spec}]`)
- Not using triggers for `updated_at` columns

## Cross-Component Patterns

### Multi-Tenant Headers

All API requests require: `Authorization` (Bearer JWT), `X-Tenant-ID` (UUID), `X-Workspace-ID` (UUID).

### Environment Variables

| Component | Prefix         | Example                    |
| --------- | -------------- | -------------------------- |
| Backend   | `DOC_ENGINE_*` | `DOC_ENGINE_DATABASE_HOST` |
| Frontend  | `VITE_*`       | `VITE_API_URL`             |

## PR Checklist

1. `make build && make test && make lint` in `core/`
2. `pnpm build && pnpm lint` in `app/`
3. `go build -tags=integration ./...` in `core/` (verify integration tests compile)
4. Update `docs/backend/authorization-matrix.md` if endpoints changed
5. Update `docs/backend/extensibility-guide.md` if injector/mapper interfaces changed
6. Run `make gen` if extensibility markers changed

## Common Pitfalls

### Backend

- Forgetting `make wire` after adding new services/repos
- Missing `-tags=integration` when testing integration code (not compiled by `make test`)
- Using `slog.Info()` instead of `slog.InfoContext(ctx, ...)`

### Frontend

- Not checking authorization matrix before implementing permissions
- Not consulting design system before UI changes
- Manually setting auth/tenant headers (api-client.ts handles this)

### Cross-Component

- Not syncing OpenAPI spec after backend changes (`make swagger`)
- Inconsistent error handling between layers

## Scripts & Tools

### docml2json — Metalanguage to PortableDocument JSON

**Path**: `scripts/docml2json/`

Converts `.docml` text files into valid PortableDocument v1.1.0 JSON importable by the editor.

```bash
python3 scripts/docml2json/docml2json.py input.docml              # → input.json
python3 scripts/docml2json/docml2json.py input.docml -o out.json   # explicit output
python3 scripts/docml2json/docml2json.py *.docml                   # batch mode
```

| File                  | Description                                   |
| --------------------- | --------------------------------------------- |
| `docml2json.py`       | Conversion script (Python 3, no dependencies) |
| `DOCML-REFERENCIA.md` | Full metalanguage syntax reference            |
| `example.docml`       | Complete working example with all node types  |

## Documentation Index

```
docs/
├── backend/
│   ├── architecture.md             # Backend architecture and layers
│   ├── authentication-guide.md     # Auth middleware and JWT flow
│   ├── authorization-matrix.md     # All endpoints with required roles
│   ├── extensibility-guide.md      # Custom injectors, mappers, init
│   ├── getting-started.md          # Backend setup guide
│   ├── go-best-practices.md        # Go coding standards
│   ├── integration-tests.md        # Testcontainers setup and patterns
│   ├── logging-guide.md            # slog context-based logging
│   ├── public-signing-flow.md      # Signing flow (Mermaid diagrams)
│   ├── sandbox-promotion.md        # Sandbox mode and promotion
│   └── worker-queue-guide.md       # River job queue architecture
├── frontend/
│   ├── architecture.md             # Frontend architecture and patterns
│   ├── design-system.md            # Visual tokens, colors, typography
│   └── mcp-setup.md               # OpenAPI MCP tool setup
├── codebase-audit-dead-duplicate-obsolete.md
├── internal-api-document-creation-flow.md
├── proceed-to-signing-concurrency.md
├── public-signing-flow-detail.md
└── template-preview-flow.md
```

## Key Technologies

- **Go 1.25**, **Gin** for HTTP, **pgx/v5** for PostgreSQL
- **Wire** for compile-time DI
- **River** for PostgreSQL-native job queue
- **Keycloak/JWKS** for JWT authentication
- **React 19**, **TanStack Router**, **Zustand**, **TipTap 3**
- **Tailwind CSS** with shadcn/ui patterns
- **Testcontainers** for integration tests (PostgreSQL + Liquibase)
- **golangci-lint** with errcheck, gosimple, govet, staticcheck, gosec, revive, errorlint
