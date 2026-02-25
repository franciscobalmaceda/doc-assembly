# Architecture

Doc Engine follows **Hexagonal Architecture** (Ports and Adapters) with **domain-based organization**.

## Directory Structure

```
internal/
├── core/                      # Domain Layer (business logic)
│   ├── entity/               # Domain entities and value objects (flat structure)
│   │   └── portabledoc/      # PDF document format types
│   ├── port/                 # Output ports (repository interfaces)
│   │
│   ├── usecase/              # Input ports organized by domain
│   │   ├── document/         # Document lifecycle and signing
│   │   ├── template/         # Template and version management
│   │   ├── organization/     # Tenant, workspace, member management
│   │   ├── injectable/       # Injectable definitions and assignments
│   │   ├── catalog/          # Folder and tag organization
│   │   └── access/           # System roles and access history
│   │
│   └── service/              # Business logic organized by domain
│       ├── document/         # Document services
│       ├── template/         # Template services + contentvalidator/
│       ├── organization/     # Organization services
│       ├── injectable/       # Injectable services + dependency resolution
│       ├── catalog/          # Catalog services
│       ├── access/           # Access control services
│       └── rendering/        # PDF rendering (pdfrenderer/)
│
├── adapters/
│   ├── primary/http/         # Driving adapters (HTTP API)
│   │   ├── controller/       # HTTP handlers
│   │   ├── dto/              # Request/Response DTOs
│   │   ├── mapper/           # Entity <-> DTO mappers
│   │   └── middleware/       # HTTP middleware
│   │
│   └── secondary/            # Driven adapters
│       ├── database/postgres/ # PostgreSQL repositories
│       └── llm/              # LLM providers (OpenAI, etc.)
│
└── infra/                    # Infrastructure
    ├── config/               # Configuration loading
    ├── server/               # HTTP server setup
    ├── di.go                 # Wire dependency injection
    └── initializer.go        # Application bootstrap
```

## Domain Organization

| Domain | Description |
|--------|-------------|
| `document` | Document creation, signing, webhooks |
| `template` | Template CRUD, versioning, content validation |
| `organization` | Tenants, workspaces, members |
| `injectable` | Injectable definitions and assignments |
| `catalog` | Folders and tags |
| `access` | System roles, access history |
| `rendering` | PDF generation |

## Entity Files by Domain

The `entity/` package uses a flat structure for simplicity. Files are logically grouped by domain:

| Domain | Entity Files |
|--------|-------------|
| **document** | `document.go`, `document_recipient.go` |
| **template** | `template.go`, `template_version.go` |
| **organization** | `tenant.go`, `workspace.go`, `user.go` |
| **injectable** | `injectable.go`, `system_injectable.go` |
| **catalog** | `folder.go`, `tag.go` |
| **access** | `user_access_history.go` |
| **shared** | `enum.go`, `errors.go`, `format.go`, `injector_context.go` |
| **rendering** | `portabledoc/` (subdirectory) |

## Dependency Flow

```
HTTP Request → Controller → UseCase (interface) → Service → Port (interface) → Repository
```
