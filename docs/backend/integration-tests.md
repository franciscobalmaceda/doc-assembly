# Integration Tests

The project includes comprehensive integration tests that validate repository operations against a real PostgreSQL database using **[Testcontainers](https://golang.testcontainers.org)**.

## Prerequisites

- **Docker** installed and running
- ~500MB disk space for Docker images (downloaded on first run)

## Test Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  go test -tags=integration ./...                            │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    testhelper.GetTestPool(t)                │
│                                                             │
│  ┌─────────────────┐    ┌────────────────────────────────┐  │
│  │ PostgreSQL      │◄───│ Liquibase Container            │  │
│  │ Container       │    │ (applies all migrations)       │  │
│  │ (postgres:16)   │    │ (liquibase:4.30-alpine)        │  │
│  └─────────────────┘    └────────────────────────────────┘  │
│           │                                                  │
│           ▼                                                  │
│  ┌─────────────────────────────────────────────────────────┐│
│  │ Singleton pattern - one container per test run          ││
│  │ Tests clean up their own data with defer functions      ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

## Running Integration Tests

```bash
# Run all integration tests
go test -tags=integration -v -timeout 10m ./internal/adapters/secondary/database/postgres/...

# Run specific test file
go test -tags=integration -v -run TestTenantRepo ./internal/adapters/secondary/database/postgres/...

# Run with short timeout (after first run, images are cached)
go test -tags=integration -v -timeout 5m ./internal/adapters/secondary/database/postgres/...

# Using Make
make test-integration
```

## Test Coverage

| Repository | Tests | Coverage |
|------------|-------|----------|
| TenantRepo | 14 | CRUD, constraints, triggers |
| WorkspaceRepo | 10 | CRUD, enums, unique constraints |
| WorkspaceMemberRepo | 11 | CRUD, roles, cascades |
| TenantMemberRepo | 13 | CRUD, roles, cascades |
| FolderRepo | ~10 | Hierarchy, path validation |
| TagRepo | ~10 | CRUD, workspace isolation |
| InjectableRepo | ~10 | CRUD, workspace isolation |
| TemplateRepo | ~15 | CRUD, status workflow |
| TemplateVersionRepo | ~15 | Versioning, publishing |
| TemplateVersionRelationsRepo | ~10 | Injectables, signer roles |

## Test Timing

| Phase | Duration |
|-------|----------|
| First run (download images) | ~30-60s |
| Container startup | ~2-3s |
| Liquibase migrations | ~3-5s |
| All tests execution | ~1-2s |
| **Total (cached images)** | **~15-20s** |

## Testcontainers Stack

| Component | Image | Version |
|-----------|-------|---------|
| testcontainers-go | - | v0.40.0 |
| PostgreSQL | postgres:16-alpine | 16 |
| Liquibase | liquibase/liquibase:4.30-alpine | 4.30 |

## Writing New Integration Tests

1. Add build tag at the top of the file:
   ```go
   //go:build integration

   package postgres_test
   ```

2. Use `getTestPool(t)` to get the database connection:
   ```go
   func TestMyRepo_Operation(t *testing.T) {
       pool := getTestPool(t)
       repo := postgres.NewMyRepository(pool)
       ctx := context.Background()

       // Test logic...

       defer cleanup(t, pool, id) // Always clean up
   }
   ```

3. Generate valid UUIDs with `testUUID()`:
   ```go
   entity := &entity.MyEntity{
       ID:   testUUID(),
       Name: "Test Entity",
   }
   ```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `Cannot connect to Docker daemon` | Ensure Docker Desktop is running |
| `Timeout waiting for container` | Increase timeout or check Docker resources |
| `Liquibase migration failed` | Check `db/changelog.master.xml` syntax |
| `Tests skipped` | Verify Docker is running and accessible |
