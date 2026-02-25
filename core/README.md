# Doc Engine

Document Assembly System API - A microservice for template management and document generation.

## Table of Contents

- [Doc Engine](#doc-engine)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
  - [Architecture](#architecture)
  - [Tech Stack](#tech-stack)
  - [Prerequisites](#prerequisites)
    - [Install Tools](#install-tools)
  - [Getting Started](#getting-started)
    - [1. Clone and Setup](#1-clone-and-setup)
    - [2. Configuration](#2-configuration)
    - [3. Build and Run](#3-build-and-run)
  - [Configuration](#configuration)
    - [Server](#server)
    - [Database (PostgreSQL)](#database-postgresql)
    - [Auth (Keycloak/JWKS)](#auth-keycloakjwks)
    - [Internal API](#internal-api)
    - [Documenso (e-signature)](#documenso-e-signature)
    - [Storage (S3/MinIO)](#storage-s3minio)
    - [Logging](#logging)
  - [API Endpoints](#api-endpoints)
  - [Sandbox \& Promotion](#sandbox--promotion)
  - [Development](#development)
    - [Make Commands](#make-commands)
    - [Code Style](#code-style)
    - [Logging](#logging-1)
  - [Extensibility](#extensibility)
  - [Integration Tests](#integration-tests)
  - [Role-Based Access Control](#role-based-access-control)

## Overview

Doc Engine is a Go-based microservice that provides:

- **Workspace Management**: Multi-tenant workspace organization with role-based access control
- **Template Management**: Create, edit, publish, and clone document templates
- **Injectable Variables**: Define and manage dynamic variables for document generation
- **Folder Organization**: Hierarchical folder structure for template organization
- **Tag System**: Flexible tagging for template categorization
- **Signer Role Management**: Configure signature roles and anchors for e-signature integration

## Architecture

The project follows **Hexagonal Architecture** (Ports and Adapters) with domain-based organization.

For detailed architecture documentation including directory structure, domain organization, and entity files, see **[docs/backend/architecture.md](../docs/backend/architecture.md)**.

## Tech Stack

| Component      | Technology    | Version |
| -------------- | ------------- | ------- |
| Language       | Go            | 1.25    |
| HTTP Framework | Gin           | 1.10.0  |
| Database       | PostgreSQL    | 15+     |
| DB Driver      | pgx/v5        | 5.7.2   |
| DI             | Wire          | 0.6.0   |
| Config         | Viper         | 1.19.0  |
| JWT            | golang-jwt/v5 | 5.3.0   |
| JWKS           | keyfunc/v3    | 3.7.0   |

## Prerequisites

- Go 1.25+
- PostgreSQL 15+
- golangci-lint (for linting)
- Wire CLI (for dependency injection)
- swag CLI (for Swagger docs)

### Install Tools

```bash
# Wire
go install github.com/google/wire/cmd/wire@latest

# Swagger
go install github.com/swaggo/swag/cmd/swag@latest

# golangci-lint
brew install golangci-lint  # macOS

# or
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Getting Started

### 1. Clone and Setup

```bash
cd apps/doc-engine
go mod tidy
```

### 2. Configuration

Default values are defined in `settings/app.yaml`. Override them using environment variables with the `DOC_ENGINE_` prefix:

```bash
export DOC_ENGINE_DATABASE_PASSWORD=your_password
export DOC_ENGINE_AUTH_JWKS_URL=https://your-keycloak/realms/your-realm/protocol/openid-connect/certs
export DOC_ENGINE_AUTH_ISSUER=https://your-keycloak/realms/your-realm
```

### 3. Build and Run

```bash
# Using Makefile with default .env
make build
make run

# Or specify a custom env file
make run ENV_FILE=.env.production

# Or directly (requires manually setting env vars)
go build -o bin/doc-engine ./cmd/api
./bin/doc-engine
```

## Configuration

Configuration is loaded from `settings/app.yaml` and can be overridden with environment variables using the prefix `DOC_ENGINE_`.

Copy `.env.example` to `.env` and configure the required values.

### Server

| Variable                             | Description                         | Default |
| ------------------------------------ | ----------------------------------- | ------- |
| `DOC_ENGINE_SERVER_PORT`             | HTTP server port                    | 8080    |
| `DOC_ENGINE_SERVER_READ_TIMEOUT`     | Read timeout (seconds)              | 30      |
| `DOC_ENGINE_SERVER_WRITE_TIMEOUT`    | Write timeout (seconds)             | 30      |
| `DOC_ENGINE_SERVER_SHUTDOWN_TIMEOUT` | Graceful shutdown timeout (seconds) | 10      |

### Database (PostgreSQL)

| Variable                                    | Description             | Default      |
| ------------------------------------------- | ----------------------- | ------------ |
| `DOC_ENGINE_DATABASE_HOST`                  | PostgreSQL host         | localhost    |
| `DOC_ENGINE_DATABASE_PORT`                  | PostgreSQL port         | 5432         |
| `DOC_ENGINE_DATABASE_USER`                  | Database user           | postgres     |
| `DOC_ENGINE_DATABASE_PASSWORD`              | Database password       | **required** |
| `DOC_ENGINE_DATABASE_NAME`                  | Database name           | doc_assembly |
| `DOC_ENGINE_DATABASE_SSL_MODE`              | SSL mode                | disable      |
| `DOC_ENGINE_DATABASE_MAX_POOL_SIZE`         | Max pool connections    | 10           |
| `DOC_ENGINE_DATABASE_MIN_POOL_SIZE`         | Min pool connections    | 2            |
| `DOC_ENGINE_DATABASE_MAX_IDLE_TIME_SECONDS` | Max idle time (seconds) | 300          |

### Auth (Keycloak/JWKS)

| Variable                   | Description            | Default      |
| -------------------------- | ---------------------- | ------------ |
| `DOC_ENGINE_AUTH_JWKS_URL` | Keycloak JWKS endpoint | **required** |
| `DOC_ENGINE_AUTH_ISSUER`   | JWT issuer             | **required** |
| `DOC_ENGINE_AUTH_AUDIENCE` | JWT audience           | **required** |

### Internal API

| Variable                          | Description                         | Default      |
| --------------------------------- | ----------------------------------- | ------------ |
| `DOC_ENGINE_INTERNAL_API_ENABLED` | Enable internal API                 | true         |
| `DOC_ENGINE_INTERNAL_API_API_KEY` | API key for service-to-service auth | **required** |

### Documenso (e-signature)

| Variable                              | Description              | Default                          |
| ------------------------------------- | ------------------------ | -------------------------------- |
| `DOC_ENGINE_DOCUMENSO_API_URL`        | Documenso API URL        | https://app.documenso.com/api/v1 |
| `DOC_ENGINE_DOCUMENSO_API_KEY`        | Documenso API key        | **required**                     |
| `DOC_ENGINE_DOCUMENSO_WEBHOOK_SECRET` | Webhook signature secret | -                                |

### Storage (S3/MinIO)

| Variable                      | Description             | Default          |
| ----------------------------- | ----------------------- | ---------------- |
| `DOC_ENGINE_STORAGE_BUCKET`   | S3 bucket name          | contracts-bucket |
| `DOC_ENGINE_STORAGE_REGION`   | AWS region              | us-east-1        |
| `DOC_ENGINE_STORAGE_ENDPOINT` | Custom endpoint (MinIO) | -                |

### Logging

| Variable                    | Description                          | Default |
| --------------------------- | ------------------------------------ | ------- |
| `DOC_ENGINE_LOGGING_LEVEL`  | Log level (debug, info, warn, error) | info    |
| `DOC_ENGINE_LOGGING_FORMAT` | Log format (json, text)              | json    |

## API Endpoints

For a complete list of API endpoints, authentication requirements, roles, and required headers, see **[docs/backend/authorization-matrix.md](../docs/backend/authorization-matrix.md)**.

## Public Signing Flow

Documents use a shared public URL with email verification. Recipients enter their email to receive a token-based signing link. The flow supports two paths: direct signing (no interactive fields) and form-first signing (with interactive fields).

For detailed documentation including sequence diagrams, security measures, and configuration, see **[docs/backend/public-signing-flow.md](../docs/backend/public-signing-flow.md)**.

## Sandbox & Promotion

Doc Engine supports sandbox environments for template development and testing before promoting to production.

For complete documentation on sandbox mode, promotion flow, and examples, see **[docs/backend/sandbox-promotion.md](../docs/backend/sandbox-promotion.md)**.

## Development

### Make Commands

```bash
make help             # Show all available commands
make build            # Build the binary
make run [ENV_FILE]   # Build and run (default: .env)
make test             # Run unit tests with coverage
make test-integration # Run integration tests (requires Docker)
make test-all         # Run all tests (unit + integration)
make coverage         # Open HTML coverage report (run 'make test' first)
make coverage-all     # Run all tests and open HTML coverage report
make lint             # Run golangci-lint
make wire             # Generate Wire DI code
make swagger          # Generate Swagger docs
make gen              # Generate all (Wire + Swagger)
make tidy             # Tidy dependencies
make clean            # Clean build artifacts
make dev              # Run with hot reload (requires air)
```

### Code Style

The project uses golangci-lint with errcheck, gosimple, govet, staticcheck, unused, gofmt, goimports, misspell, gocritic, revive, gosec, errorlint, exhaustive.

Run `make lint` before committing. For comprehensive Go coding standards, see **[docs/backend/go-best-practices.md](../docs/backend/go-best-practices.md)**.

### Logging

The project uses Go's standard `log/slog` package with a context-based handler. For complete documentation, see **[docs/backend/logging-guide.md](../docs/backend/logging-guide.md)**.

## Extensibility

Doc Engine supports custom **injectors**, **mappers**, and **init functions** to extend document generation with business-specific logic.

For complete documentation including creating injectors, mappers, init functions, and i18n, see **[docs/backend/extensibility-guide.md](../docs/backend/extensibility-guide.md)**.

## Integration Tests

Integration tests validate repository operations against a real PostgreSQL database using Testcontainers.

For complete documentation including setup, running tests, and troubleshooting, see **[docs/backend/integration-tests.md](../docs/backend/integration-tests.md)**.

## Role-Based Access Control

Workspace roles hierarchy (highest to lowest):

| Role     | Weight | Permissions              |
| -------- | ------ | ------------------------ |
| OWNER    | 50     | Full control             |
| ADMIN    | 40     | Manage members, settings |
| EDITOR   | 30     | Create/edit templates    |
| OPERATOR | 20     | Generate documents       |
| VIEWER   | 10     | Read-only access         |

For a complete authorization matrix with all endpoints, roles (System, Tenant, Workspace), and required headers, see **[docs/backend/authorization-matrix.md](../docs/backend/authorization-matrix.md)**.
