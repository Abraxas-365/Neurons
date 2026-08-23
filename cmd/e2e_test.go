package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

// TestMain boots real Postgres and Redis containers via testcontainers-go,
// applies the SQL migrations and points the whole app at them through the
// same DB_*/REDIS_* environment variables the container.go/config package
// already reads. This lets TestFullTeacherStudentJourney and friends in
// api_test.go exercise the real routing, auth, services and SQL against a
// throwaway database instead of relying on `make up` being run beforehand.
//
// Set NEURONS_E2E_USE_LOCAL_INFRA=1 to skip container startup and reuse
// whatever postgres/redis are already reachable (e.g. `make up`).
func TestMain(m *testing.M) {
	if os.Getenv("NEURONS_E2E_USE_LOCAL_INFRA") == "1" {
		os.Exit(m.Run())
	}

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("neuronsdb"),
		postgres.WithUsername("neurons"),
		postgres.WithPassword("supersecret"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		fatal("start postgres container", err)
	}
	defer terminate(pgContainer)

	redisContainer, err := redis.Run(ctx, "redis:7-alpine")
	if err != nil {
		fatal("start redis container", err)
	}
	defer terminate(redisContainer)

	pgHost, pgPort := hostAndPort(ctx, pgContainer.Container, "5432/tcp")
	redisHost, redisPort := hostAndPort(ctx, redisContainer.Container, "6379/tcp")

	os.Setenv("DB_HOST", pgHost)
	os.Setenv("DB_PORT", pgPort)
	os.Setenv("DB_USER", "neurons")
	os.Setenv("DB_PASSWORD", "supersecret")
	os.Setenv("DB_NAME", "neuronsdb")
	os.Setenv("DB_SSL_MODE", "disable")

	os.Setenv("REDIS_HOST", redisHost)
	os.Setenv("REDIS_PORT", redisPort)

	if os.Getenv("JWT_SECRET_KEY") == "" {
		os.Setenv("JWT_SECRET_KEY", "dev-only-insecure-secret-change-me")
	}

	if err := applyMigrations(ctx, pgHost, pgPort); err != nil {
		fatal("apply migrations", err)
	}

	os.Exit(m.Run())
}

func hostAndPort(ctx context.Context, c testcontainers.Container, exposedPort string) (string, string) {
	host, err := c.Host(ctx)
	if err != nil {
		fatal("resolve container host", err)
	}
	mapped, err := c.MappedPort(ctx, exposedPort)
	if err != nil {
		fatal("resolve mapped port", err)
	}
	return host, mapped.Port()
}

func terminate(c testcontainers.Container) {
	if err := testcontainers.TerminateContainer(c); err != nil {
		fmt.Fprintf(os.Stderr, "e2e teardown: failed to terminate container: %v\n", err)
	}
}

// applyMigrations runs every migrations/*.up.sql file in order against the
// freshly created database, mirroring what `make migrate` does against the
// docker-compose database.
func applyMigrations(ctx context.Context, host, port string) error {
	dsn := "postgres://neurons:supersecret@" + host + ":" + port + "/neuronsdb?sslmode=disable"

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	files, err := filepath.Glob(filepath.Join("..", "migrations", "*.up.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)

	for _, f := range files {
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
	}
	return nil
}

func fatal(step string, err error) {
	fmt.Fprintf(os.Stderr, "e2e setup: %s: %v\n", step, err)
	os.Exit(1)
}
