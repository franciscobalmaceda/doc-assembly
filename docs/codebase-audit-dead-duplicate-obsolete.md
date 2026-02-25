# Codebase Audit: Dead, Duplicate, and Obsolete Code

## Scope

Audit and cleanup executed for:

1. Removal of `/api/v1/render` route group and related wiring.
2. Duplicate code hotspots reported by `dupl`.
3. Obsolete/dead code scan after route removal.

Out of scope:

- Database schema/migrations (`db/src`).
- Functional redesign of auth model outside current API behavior.

---

## Implemented Changes

## 1) Removed `/api/v1/render` route wiring

- Removed render route setup from `core/internal/infra/server/http.go`.
- Removed `renderAuthenticator` constructor dependency from `NewHTTPServer`.
- Removed initializer wiring from `core/cmd/api/bootstrap/initializer.go`.

## 2) Removed obsolete render-auth API surface and middleware

- Deleted `core/internal/adapters/primary/http/middleware/custom_render_auth.go` (unreachable after route removal).
- Removed `RenderAuth` and `RenderClaimsContext` from `core/internal/adapters/primary/http/middleware/jwt_auth.go`.
- Removed `SetRenderAuthenticator` and `GetRenderAuthenticator` from `core/cmd/api/bootstrap/engine.go`.
- Removed render-auth SDK aliases from `core/sdk/interfaces.go`.
- Removed `core/internal/core/port/render_authenticator.go`.
- Removed `auth.render_providers` and `AuthConfig.GetAllOIDCProviders()` from config model.

## 3) Duplicate code simplifications

- `core/internal/adapters/primary/http/controller/error_handler.go`
  - Replaced repetitive `errors.Is` chains with categorized error lists and shared matcher.
- `core/internal/adapters/secondary/database/postgres/document_type_repo/repo.go`
  - Consolidated repeated query+count+scan patterns through shared helpers.
- `core/internal/adapters/secondary/database/postgres/template_version_repo/repo.go`
  - Centralized row scanning and collection logic for template versions.
- `core/internal/adapters/secondary/signing/documenso/adapter.go`
  - Consolidated repeated envelope action POST logic via `postEnvelopeAction`.
- `core/internal/core/service/document/notification_service.go`
  - Consolidated repeated notification flows via `notifyDocumentStatusChange`.
- `core/internal/extensions/injectors/datetime/day_now.go`
- `core/internal/extensions/injectors/datetime/year_now.go`
  - Shared numeric date injector behavior via `number_now_shared.go`.

## 4) Documentation alignment

- Rewrote `docs/backend/authentication-guide.md` to reflect current route/middleware architecture without render routes.
- Removed render-auth extension example from `docs/backend/extensibility-guide.md`.

---

## Validation Results

Executed checks:

1. `env -u GOROOT go test ./...` in `core/`: pass.
2. `env -u GOROOT go vet ./...` in `core/`: pass.
3. `env -u GOROOT /tmp/gobin/golangci-lint run --enable dupl --enable unused --enable staticcheck`: pass (`0 issues`).
4. Extended lint check:
   `env -u GOROOT /tmp/gobin/golangci-lint run --enable dupl --enable unused --enable staticcheck --enable ineffassign`: pass (`0 issues`).

Note:

- Local `make lint` is currently coupled to a `golangci-lint` binary built with Go 1.25.1, while active toolchain is Go 1.26.0. This can panic due toolchain mismatch, independent of repository code changes.

---

## Dead/Obsolete Code Findings

### Removed now-unreachable code

- Render-only middleware and route wiring removed as listed above.

No additional render-route-specific compatibility artifacts remain.

---

## Risk and Regression Assessment

- No route registration remains for `/api/v1/render/*`.
- Existing active flows remain wired:
  - Template preview (`/api/v1/content/templates/{templateId}/versions/{versionId}/preview`)
  - Internal create (`/api/v1/internal/documents/create`)
  - Public signing (`/public/*`)
- No DB or migration changes performed.

Overall regression risk is low for active routes.
