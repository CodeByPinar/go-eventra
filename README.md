# Eventra Auth Platform

![Eventra Overview Widget](docs/assets/eventra-readme-widget.svg)

Eventra is a production-oriented authentication platform built with Go and PostgreSQL, with a premium React console for real-time auth flow testing.

[![Go Version](https://img.shields.io/badge/Go-1.26.1-00ADD8?style=for-the-badge&logo=go&logoColor=white)](go.mod)
[![Backend CI](https://img.shields.io/github/actions/workflow/status/CodeByPinar/go-eventra/ci-backend.yml?branch=main&style=for-the-badge&label=Backend%20CI)](https://github.com/CodeByPinar/go-eventra/actions/workflows/ci-backend.yml)
[![Frontend CI](https://img.shields.io/github/actions/workflow/status/CodeByPinar/go-eventra/ci-frontend.yml?branch=main&style=for-the-badge&label=Frontend%20CI)](https://github.com/CodeByPinar/go-eventra/actions/workflows/ci-frontend.yml)
[![License](https://img.shields.io/badge/License-Unspecified-6B7280?style=for-the-badge&logo=bookstack&logoColor=white)](README.md#license)

![Eventra Thumbnail](docs/assets/eventra-thumbnail.svg)

## Table of Contents

- [What This Project Includes](#what-this-project-includes)
- [Core Features](#core-features)
- [Architecture](#architecture)
- [Project Snapshot](#project-snapshot)
- [Auth Lifecycle](#auth-lifecycle)
- [Repository Layout](#repository-layout)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [API Surface](#api-surface)
- [Testing](#testing)
- [Operational Defaults](#operational-defaults)
- [Security Notes](#security-notes)
- [Roadmap](#roadmap)
- [Developer](#developer)
- [License](#license)
- [Troubleshooting](#troubleshooting)

## What This Project Includes

- Go backend auth service (REST API)
- PostgreSQL persistence layer
- JWT access token issuance
- Refresh token rotation and revocation
- Access-token blacklist support for logout/rotation
- Login protection mechanisms (rate limiting and account lock escalation)
- Security audit event pipeline
- React + Vite frontend auth console

## Core Features

- Register users with input validation and password hashing (bcrypt)
- Login with failed-attempt tracking and temporary lockouts
- Refresh endpoint with rotation semantics
- Logout with refresh token revoke and access token blacklist
- Protected profile endpoint (`/api/v1/auth/me`)
- Health endpoint for readiness checks
- CORS allowlist controls via environment variable
- HTTP hardening headers and request body size limits

## Architecture

```mermaid
flowchart LR
  subgraph FE[Frontend]
    UI[React + Vite Console]
  end

  subgraph API[Go Auth API]
    Router[Router + Middleware]
    Handler[Auth Handler]
    Service[Auth Service Use Case]
    Security[JWT Password Token Utils]
  end

  subgraph DB[PostgreSQL]
    Users[(users)]
    Refresh[(refresh_tokens)]
    LoginSec[(login_security)]
    Blacklist[(token_blacklist)]
    Audit[(security_audit)]
  end

  UI -->|JSON HTTP| Router
  Router --> Handler
  Handler --> Service
  Service --> Security
  Service --> Users
  Service --> Refresh
  Service --> LoginSec
  Service --> Blacklist
  Service --> Audit
```

```text
Frontend (React + Vite)
   |
   | HTTP (JSON)
   v
Auth API (Go net/http)
   |- Delivery: internal/delivery/httpserver
   |- Use Case: internal/usecase/auth
   |- Repository: internal/repository/postgres
   |- Security: pkg/security
   v
PostgreSQL
```

Request flow summary:

1. HTTP handlers validate and decode requests.
2. Auth service executes business rules (validation, credential checks, lock logic, token workflow).
3. Repositories read/write users, refresh tokens, security state, audit data, blacklist state.
4. Response payloads return token pairs and user claims where appropriate.

## Project Snapshot

| Area | Value |
| --- | --- |
| Backend runtime | Go net/http |
| Frontend runtime | React 19 + Vite 8 + TypeScript |
| Database | PostgreSQL |
| Auth endpoints | 6 |
| Default backend URL | `http://localhost:8080` |
| Default frontend URL | `http://localhost:5173` |
| Primary auth model | JWT access token + rotating refresh token |

## Auth Lifecycle

```mermaid
sequenceDiagram
  autonumber
  participant U as Client UI
  participant A as Auth API
  participant D as PostgreSQL

  U->>A: POST /api/v1/auth/register
  A->>D: Insert user + refresh token hash
  A-->>U: access_token + refresh_token

  U->>A: POST /api/v1/auth/login
  A->>D: Validate user + password + lock state
  A-->>U: access_token + refresh_token

  U->>A: GET /api/v1/auth/me (Bearer access_token)
  A->>D: Optional blacklist check
  A-->>U: user_id + email

  U->>A: POST /api/v1/auth/refresh
  A->>D: Validate refresh hash + revoke previous + blacklist previous access token
  A-->>U: new access_token + new refresh_token

  U->>A: POST /api/v1/auth/logout
  A->>D: Revoke refresh token + blacklist current access token
  A-->>U: 204 No Content
```

## Repository Layout

```text
api/                        # API documentation
cmd/eventra/                # Application entrypoint
configs/
  .env.example              # Environment template
  migrations/               # SQL migrations
frontend/                   # React UI console
internal/
  app/                      # App bootstrap/wiring
  config/                   # Env config loader
  delivery/httpserver/      # Router, handlers, middleware
  domain/user/              # Domain entity
  repository/postgres/      # Data access layer
  usecase/auth/             # Core auth business logic
pkg/
  database/                 # DB setup
  security/                 # JWT/password/token utilities
scripts/
  apply_migration.go        # Migration helper
  auth_smoke_test.ps1       # End-to-end smoke test
```

## Prerequisites

- Go 1.26+
- Node.js 20+
- npm 10+
- PostgreSQL 14+
- PowerShell (for provided smoke test script)

## Quick Start

### 1) Create database

```sql
CREATE DATABASE eventra;
```

### 2) Configure environment

Use the template in `configs/.env.example`.

PowerShell example:

```powershell
$env:PORT="8080"
$env:DB_URL="postgres://postgres:postgres@localhost:5432/eventra?sslmode=disable"
$env:JWT_SECRET="change-me-in-dev"
$env:JWT_EXPIRATION="24h"
$env:REFRESH_TOKEN_EXPIRATION="168h"
$env:CORS_ALLOWED_ORIGINS="http://localhost:5173,http://127.0.0.1:5173"
$env:SECURITY_ALERT_WEBHOOK_URL=""
$env:SECURITY_ALERT_WEBHOOK_FORMAT="slack"
```

### 3) Apply migrations

Option A - direct SQL with psql:

```powershell
psql "postgres://postgres:postgres@localhost:5432/eventra?sslmode=disable" -f configs/migrations/000001_create_users.up.sql
psql "postgres://postgres:postgres@localhost:5432/eventra?sslmode=disable" -f configs/migrations/000002_create_refresh_tokens.up.sql
psql "postgres://postgres:postgres@localhost:5432/eventra?sslmode=disable" -f configs/migrations/000003_security_hardening.up.sql
```

Option B - migration helper:

```powershell
go run ./scripts/apply_migration.go -file configs/migrations/000001_create_users.up.sql
go run ./scripts/apply_migration.go -file configs/migrations/000002_create_refresh_tokens.up.sql
go run ./scripts/apply_migration.go -file configs/migrations/000003_security_hardening.up.sql
```

### 4) Run backend API

```powershell
go run ./cmd/eventra
```

Backend default: `http://localhost:8080`

### 5) Run frontend console

```powershell
cd frontend
npm install
npm run dev
```

Frontend default: `http://localhost:5173`

## Configuration

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `PORT` | No | `8080` | API server port |
| `DB_URL` | Yes | - | PostgreSQL connection string |
| `JWT_SECRET` | Yes | - | JWT signing secret |
| `JWT_EXPIRATION` | No | `24h` | Access token lifetime |
| `REFRESH_TOKEN_EXPIRATION` | No | `168h` | Refresh token lifetime |
| `CORS_ALLOWED_ORIGINS` | No | `http://localhost:5173,http://127.0.0.1:5173` | Comma-separated CORS allowlist |
| `SECURITY_ALERT_WEBHOOK_URL` | No | empty | Optional outbound security alert target |
| `SECURITY_ALERT_WEBHOOK_FORMAT` | No | empty/slack | Alert payload format selector |

## API Surface

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/health` | No | Service health |
| POST | `/api/v1/auth/register` | No | Register new account |
| POST | `/api/v1/auth/login` | No | Authenticate with email/password |
| POST | `/api/v1/auth/refresh` | No | Rotate refresh token, issue new access token |
| POST | `/api/v1/auth/logout` | No | Revoke refresh token and blacklist access token |
| GET | `/api/v1/auth/me` | Yes | Return user claims from JWT |

Full request and response examples:

- `api/auth.md`

Quick smoke checks:

```powershell
# Health
Invoke-RestMethod -Method Get -Uri "http://localhost:8080/health"

# Auth workflow smoke suite
powershell -ExecutionPolicy Bypass -File ./scripts/auth_smoke_test.ps1
```

## Testing

Backend tests:

```powershell
go test ./...
```

Frontend lint/build:

```powershell
cd frontend
npm run lint
npm run build
```

## Operational Defaults

- HTTP server read header timeout: 5 seconds
- Graceful shutdown timeout: 10 seconds
- Request body limit middleware: 1 MB
- In-memory auth rate limit: 20 requests/minute per IP and auth route
- Login lock escalation default threshold: 5 failed attempts
- Login lock escalation stricter threshold: 3 failed attempts when login IP changes

Source references:

- `internal/app/app.go`
- `internal/delivery/httpserver/security_middleware.go`
- `internal/usecase/auth/service.go`

## Security Notes

Implemented protections include:

- Password hashing using bcrypt
- JWT access token validation
- Refresh token hashing in storage
- Access token blacklist support
- Login failed-attempt tracking with escalating lock windows
- In-memory auth endpoint rate limiting
- Body size limit middleware
- CORS origin allowlist
- HTTP hardening headers (CSP, frame deny, no-sniff, etc.)
- Panic recovery middleware

Recommended production hardening:

- Use a strong, rotated `JWT_SECRET`
- Run Postgres with TLS and least-privilege credentials
- Put API behind HTTPS-only reverse proxy
- Replace in-memory rate limiter with distributed store in multi-instance deployments
- Wire `SECURITY_ALERT_WEBHOOK_URL` to monitored incident channel

## Roadmap

- Add CI workflow and wire real build/lint/test badges
- Add OpenAPI spec generation and publish API docs
- Add structured logging + request correlation IDs
- Add Redis-backed distributed rate limiting
- Add email verification and password reset flow
- Add role-based access control for future service domains

## Developer

- Name: PÄ±nar Topuz
- Signature: CodeByPinar
- Role: Full-stack developer
- Project focus: Secure authentication architecture, robust backend workflows, and premium frontend developer experience

## License

No explicit license file is currently defined in this repository.

If this project will be shared publicly, add a `LICENSE` file (for example MIT, Apache-2.0, or GPL-3.0) and update the badge accordingly.

## Troubleshooting

Common issues:

- `DB_URL is required`: Export environment variables before running the API.
- `origin not allowed` errors: Update `CORS_ALLOWED_ORIGINS` to include frontend URL.
- `invalid credentials` on login: Check seeded or newly registered account email/password pair.
- `invalid refresh token`: Token may already be rotated or revoked (expected after logout or refresh reuse).
- frontend cannot reach backend: Verify backend is listening on the same base URL configured by `VITE_API_BASE_URL`.

---

Built for Eventra auth iteration with a strong security baseline and a practical developer UX.
