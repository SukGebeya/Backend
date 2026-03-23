# Go Boilerplate

A production-ready Go REST API boilerplate with clean architecture, JWT authentication, PostgreSQL, and Redis.

## Features

### Core Features
- **Clean Architecture** - Separation of concerns with domain, service, repository, and HTTP layers
- **JWT Authentication** - RSA-signed access and refresh tokens with automatic rotation
- **PostgreSQL** - Type-safe SQL with sqlc code generation
- **Redis** - Refresh token storage and distributed rate limiting
- **Password Hashing** - Argon2id with configurable parameters
- **Request Validation** - go-playground/validator with custom phone validation
- **Database Migrations** - Goose for version-controlled schema changes
- **Docker Ready** - Multi-stage Dockerfile and docker-compose setup
- **Soft Delete** - Notes support soft delete with restore capability

### Production Readiness ✨

- **API Documentation** - Swagger/OpenAPI documentation for all 18 endpoints
- **Testing Infrastructure** - testcontainers, mocks, and comprehensive unit tests
- **Structured Logging** - JSON and text format logging with slog
- **CI/CD Pipeline** - GitHub Actions with automated linting, testing, and Docker builds
- **Service Interfaces** - Interface-based architecture with compile-time verification
- **Resilience Patterns** - Request timeouts, connection pooling, and circuit breakers
- **Observability** - Request logging, health checks, and password redaction

## Quick Start

```bash
# Clone the repository
git clone https://github.com/nate/go-boilerplate.git
cd go-boilerplate

# Copy environment file
cp .env.example .env

# Generate RSA keys
make generate-keys

# Start services
make docker-up

# Run migrations (if not using docker)
make migrate-up

# Start the server
make run
```

The API will be available at `http://localhost:8080`.

**Swagger Documentation**: `http://localhost:8080/swagger/index.html`

## Prerequisites

- Go 1.24+
- PostgreSQL 16+
- Redis 7+
- Docker & Docker Compose (optional)
- sqlc (for code generation)
- goose (for migrations)

## Project Structure

```
.
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/                  # Configuration management
│   ├── db/                      # Generated sqlc code
│   ├── domain/                  # Domain models (User, Note)
│   │   ├── user.go
│   │   ├── note.go
│   │   └── common.go
│   ├── errors/                  # Custom error types with codes
│   ├── http/
│   │   ├── handler/             # HTTP handlers by domain
│   │   │   ├── auth_handler.go
│   │   │   ├── user_handler.go
│   │   │   ├── note_handler.go
│   │   │   ├── health_handler.go
│   │   │   ├── helpers.go
│   │   │   └── responses.go
│   │   ├── middleware/          # Chi middleware
│   │   │   ├── auth.go
│   │   │   ├── rate_limit.go
│   │   │   ├── timeout.go       # Request timeout middleware
│   │   │   └── common.go
│   │   └── router.go            # Route definitions
│   ├── repository/              # Data access layer
│   │   ├── interfaces.go
│   │   ├── user_repository.go
│   │   ├── note_repository.go
│   │   ├── redis.go
│   │   ├── circuit_breaker.go   # Circuit breaker for Redis
│   │   └── mocks/               # Generated repository mocks
│   ├── service/                 # Business logic layer
│   │   ├── interfaces.go        # Service interfaces
│   │   ├── auth_service.go
│   │   ├── user_service.go
│   │   ├── note_service.go
│   │   ├── mocks/               # Generated service mocks
│   │   └── generate.go          # Mock generation
│   └── testutil/                # Test utilities
│       ├── postgres.go          # PostgreSQL testcontainers
│       ├── redis.go             # Redis testcontainers
│       ├── fixtures.go          # Test data builders
│       └── context.go           # Context helpers
├── pkg/                         # Reusable packages
│   ├── jwt/
│   │   ├── jwt.go               # JWT token management
│   │   ├── keys.go              # RSA key handling
│   │   ├── interfaces.go        # Token manager interfaces
│   │   └── jwt_test.go          # Comprehensive unit tests
│   ├── password/
│   │   ├── argon2id.go          # Password hashing
│   │   ├── interfaces.go        # Hasher interface
│   │   └── argon2id_test.go     # Comprehensive unit tests
│   ├── logger/
│   │   ├── logger.go            # Structured logging with slog
│   │   └── middleware.go        # HTTP request logging
│   ├── response/
│   │   └── response.go          # JSON response utilities
│   └── validation/
│       └── validation.go        # Request validation
├── docs/                        # Documentation
│   ├── SERVICE_INTERFACES.md    # Service interfaces guide
│   ├── RESILIENCE.md            # Resilience patterns guide
│   ├── swagger.json             # Generated Swagger spec
│   ├── swagger.yaml
│   └── docs.go
├── .github/
│   ├── workflows/
│   │   └── ci.yml               # GitHub Actions CI/CD
│   └── dependabot.yml           # Dependency updates
├── migrations/                  # Database migrations
├── sqlc/
│   └── queries/
│       └── queries.sql          # SQL queries for sqlc
├── scripts/
│   └── generate_keys.sh         # RSA key generation script
├── keys/                        # RSA key pairs (generated)
├── Makefile
├── docker-compose.yml
├── docker-compose-dev.yml       # Development with hot reload
├── Dockerfile
├── sqlc.yaml
├── .golangci.yml                # Linter configuration
└── .env.example
```

## Architecture

### Layered Architecture

```
HTTP Layer (Handlers)
       ↓
Service Layer (Business Logic)
       ↓
Repository Layer (Data Access)
       ↓
Database (PostgreSQL/Redis)
```

- **Handlers** - Parse requests, validate input, call services, format responses
- **Services** - Business logic, orchestration, domain rules
- **Repositories** - Database operations, data persistence

### Domain-Driven Structure

Each domain (auth, user, note) has its own:
- Handler file(s)
- Service file with interface definition
- Repository file with interface definition
- Domain model(s)

## API Endpoints

### Public Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check (includes DB + Redis status) |
| POST | `/api/v1/register` | Register new user |
| POST | `/api/v1/login` | Login with email/phone |
| POST | `/api/v1/refresh` | Refresh access token |
| POST | `/api/v1/logout` | Logout (revoke refresh token) |

### Authenticated Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/logout-all` | Logout from all devices |
| GET | `/api/v1/users/me` | Get current user profile |
| PUT | `/api/v1/users/me` | Update current user profile |
| GET | `/api/v1/notes` | List user's notes |
| POST | `/api/v1/notes` | Create note |
| GET | `/api/v1/notes/{id}` | Get note by ID |
| PUT | `/api/v1/notes/{id}` | Update note |
| DELETE | `/api/v1/notes/{id}` | Soft delete note |
| POST | `/api/v1/notes/{id}/restore` | Restore deleted note |

### Admin Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/users` | List all users (paginated) |
| GET | `/api/v1/users/{id}` | Get user by ID |
| DELETE | `/api/v1/users/{id}` | Delete user |

## Authentication Flow

### Registration

```bash
# Register with email
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123"}'

# Register with phone
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{"phone": "+1234567890", "password": "password123"}'
```

### Login

```bash
# Login with email or phone
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"identifier": "user@example.com", "password": "password123"}'
```

### Using Access Token

```bash
curl http://localhost:8080/api/v1/notes \
  -H "Authorization: Bearer <access_token>"
```

### Refreshing Tokens

```bash
curl -X POST http://localhost:8080/api/v1/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "<refresh_token>"}'
```

## Configuration

Environment variables (see `.env.example`):

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `ENVIRONMENT` | Environment | `development` |
| `DATABASE_URL` | PostgreSQL connection string | Required |
| `REDIS_URL` | Redis connection string | Required |
| `JWT_ACCESS_PRIVATE_KEY_PATH` | Access token private key path | `keys/access_private.pem` |
| `JWT_ACCESS_PUBLIC_KEY_PATH` | Access token public key path | `keys/access_public.pem` |
| `JWT_REFRESH_PRIVATE_KEY_PATH` | Refresh token private key path | `keys/refresh_private.pem` |
| `JWT_REFRESH_PUBLIC_KEY_PATH` | Refresh token public key path | `keys/refresh_public.pem` |
| `JWT_ACCESS_TTL` | Access token validity | `15m` |
| `JWT_REFRESH_TTL` | Refresh token validity | `168h` (7 days) |
| `ARGON2_MEMORY` | Argon2 memory (KB) | `65536` |
| `ARGON2_ITERATIONS` | Argon2 iterations | `3` |
| `ARGON2_PARALLELISM` | Argon2 parallelism | `2` |
| `RATE_LIMIT_REQUESTS` | Rate limit requests per window | `100` |
| `RATE_LIMIT_WINDOW` | Rate limit window | `1m` |
| `LOG_FORMAT` | Log format (json/text) | `text` |
| `LOG_LEVEL` | Log level (debug/info/warn/error) | `info` |

## Database Schema

### Users Table

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255),
    phone VARCHAR(20),
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_email_or_phone CHECK (email IS NOT NULL OR phone IS NOT NULL)
);
```

### Notes Table

```sql
CREATE TABLE notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
```

## Development

### Available Make Commands

```bash
# Building
make build              # Build the binary

# Running
make run                # Run the application
make air                # Run with hot reload (requires air)
make dev                # Start services and run app
make local              # Start only DB and Redis for local dev

# Testing
make test               # Run all tests
make test-unit          # Run unit tests only
make test-integration   # Run integration tests
make test-coverage      # Run tests with coverage report

# Code Quality
make lint               # Run golangci-lint
make fmt                # Format code
make vet                # Run go vet
make deps               # Download dependencies

# Code Generation
make sqlc-generate      # Generate sqlc code
make swagger            # Generate swagger documentation
make mocks              # Generate mocks for testing
make generate-keys      # Generate RSA key pairs

# Database
make migrate-up         # Run database migrations
make migrate-down       # Rollback migrations
make migrate-status     # Check migration status
make migrate-create     # Create new migration

# Docker
make docker-dev-up      # Start development services (with hot reload)
make docker-dev-down    # Stop development services
make docker-dev-logs    # View development logs
make docker-prod-up     # Start production services
make docker-prod-down   # Stop production services
make docker-prod-logs   # View production logs
make docker-build       # Build Docker image

# Utilities
make redis-cli          # Access Redis CLI
make psql               # Access PostgreSQL CLI
```

### Adding a New Migration

```bash
make migrate-create
# Enter migration name when prompted
```

### Generating sqlc Code

After modifying `sqlc/queries/queries.sql`:

```bash
make sqlc-generate
```

### Generating RSA Keys

```bash
make generate-keys
# Or manually:
./scripts/generate_keys.sh keys
```

### Generating Mocks

```bash
make mocks
```

## Testing

### Test Structure

- **Unit Tests** - Fast tests without external dependencies (`*_test.go`)
- **Integration Tests** - Tests with real databases using testcontainers
- **Test Utilities** - Shared test helpers in `internal/testutil/`

### Running Tests

```bash
# Run all tests
make test

# Run unit tests only (fast)
make test-unit

# Run integration tests (requires Docker)
make test-integration

# Run with coverage
make test-coverage
```

### Test Coverage

- `pkg/password` - 8 test functions
- `pkg/jwt` - 17 test functions
- Target coverage: 70-80%

### Testcontainers

Integration tests use testcontainers for:
- PostgreSQL - Real database testing
- Redis - Real cache testing
- Automatic cleanup after tests

## API Documentation

### Swagger UI

Access interactive API documentation at:
```
http://localhost:8080/swagger/index.html
```

### Generating Swagger Docs

```bash
make swagger
```

Swagger documentation includes:
- All 18 API endpoints
- Request/response schemas
- Authentication requirements
- Error responses
- Example values

## Logging

### Structured Logging

The application uses Go's `slog` package for structured logging:

**Development (text format)**:
```
level=INFO msg="Request completed" method=GET path=/api/v1/users/me status=200 duration_ms=45
```

**Production (JSON format)**:
```json
{"level":"info","msg":"Request completed","method":"GET","path":"/api/v1/users/me","status":200,"duration_ms":45}
```

### Features

- **Format Selection** - JSON for production, text for development
- **Request ID Tracking** - Every request has a unique ID
- **Password Redaction** - Passwords automatically redacted from logs
- **Performance Metrics** - Request duration in milliseconds
- **Error Logging** - Detailed error information with context

### Log Levels

Set via `LOG_LEVEL` environment variable:
- `debug` - Detailed debugging information
- `info` - General operational information
- `warn` - Warning conditions
- `error` - Error conditions

## Resilience Patterns

### Request Timeout

- 30-second timeout for all requests
- Returns HTTP 408 when exceeded
- Prevents resource exhaustion

### Database Connection Pool

Production-optimized configuration:
- **Max Connections**: 25
- **Min Connections**: 5
- **Connection Lifetime**: 1 hour
- **Idle Timeout**: 30 minutes
- **Health Checks**: Every minute

### Circuit Breaker for Redis

Protects against Redis failures:
- **Failure Threshold**: 60% failure rate
- **Timeout**: 30 seconds before retry
- **States**: Closed → Open → Half-Open
- **Automatic Recovery**: Tests Redis health periodically

See `docs/RESILIENCE.md` for detailed documentation.

## CI/CD Pipeline

### GitHub Actions

Automated pipeline on every push and PR:

**Jobs:**
1. **lint** - Run golangci-lint
2. **test** - Run unit tests with coverage
3. **build** - Build binary and verify
4. **swagger** - Verify swagger docs are current
5. **docker** - Build and push Docker images (main/master only)

### Dependabot

Automated dependency updates:
- Weekly updates for Go modules
- Weekly updates for GitHub Actions
- Grouped updates for better organization

### Code Quality

24+ linters enabled including:
- errcheck, govet, staticcheck
- gocyclo, gocritic, revive
- misspell, godot, whitespace

See `.golangci.yml` for full configuration.

## Service Interfaces

### Interface-Based Architecture

All services implement interfaces for better testability:

```go
type Authenticator interface {
    Register(ctx context.Context, input RegisterInput) (*AuthResult, error)
    Login(ctx context.Context, input LoginInput) (*AuthResult, error)
    // ...
}

type NoteManager interface {
    Create(ctx context.Context, input CreateNoteInput) (*domain.Note, error)
    // ...
}

type UserManager interface {
    GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
    // ...
}
```

### Compile-Time Verification

Services verify interface implementation at compile time:
```go
var _ Authenticator = (*AuthService)(nil)
```

### Using Mocks in Tests

```go
mockAuth := new(mocks.MockAuthenticator)
mockAuth.On("Login", ctx, mock.Anything).
    Return(&service.AuthResult{...}, nil)
```

See `docs/SERVICE_INTERFACES.md` for detailed documentation.

## Request Validation

The validation package uses `go-playground/validator` with custom validators:

### Validation Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `required` | Field is required | `validate:"required"` |
| `email` | Valid email format | `validate:"email"` |
| `phone` | Valid E.164 phone | `validate:"phone"` |
| `min` | Minimum length | `validate:"min=8"` |
| `max` | Maximum length | `validate:"max=255"` |
| `omitempty` | Skip if empty | `validate:"omitempty,email"` |

### Validation Error Response

```json
[
  {
    "field": "Email",
    "message": "Email must be a valid email address"
  },
  {
    "field": "Password",
    "message": "Password must be at least 8 characters"
  }
]
```

## Error Handling

All errors follow a consistent format:

```json
{
  "error": "Bad Request",
  "code": "VALIDATION_ERROR",
  "message": "Validation failed"
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INTERNAL_ERROR` | 500 | Server error |
| `NOT_FOUND` | 404 | Resource not found |
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `BAD_REQUEST` | 400 | Invalid request |
| `CONFLICT` | 409 | Resource conflict |
| `VALIDATION_ERROR` | 400 | Validation failed |
| `RATE_LIMITED` | 429 | Too many requests |
| `INVALID_TOKEN` | 401 | Invalid/expired token |
| `INVALID_CREDENTIALS` | 401 | Wrong email/phone or password |
| `EMAIL_EXISTS` | 409 | Email already registered |

## Rate Limiting

- Redis-backed distributed rate limiting
- Rate limit headers in response:
  - `X-RateLimit-Limit` - Maximum requests per window
  - `X-RateLimit-Remaining` - Remaining requests
  - `X-RateLimit-Reset` - Seconds until window resets

## Deployment

### Docker

#### Development (with hot reload)
```bash
make docker-dev-up
```

#### Production
```bash
make docker-prod-up
```

### Manual Deployment

1. Build the binary:
   ```bash
   make build
   ```

2. Generate keys:
   ```bash
   make generate-keys
   ```

3. Set environment variables

4. Run migrations:
   ```bash
   make migrate-up
   ```

5. Run the binary:
   ```bash
   ./bin/api
   ```

### Health Check

The health endpoint verifies all dependencies:

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "checks": {
    "database": "ok",
    "redis": "ok"
  }
}
```

## Security Considerations

- **RSA JWT Signing** - Uses asymmetric encryption (RS256)
- **Argon2id Password Hashing** - Memory-hard algorithm resistant to GPU attacks
- **Refresh Token Rotation** - Old refresh tokens are revoked on use
- **Rate Limiting** - Prevents brute force attacks
- **CORS** - Configurable allowed origins
- **Soft Delete** - Notes can be recovered if deleted accidentally
- **Request Timeout** - Prevents DoS from slow requests
- **Circuit Breaker** - Prevents cascading failures
- **Password Redaction** - Passwords never appear in logs

## Documentation

- **API Documentation** - `http://localhost:8080/swagger/index.html`
- **Service Interfaces** - `docs/SERVICE_INTERFACES.md`
- **Resilience Patterns** - `docs/RESILIENCE.md`
- **Production Plan** - `nextplan.md`

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linters (`make test lint`)
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) for details.
