package testutil

import (
	"context"
	"sync"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	redistest "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

type RedisContainer struct {
	*redistest.RedisContainer
	ConnectionString string
}

func SetupRedis(t *testing.T) *RedisContainer {
	t.Helper()
	ctx := context.Background()

	container, err := redistest.Run(ctx,
		"redis:7-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithOccurrence(1).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start container: %s", err)
	}

	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get connection string: %s", err)
	}

	return &RedisContainer{
		RedisContainer:   container,
		ConnectionString: connStr,
	}
}

func CreateRedisClient(t *testing.T, connStr string) *goredis.Client {
	t.Helper()

	opts, err := goredis.ParseURL(connStr)
	if err != nil {
		t.Fatalf("failed to parse redis URL: %s", err)
	}

	client := goredis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("failed to ping redis: %s", err)
	}

	return client
}

func CleanupRedis(t *testing.T, container *RedisContainer, client *goredis.Client) {
	t.Helper()

	if client != nil {
		if err := client.Close(); err != nil {
			t.Logf("failed to close redis client: %s", err)
		}
	}

	if container != nil {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}
}

func FlushRedis(t *testing.T, client *goredis.Client) {
	t.Helper()

	ctx := context.Background()
	if err := client.FlushAll(ctx).Err(); err != nil {
		t.Fatalf("failed to flush redis: %s", err)
	}
}

var sharedRedis struct {
	once      sync.Once
	container *RedisContainer
	url       string
	err       error
}

// NewTestRedis returns a client for a shared Redis container, flushed before use.
// Tests in a package run sequentially, so a flush gives each test a clean slate.
func NewTestRedis(t *testing.T) *goredis.Client {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test: requires Docker (run without -short)")
	}

	sharedRedis.once.Do(func() {
		ctx := context.Background()
		container, err := redistest.Run(ctx,
			"redis:7-alpine",
			testcontainers.WithWaitStrategy(
				wait.ForLog("Ready to accept connections").
					WithOccurrence(1).
					WithStartupTimeout(60*time.Second)),
		)
		if err != nil {
			sharedRedis.err = err
			return
		}
		sharedRedis.url, sharedRedis.err = container.ConnectionString(ctx)
		sharedRedis.container = &RedisContainer{RedisContainer: container, ConnectionString: sharedRedis.url}
	})
	if sharedRedis.err != nil {
		t.Fatalf("failed to start shared redis: %v", sharedRedis.err)
	}

	client := CreateRedisClient(t, sharedRedis.url)
	FlushRedis(t, client)
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func StopSharedRedis() {
	if sharedRedis.container != nil {
		_ = sharedRedis.container.Terminate(context.Background())
	}
}
