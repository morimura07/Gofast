# CONTEXT.md

This file provides an overview of the GoFast template, including architecture, commands, and development workflow.

## Project Overview

GoFast is a full-stack template combining:
- **Go 1.25 backend** (ConnectRPC + PostgreSQL + SQLC)
- **SvelteKit frontend** (TypeScript + Tailwind CSS + DaisyUI)
- **PostgreSQL 18** database with Goose migrations
- **Observability stack** (OTEL Alloy, Prometheus, Tempo, Loki, Grafana)

## Template Foundation for GoFast CLI

This repository serves as the **reference template** for the gofast-cli (`gof`) - a code generation tool that builds Go applications like Lego blocks.

### How It Works

1. **`gof init myproject`** - Downloads this template and creates a new project
2. **`gof model note title:string`** - Generates CRUD models using skeleton templates
3. **`gof add stripe`** - Enables optional integrations by preserving marker blocks

### GF_* Marker System

Throughout this codebase you'll find `GF_*_START` / `GF_*_END` comment markers. These are **not dead code** - they're injection points and toggles used by the CLI:

| Marker Type | Purpose |
|-------------|---------|
| `GF_STRIPE_START/END` | Optional Stripe integration - stripped if not enabled |
| `GF_TP_IMPORT_SERVICES_START/END` | Injection point for service imports |
| `GF_TP_HANDLER_FIELDS_START/END` | Injection point for handler struct fields |
| `GF_TP_ROUTES_START/END` | Injection point for route registration |
| `GF_MAIN_INIT_SERVICES_START/END` | Injection point for service instantiation |
| `GF_ACCESS_FLAGS_START/END` | Injection point for permission flags |
| `GF_FIXTURES_START/END` | Generated test fixtures |

### Skeleton Templates

The CLI copies and transforms skeleton directories for new models:
- `app/service-core/domain/skeleton/` → `domain/{model}/`
- `app/service-core/transport/skeleton/` → `transport/{model}/`
- `app/service-client/src/routes/(app)/models/skeletons/` → `models/{model}/`

Token replacements: `skeleton` → `note`, `Skeleton` → `Note`, `skeletons` → `notes`, etc.

### Integration System

Optional features (Stripe, analytics, etc.) are wrapped in markers. The CLI:
- **On init:** Strips ALL integration markers not in `gofast.json`
- **On `gof add`:** Preserves that integration's markers, strips others

This means gofast-app contains the **complete superset** of all features, and the CLI subtracts what isn't needed.

## Build & Development Commands

### Docker Compose
```bash
make start      # Core service only
make startc     # Core + client
make startm     # Core + client + monitoring
make stop       # Stop containers
make down       # Remove containers
```

### Code Generation
```bash
make gen        # buf generate (protobuf → Go + TypeScript)
make sql        # sqlc generate (SQL → type-safe Go)
make migrate    # Run Goose migrations
make keys       # Generate Ed25519 JWT keys
```

### Testing
```bash
./scripts/run_tests.sh                      # Full test suite (Go + frontend lint + E2E)
cd app && go test ./... -race               # Go pkg unit tests only
cd app/service-core && go test ./... -race  # Go core unit tests only
cd e2e && npm run test                      # E2E tests only
cd e2e && npm run test:headed               # E2E with browser UI
```

### Frontend
```bash
cd app/service-client
npm run dev     # Dev server (localhost:3000)
npm run check   # Type checking
npm run lint    # ESLint + Prettier
```

### Backend
```bash
cd app/service-core
go build ./...      # Build
go test -race ./... # Tests with race detection
golangci-lint run   # Linting
```

## Architecture

### Code Generation Flow
```
proto/v1/*.proto → (buf generate) → app/gen/ (Go) + service-client/src/lib/gen/ (TS)
storage/query.sql → (sqlc generate) → storage/query/*.sql.go
```

Generated code is read-only. Comment markers (`// GF_*_START/END`) indicate generation points.

### Service Pattern
Each feature follows domain-driven structure:
- `domain/[feature]/service.go` - Business logic with `Deps` struct for dependency injection
- `domain/[feature]/validation.go` - Input validation
- `transport/[feature]/route.go` - ConnectRPC handlers

### Database Layer
- Migrations: `app/service-core/storage/migrations/` (Goose format)
- Queries: `app/service-core/storage/query.sql` → generates `storage/query/`
- Pattern: UUID primary keys, user-scoped data access

### Authentication
- Ed25519 JWT keys: `app/private.pem` + `app/public.pem`
- Context-based auth: `auth.Authorize(ctx, span, permission)`
- OAuth flows: Google, GitHub, Facebook, Microsoft

### Error Handling
- Never use inline errors (`if err := ...; err != nil`). Always split into two lines.
- Custom errors (`ForbiddenError`, `NotFoundError`, `InternalError` from `pkg/error.go`) are only used inside service layer (`domain/*/service.go`)
- All other layers wrap errors using `fmt.Errorf` or `errors.New`

### Authentication Flow

**OAuth Login Flow:**
1. Client calls `LoginURL` → `Login()` generates OAuth URL with state/verifier stored in DB
2. User authenticates with provider (Google, GitHub, etc.)
3. Provider redirects to `/login/callback` → `Callback()` exchanges code for token
4. Fetches user info from provider, creates user if new
5. If Twilio configured and user has phone → sends 2FA code, returns `session_token`
6. If no 2FA → `createAuthTokens()` generates JWT access/refresh tokens, stores refresh in DB

**Request Authentication (AuthInterceptor):**
1. `WrapUnary` intercepts all ConnectRPC requests
2. Checks dev mode bypass (`DEV_USER_ID` env var)
3. Skips public routes (e.g., `/proto.v1.LoginService/LoginURL`)
4. Extracts `access_token` and `refresh_token` from cookies
5. Calls `authenticate()` → `loginSvc.Refresh()`:
   - If access token valid → returns existing tokens
   - If access token invalid → validates refresh token from DB, generates new tokens
6. Sets claims in context via `auth.NewContextWithUser(ctx, claims)`
7. If tokens refreshed (`Fresh: true`) → sets new cookies in response

**Client Auth Check (`+layout.svelte`):**
1. On protected route load, calls `login_client.refresh({})`
2. If success → stores user in `store.user`, renders page
3. If fail → redirects to `/login`

**Access Control (`pkg/auth/auth.go`):**
- Bitwise flags: `BasicPlan`, `ProPlan`, `GetSkeletons`, `CreateSkeleton`, etc.
- `UserAccess` constant = default permissions for all users
- `HasAccess(required, userAccess)` → checks if user has permission
- `Authorize(ctx, span, requiredAccess)` → gets claims from context, checks access
- Services call `auth.Authorize()` at start of protected operations

**Subscription Change → Token Refresh (lazy):**
1. Stripe webhook updates subscription in DB (no token refresh)
2. On next request, `Refresh()` calls `CheckUserAccess()` → queries DB for subscription
3. Compares JWT claims vs DB state → if mismatch, issues fresh tokens with updated access bits
4. `AuthInterceptor` sets new cookies when `Fresh: true`

## Key Directories

| Path | Purpose |
|------|---------|
| `app/service-core/domain/` | Business logic by feature |
| `app/service-core/transport/` | HTTP/RPC handlers |
| `app/service-core/storage/` | Migrations, queries, SQLC |
| `app/service-client/src/routes/` | SvelteKit pages |
| `proto/v1/` | Protobuf definitions |
| `e2e/` | Playwright E2E tests |
