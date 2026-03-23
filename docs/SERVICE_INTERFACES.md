# Service Interfaces

This document describes the service interfaces and their usage in the application.

## Overview

All service packages define interfaces that are implemented by concrete types. This allows for:
- Easy mocking in tests
- Clear API contracts
- Loose coupling between components

## Available Interfaces

### Authenticator (`internal/service`)

Handles user authentication and token management.

```go
type Authenticator interface {
    Register(ctx context.Context, input RegisterInput) (*AuthResult, error)
    Login(ctx context.Context, input LoginInput) (*AuthResult, error)
    Refresh(ctx context.Context, refreshToken string) (*AuthTokens, error)
    Logout(ctx context.Context, refreshToken string) error
    LogoutAll(ctx context.Context, userID uuid.UUID) error
}
```

### NoteManager (`internal/service`)

Manages note operations.

```go
type NoteManager interface {
    GetByID(ctx context.Context, id uuid.UUID) (*domain.Note, error)
    GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*domain.Note, error)
    ListByUserID(ctx context.Context, userID uuid.UUID, opts domain.NoteListOptions) (*domain.PaginatedResult[domain.Note], error)
    Create(ctx context.Context, input CreateNoteInput) (*domain.Note, error)
    Update(ctx context.Context, input UpdateNoteInput) (*domain.Note, error)
    SoftDelete(ctx context.Context, id, userID uuid.UUID) error
    Restore(ctx context.Context, id, userID uuid.UUID) error
}
```

### UserManager (`internal/service`)

Manages user operations.

```go
type UserManager interface {
    GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
    GetByEmail(ctx context.Context, email string) (*domain.User, error)
    List(ctx context.Context, opts domain.ListOptions) (*domain.PaginatedResult[domain.User], error)
    Delete(ctx context.Context, id uuid.UUID) error
    UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (*domain.User, error)
}
```

## Compile-Time Verification

All services include compile-time interface checks to ensure they implement their interfaces:

```go
var _ Authenticator = (*AuthService)(nil)
var _ NoteManager = (*NoteService)(nil)
var _ UserManager = (*UserService)(nil)
```

## Using Mocks

Mocks are auto-generated using `mockgen`. To regenerate mocks:

```bash
make mocks
```

Mocks are available at:
- `internal/service/mocks` - Service interface mocks
- `internal/repository/mocks` - Repository interface mocks

### Example: Using Mocks in Tests

```go
func TestMyHandler(t *testing.T) {
    mockAuth := new(mocks.MockAuthenticator)
    
    mockAuth.On("Login", ctx, mock.Anything).
        Return(&service.AuthResult{...}, nil)
    
    handler := NewHandler(mockAuth)
    // Test your handler
}
```

## Repository Interfaces

Repository interfaces are defined in `internal/repository/interfaces.go`:

- `UserRepository` - User data access
- `NoteRepository` - Note data access  
- `RefreshTokenRepository` - Token storage
- `RateLimitRepository` - Rate limiting

## Best Practices

1. **Depend on interfaces, not implementations** - Accept interfaces as function parameters
2. **Mock in tests** - Use generated mocks for unit testing
3. **Keep interfaces small** - Each interface should have a single responsibility
4. **Document interfaces** - Add godoc comments to explain purpose and usage
