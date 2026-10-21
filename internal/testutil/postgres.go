package testutil

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx database/sql driver for goose
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type PostgresContainer struct {
	*postgres.PostgresContainer
	ConnectionString string
}

func SetupPostgres(t *testing.T) *PostgresContainer {
	t.Helper()
	container, err := startPostgres(context.Background())
	if err != nil {
		t.Fatalf("failed to start container: %s", err)
	}
	return container
}

func startPostgres(ctx context.Context) (*PostgresContainer, error) {
	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		return nil, err
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, err
	}

	return &PostgresContainer{
		PostgresContainer: container,
		ConnectionString:  connStr,
	}, nil
}

func RunMigrations(t *testing.T, connStr string) {
	t.Helper()
	if err := migrate(connStr); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}
}

func migrate(connStr string) error {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("cannot locate migrations directory")
	}
	migrationsDir := filepath.Join(filepath.Dir(filename), "..", "..", "migrations")

	db, err := goose.OpenDBWithDriver("postgres", connStr)
	if err != nil {
		return fmt.Errorf("open DB: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, migrationsDir)
}

const templateDatabase = "gebeya_template"

var shared struct {
	once      sync.Once
	container *PostgresContainer
	admin     *pgxpool.Pool
	err       error
	counter   atomic.Int64
}

func withDatabase(connStr, name string) (string, error) {
	u, err := url.Parse(connStr)
	if err != nil {
		return "", err
	}
	u.Path = "/" + name
	return u.String(), nil
}

func startShared() {
	ctx := context.Background()
	shared.container, shared.err = startPostgres(ctx)
	if shared.err != nil {
		return
	}

	shared.admin, shared.err = pgxpool.New(ctx, shared.container.ConnectionString)
	if shared.err != nil {
		return
	}

	if _, shared.err = shared.admin.Exec(ctx, "CREATE DATABASE "+templateDatabase); shared.err != nil {
		return
	}

	templateURL, err := withDatabase(shared.container.ConnectionString, templateDatabase)
	if err != nil {
		shared.err = err
		return
	}
	shared.err = migrate(templateURL)
}

// NewTestDatabase returns a pool for a fresh, fully migrated database. One
// Postgres container is shared per test binary and every test gets its own
// database cloned from a migrated template, so tests are isolated and fast.
// Integration tests are skipped under -short.
func NewTestDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test: requires Docker (run without -short)")
	}

	shared.once.Do(startShared)
	if shared.err != nil {
		t.Fatalf("failed to start shared postgres: %v", shared.err)
	}

	ctx := context.Background()
	name := fmt.Sprintf("test_%d_%d", os.Getpid(), shared.counter.Add(1))
	if _, err := shared.admin.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", name, templateDatabase)); err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	connStr, err := withDatabase(shared.container.ConnectionString, name)
	if err != nil {
		t.Fatalf("failed to build test database URL: %v", err)
	}

	pool := CreatePool(t, connStr)
	t.Cleanup(func() {
		pool.Close()
		if _, err := shared.admin.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", name)); err != nil {
			t.Logf("failed to drop test database %s: %v", name, err)
		}
	})
	return pool
}

// StopSharedPostgres terminates the shared container; call it from TestMain.
func StopSharedPostgres() {
	if shared.admin != nil {
		shared.admin.Close()
	}
	if shared.container != nil {
		_ = shared.container.Terminate(context.Background())
	}
}

func CreatePool(t *testing.T, connStr string) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		t.Fatalf("failed to parse connection string: %s", err)
	}

	config.MaxConns = 5
	config.MinConns = 1
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("failed to create connection pool: %s", err)
	}

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("failed to ping database: %s", err)
	}

	return pool
}

func CleanupPostgres(t *testing.T, container *PostgresContainer, pool *pgxpool.Pool) {
	t.Helper()

	if pool != nil {
		pool.Close()
	}

	if container != nil {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}
}

func TruncateTables(t *testing.T, db *pgxpool.Pool, tables ...string) {
	t.Helper()

	ctx := context.Background()
	for _, table := range tables {
		query := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
		if _, err := db.Exec(ctx, query); err != nil {
			t.Fatalf("failed to truncate table %s: %s", table, err)
		}
	}
}
