package testutil

import (
	"context"
	"testing"
	"time"

	redisv8 "github.com/go-redis/redis/v8"
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

func CreateRedisClient(t *testing.T, connStr string) *redisv8.Client {
	t.Helper()

	opts, err := redisv8.ParseURL(connStr)
	if err != nil {
		t.Fatalf("failed to parse redis URL: %s", err)
	}

	client := redisv8.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("failed to ping redis: %s", err)
	}

	return client
}

func CleanupRedis(t *testing.T, container *RedisContainer, client *redisv8.Client) {
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

func FlushRedis(t *testing.T, client *redisv8.Client) {
	t.Helper()

	ctx := context.Background()
	if err := client.FlushAll(ctx).Err(); err != nil {
		t.Fatalf("failed to flush redis: %s", err)
	}
}
