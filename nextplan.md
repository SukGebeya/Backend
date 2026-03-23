# Production Readiness Implementation Plan

## Overview

This plan outlines the steps to make this Go boilerplate production-ready, including critical fixes, API documentation, testing infrastructure, observability, CI/CD, and resilience patterns.

**Test Coverage Target:** 70-80%
**Migration Naming:** `.updown.sql` (combined files)

---

## Phase 1: Critical Fixes (30 min) ✅ COMPLETED

| Task | Files | Details | Status |
|------|-------|---------|--------|
| Fix migration path in main.go | `cmd/api/main.go:134` | Change `db/migrations` → `migrations` | ✅ |
| Fix migration path in Dockerfile | `Dockerfile:21` | Change `db/migrations` → `migrations` | ✅ |
| Remove dead code | `cmd/api/main.go:141-143` | Remove `parseRedisAddr` function | ✅ |

---

## Phase 2: API Documentation - PRIORITY (1-2 days) ✅ COMPLETED

### 2.1 Setup Swagger Dependencies ✅
- Add `github.com/swaggo/swag`
- Add `github.com/swaggo/http-swagger`

### 2.2 Add Swagger Annotations ✅

| File | Endpoints to Document | Status |
|------|----------------------|--------|
| `auth_handler.go` | Register, Login, Refresh, Logout, LogoutAll | ✅ |
| `user_handler.go` | GetMe, UpdateProfile, List, Get, Delete | ✅ |
| `note_handler.go` | List, Create, Get, Update, Delete, Restore | ✅ |
| `health_handler.go` | Health | ✅ |

### 2.3 Swagger Configuration ✅
- Created `docs/` package with generated swagger files
- Added swagger route to router at `/swagger/*`
- Configured swagger with proper security schemes (Bearer JWT)

### 2.4 Update Makefile ✅

```makefile
swagger: ## Generate swagger documentation
	swag init -g cmd/api/main.go -o ./docs --parseInternal --parseDependency --parseDepth 2
```

### 2.5 Additional Improvements ✅
- Added enhanced health check handler with DB/Redis connectivity checks
- Added user profile endpoints (GET/PUT /users/me)
- Added admin endpoint to get user by ID (GET /users/{id})
- Added `UpdatedAt` field to UserResponse

---

## Phase 3: Testing Infrastructure (2-3 days)

### 3.1 Test Dependencies

```
github.com/testcontainers/testcontainers-go
github.com/testcontainers/testcontainers-go/modules/postgres
github.com/testcontainers/testcontainers-go/modules/redis
github.com/stretchr/testify
github.com/stretchr/testify/mock
```

### 3.2 Test Utilities

Create `internal/testutil/`:

| File | Purpose |
|------|---------|
| `postgres.go` | testcontainer PostgreSQL setup, migrations |
| `redis.go` | testcontainer Redis setup |
| `fixtures.go` | Test user, note builders |
| `context.go` | Context helpers with user IDs |

### 3.3 Unit Tests (Target: 70-80% coverage)

| Package | Test File | Priority |
|---------|-----------|----------|
| `pkg/password` | `argon2id_test.go` | High |
| `pkg/jwt` | `jwt_test.go`, `keys_test.go` | High |
| `pkg/validation` | `validation_test.go` | Medium |
| `pkg/response` | `response_test.go` | Low |
| `internal/errors` | `errors_test.go` | Medium |

### 3.4 Service Interfaces & Mocks

- Create `internal/service/interfaces.go`
- Define `Authenticator`, `NoteManager`, `UserManager` interfaces
- Generate mocks in `internal/service/mocks/`

### 3.5 Service Tests (with mocks)

| File | Coverage Target |
|------|-----------------|
| `auth_service_test.go` | Register, Login, Refresh, Logout flows |
| `note_service_test.go` | CRUD + soft delete/restore |
| `user_service_test.go` | List, Delete |

### 3.6 Repository Integration Tests (testcontainers)

| File | Coverage Target |
|------|-----------------|
| `user_repository_test.go` | All 9 methods |
| `note_repository_test.go` | All 10 methods |
| `redis_test.go` | Token storage, rate limiting |

### 3.7 Handler Tests (HTTP tests)

| File | Coverage Target |
|------|-----------------|
| `auth_handler_test.go` | All 5 endpoints |
| `note_handler_test.go` | All 6 endpoints |
| `user_handler_test.go` | Admin endpoints |
| `health_handler_test.go` | Health check |

---

## Phase 4: Observability (1 day)

### 4.1 Structured Logging with slog

| File | Purpose |
|------|---------|
| `pkg/logger/logger.go` | Configured slog with JSON/text output |
| `pkg/logger/middleware.go` | HTTP request logging middleware |

### 4.2 Enhanced Health Check

| File | Purpose |
|------|---------|
| `internal/http/handler/health_handler.go` | Check DB + Redis connectivity |

Response format:

```json
{
  "status": "healthy",
  "checks": {
    "database": "ok",
    "redis": "ok"
  }
}
```

---

## Phase 5: CI/CD (1 day)

### 5.1 GitHub Actions Workflow

Create `.github/workflows/ci.yml`:

| Job | Triggers | Steps |
|-----|----------|-------|
| `lint` | PR, push to main | golangci-lint |
| `test` | PR, push to main | Run tests with coverage, upload to codecov |
| `build` | PR, push to main | Build binary, verify |
| `docker` | push to main | Build Docker image |
| `swagger` | PR, push to main | Verify swagger docs generate |

### 5.2 Dependabot

Create `.github/dependabot.yml` for Go modules and GitHub Actions.

---

## Phase 6: Service Interfaces (1 day)

### 6.1 Define Interfaces

```go
// internal/service/interfaces.go
type Authenticator interface {
    Register(ctx context.Context, input RegisterInput) (*AuthResult, error)
    Login(ctx context.Context, input LoginInput) (*AuthResult, error)
    Refresh(ctx context.Context, refreshToken string) (*AuthTokens, error)
    Logout(ctx context.Context, refreshToken string) error
    LogoutAll(ctx context.Context, userID uuid.UUID) error
}

type NoteManager interface {
    Create(ctx context.Context, userID uuid.UUID, input CreateNoteInput) (*domain.Note, error)
    GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Note, error)
    List(ctx context.Context, userID uuid.UUID, opts domain.NoteListOptions) (*domain.PaginatedResult[domain.Note], error)
    Update(ctx context.Context, id, userID uuid.UUID, input UpdateNoteInput) (*domain.Note, error)
    Delete(ctx context.Context, id, userID uuid.UUID) error
    Restore(ctx context.Context, id, userID uuid.UUID) (*domain.Note, error)
}

type UserManager interface {
    List(ctx context.Context, opts domain.ListOptions) (*domain.PaginatedResult[domain.User], error)
    Delete(ctx context.Context, id uuid.UUID) error
}
```

### 6.2 Generate Mocks

- Use `mockgen` or `moq` to generate mocks
- Add to Makefile

---

## Phase 7: Resilience (1 day)

### 7.1 Request Timeout Middleware

| File | Purpose |
|------|---------|
| `internal/http/middleware/timeout.go` | Enforce max request duration |

### 7.2 Database Connection Pool Configuration

| File | Changes |
|------|---------|
| `cmd/api/main.go` | Configure `MaxConns`, `MinConns`, `MaxConnLifetime`, `HealthCheckPeriod` |

### 7.3 Circuit Breaker for Redis

| File | Purpose |
|------|---------|
| `internal/repository/circuit_breaker.go` | Wrap Redis calls with gobreaker |

---

## Final File Structure

```
├── .github/
│   ├── workflows/
│   │   └── ci.yml
│   └── dependabot.yml
├── cmd/api/main.go                 # Fixed migration path
├── internal/
│   ├── service/
│   │   ├── interfaces.go           # NEW
│   │   ├── mocks/                  # NEW
│   │   │   └── mocks.go
│   │   ├── auth_service.go
│   │   ├── auth_service_test.go    # NEW
│   │   ├── note_service.go
│   │   ├── note_service_test.go    # NEW
│   │   ├── user_service.go
│   │   └── user_service_test.go    # NEW
│   ├── repository/
│   │   ├── interfaces.go
│   │   ├── user_repository.go
│   │   ├── user_repository_test.go # NEW
│   │   ├── note_repository.go
│   │   ├── note_repository_test.go # NEW
│   │   ├── redis.go
│   │   ├── redis_test.go           # NEW
│   │   └── circuit_breaker.go      # NEW (Phase 7)
│   ├── http/
│   │   ├── router.go
│   │   ├── handler/
│   │   │   ├── auth_handler.go
│   │   │   ├── auth_handler_test.go # NEW
│   │   │   ├── note_handler.go
│   │   │   ├── note_handler_test.go # NEW
│   │   │   ├── user_handler.go
│   │   │   ├── user_handler_test.go # NEW
│   │   │   ├── health_handler.go    # NEW
│   │   │   └── health_handler_test.go # NEW
│   │   └── middleware/
│   │       ├── auth.go
│   │       ├── timeout.go           # NEW (Phase 7)
│   │       └── ...
│   └── testutil/                    # NEW
│       ├── postgres.go
│       ├── redis.go
│       ├── fixtures.go
│       └── context.go
├── pkg/
│   ├── logger/                      # NEW
│   │   ├── logger.go
│   │   └── middleware.go
│   ├── password/
│   │   ├── argon2id.go
│   │   └── argon2id_test.go         # NEW
│   ├── jwt/
│   │   ├── jwt.go
│   │   ├── jwt_test.go              # NEW
│   │   ├── keys.go
│   │   └── keys_test.go             # NEW
│   ├── validation/
│   │   ├── validation.go
│   │   └── validation_test.go       # NEW
│   └── response/
│       ├── response.go
│       └── response_test.go         # NEW
├── docs/                            # NEW (Swagger)
│   ├── swagger.json
│   ├── swagger.yaml
│   └── docs.go
├── Dockerfile                       # Fixed migration path
├── Makefile                         # Updated with new commands
└── go.mod                           # New dependencies
```

---

## Timeline Summary

| Phase | Description | Duration | Priority |
|-------|-------------|----------|----------|
| 1 | Critical Fixes | 30 min | Critical |
| 2 | API Documentation | 1-2 days | **High** |
| 3 | Testing Infrastructure | 2-3 days | High |
| 4 | Observability | 1 day | Medium |
| 5 | CI/CD | 1 day | Medium |
| 6 | Service Interfaces | 1 day | Medium |
| 7 | Resilience | 1 day | Low |

**Total: ~8-10 days**

---

## Execution Order

1. **Phase 1** - Critical fixes (blocking issues)
2. **Phase 2** - API Documentation (priority)
3. **Phase 3** - Testing (foundation for reliability)
4. **Phase 4** - Observability (logging + health)
5. **Phase 5** - CI/CD (automation)
6. **Phase 6** - Service Interfaces (enables better testing)
7. **Phase 7** - Resilience (production hardening)

---

## Dependencies to Add

```go
// Testing
github.com/testcontainers/testcontainers-go
github.com/testcontainers/testcontainers-go/modules/postgres
github.com/testcontainers/testcontainers-go/modules/redis
github.com/stretchr/testify

// API Documentation
github.com/swaggo/swag/v2
github.com/swaggo/http-swagger/v2

// Resilience
github.com/sony/gobreaker

// Mocking (choose one)
github.com/uber-go/mock
// OR
github.com/matryer/moq
```

---

## Makefile Commands to Add

```makefile
.PHONY: swagger test-unit test-integration test-coverage mocks

swagger: ## Generate swagger documentation
	swag init -g cmd/api/main.go -o ./docs --packages internal/http/handler

mocks: ## Generate mocks for testing
	mockgen -source=internal/service/interfaces.go -destination=internal/service/mocks/mocks.go

test-unit: ## Run unit tests only
	go test -v -short -coverprofile=coverage.out ./...

test-integration: ## Run integration tests
	go test -v -run Integration ./...

test-coverage: ## Generate HTML coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
```
