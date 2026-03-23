# Resilience Patterns

This document describes the resilience patterns implemented in the Go Boilerplate application.

## Overview

The application implements several resilience patterns to ensure reliability and fault tolerance in production environments.

## 1. Request Timeout Middleware

### Purpose
Prevents long-running requests from consuming resources indefinitely.

### Implementation
Located in `internal/http/middleware/timeout.go`

**Configuration:**
- Default timeout: 30 seconds
- Applied globally to all routes
- Returns HTTP 408 (Request Timeout) when exceeded

**Usage:**
```go
r.Use(httpmiddleware.Timeout(30 * time.Second))
```

**Behavior:**
- Enforces maximum request duration
- Returns JSON error response on timeout
- Prevents goroutine leaks

## 2. Database Connection Pool

### Purpose
Optimizes database connections for performance and reliability.

### Configuration
Configured in `cmd/api/main.go`

**Settings:**
```go
poolConfig.MaxConns = 25              // Maximum connections
poolConfig.MinConns = 5               // Minimum idle connections
poolConfig.MaxConnLifetime = time.Hour // Connection lifetime
poolConfig.MaxConnIdleTime = 30 * time.Minute // Idle timeout
poolConfig.HealthCheckPeriod = time.Minute // Health check interval
```

**Benefits:**
- Prevents connection exhaustion
- Maintains optimal connection pool size
- Automatically removes unhealthy connections
- Balances performance and resource usage

**Monitoring:**
Connection pool stats are logged on startup:
```
Database connection pool configured max_conns=25 min_conns=5 max_conn_lifetime=1h0m0s
```

## 3. Circuit Breaker for Redis

### Purpose
Prevents cascading failures when Redis is unavailable or slow.

### Implementation
Located in `internal/repository/circuit_breaker.go`

**Two Circuit Breakers:**
1. **CircuitBreakerRefreshTokenRepository** - For refresh token operations
2. **CircuitBreakerRateLimitRepository** - For rate limiting operations

**Circuit Breaker Settings:**
```go
Name:        "Redis" / "RedisRateLimit"
MaxRequests: 5            // Requests allowed in half-open state
Interval:    60 seconds   // Stats reset interval
Timeout:     30 seconds   // Time before retrying in open state
ReadyToTrip: func(counts) {
    failureRatio >= 0.6 && requests >= 3
}
```

**States:**
- **Closed** - Normal operation, all requests pass through
- **Open** - Circuit tripped, requests fail fast without calling Redis
- **Half-Open** - Testing if Redis recovered, limited requests allowed

**Failure Detection:**
Circuit opens when:
- At least 3 requests have been made
- 60% or more requests have failed

**Recovery:**
- After 30 seconds in open state, circuit enters half-open
- Allows up to 5 test requests
- If successful, circuit closes
- If failures continue, circuit reopens

**Benefits:**
- Fast failure when Redis is down
- Prevents resource exhaustion
- Automatic recovery detection
- Graceful degradation

### Example Usage

The circuit breaker repositories are automatically used in production:

```go
// Instead of:
refreshTokenRepo := repository.NewRefreshTokenRepository(redisClient)

// Use:
refreshTokenRepo := repository.NewCircuitBreakerRefreshTokenRepository(redisClient)
```

All repository methods are wrapped with circuit breaker protection:
- `Store()` - Store refresh token
- `Get()` - Retrieve refresh token
- `Delete()` - Delete refresh token
- `DeleteAllForUser()` - Delete all user tokens

## Monitoring and Observability

### Logging
All resilience features include structured logging:

**Database Pool:**
```
INFO Database connection pool configured max_conns=25 min_conns=5
```

**Circuit Breaker:**
```
INFO Circuit breakers enabled for Redis repositories
```

**Request Timeout:**
```
WARN Client error method=GET path=/api/v1/endpoint status=408 duration_ms=30000
```

### Metrics to Track
Consider monitoring these metrics in production:

1. **Database Pool:**
   - Active connections
   - Idle connections
   - Wait count/duration
   - Max connections reached

2. **Circuit Breaker:**
   - State transitions (closed → open → half-open)
   - Failed requests
   - Successful requests
   - Timeout events

3. **Request Timeouts:**
   - Number of timed-out requests
   - Average request duration
   - 95th percentile duration

## Best Practices

### 1. Timeout Configuration
- Set timeouts slightly higher than expected 95th percentile response time
- Consider end-to-end timeout (including downstream services)
- Document timeout values in API documentation

### 2. Connection Pool Sizing
- **MaxConns**: Start with 2x CPU cores, adjust based on load
- **MinConns**: 20-25% of MaxConns for quick response
- **MaxConnLifetime**: Balance between connection refresh and overhead
- **HealthCheckPeriod**: 1-2 minutes is usually sufficient

### 3. Circuit Breaker Tuning
- **Failure threshold**: Balance sensitivity vs false positives
- **Timeout**: Allow enough time for recovery
- **MaxRequests**: Low enough to prevent overwhelming, high enough to confirm recovery

### 4. Fallback Strategies
When circuit is open, consider:
- Return cached data (if available)
- Return degraded functionality
- Return meaningful error messages
- Queue operations for retry (if applicable)

## Troubleshooting

### Database Connection Pool Issues

**Symptom:** "connection pool exhausted"
**Solution:** Increase MaxConns or optimize queries

**Symptom:** Slow response times under load
**Solution:** Check pool stats, may need more MinConns

### Circuit Breaker Issues

**Symptom:** Frequent circuit opens
**Solution:** Check Redis health, network connectivity, or increase failure threshold

**Symptom:** Circuit stays open too long
**Solution:** Decrease Timeout value for faster recovery attempts

### Request Timeout Issues

**Symptom:** Legitimate requests timing out
**Solution:** Increase timeout or optimize slow endpoints

**Symptom:** No timeouts logged
**Solution:** Verify middleware is applied, check middleware order

## Configuration Summary

| Feature | Setting | Value | Location |
|---------|---------|-------|----------|
| Request Timeout | Duration | 30s | router.go |
| DB Max Connections | MaxConns | 25 | main.go |
| DB Min Connections | MinConns | 5 | main.go |
| DB Connection Lifetime | MaxConnLifetime | 1h | main.go |
| DB Idle Timeout | MaxConnIdleTime | 30m | main.go |
| DB Health Check | HealthCheckPeriod | 1m | main.go |
| CB Max Requests | MaxRequests | 5 | circuit_breaker.go |
| CB Interval | Interval | 60s | circuit_breaker.go |
| CB Timeout | Timeout | 30s | circuit_breaker.go |
| CB Failure Threshold | ReadyToTrip | 60% | circuit_breaker.go |

## Future Improvements

Potential enhancements:
1. **Retry Logic** - Implement exponential backoff for transient failures
2. **Bulkhead Pattern** - Isolate critical resources
3. **Rate Limiting** - Per-user or per-endpoint limits
4. **Health Checks** - Detailed health check endpoints
5. **Metrics Export** - Prometheus metrics for monitoring
6. **Adaptive Timeout** - Dynamically adjust timeouts based on performance
