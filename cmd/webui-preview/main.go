// Command webui-preview runs all /web/* routes (via transport/web) against the
// real repository on a reusable Postgres testcontainer, so web pages can be
// eyeballed in a browser without the production database or Telegram
// credentials the full app (main.go) requires. Migrations and fixtures.sql are
// applied once and cached as a template snapshot; every start restores it.
// Auth still runs for real: set USERS and JWT_SECRET (e.g. in .env.local) the
// same way main.go requires, unless AUTH_DISABLED is set.
package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"personal/action/auth"
	"personal/gateways/db"
	"personal/transport/web"
)

const (
	containerName = "personal-webui-preview"
	hashLabel     = "personal.webui-preview.hash"
	snapshotName  = "webui_preview_snapshot"
)

//go:embed fixtures.sql
var fixturesSQL string

func main() {
	if err := godotenv.Overload(".env.local"); err != nil {
		log.Println("Error loading .env.local file", err)
	}

	authDisabled := os.Getenv("AUTH_DISABLED") != ""
	if !authDisabled {
		auth.InitializeAuth()
	}

	ctx := context.Background()
	conn, err := startDatabase(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)

	repo, _ := db.NewRepository(conn)

	gin.SetMode(gin.DebugMode)
	router := gin.Default()

	web.Register(router, repo, authDisabled)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("webui preview running at http://localhost:%s/web/design-system", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

// startDatabase reuses (or creates) the preview container, restores the
// cached migrations + fixtures snapshot (building it first when missing), and
// returns a connection to the restored database.
func startDatabase(ctx context.Context) (*pgx.Conn, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	if os.Getenv("DOCKER_HOST") == "" {
		os.Setenv("DOCKER_HOST", fmt.Sprintf("unix://%s/.colima/default/docker.sock", homeDir))
		os.Setenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE", "/var/run/docker.sock")
	}
	// Ryuk would remove the container when this process exits.
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	hash, err := schemaHash()
	if err != nil {
		return nil, err
	}
	if err := removeStaleContainer(ctx, hash); err != nil {
		return nil, err
	}

	pg, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("preview"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		postgres.WithSQLDriver("pgx"),
		postgres.BasicWaitStrategies(),
		testcontainers.WithReuseByName(containerName),
		testcontainers.WithLabels(map[string]string{hashLabel: hash}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	dbURL, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, err
	}

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return nil, err
	}
	var snapshotExists bool
	err = conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, snapshotName).Scan(&snapshotExists)
	if err == nil && !snapshotExists {
		err = seed(ctx, conn)
	}
	conn.Close(ctx)
	if err != nil {
		return nil, err
	}

	if snapshotExists {
		log.Println("restoring cached preview snapshot")
		err = pg.Restore(ctx, postgres.WithSnapshotName(snapshotName))
	} else {
		err = pg.Snapshot(ctx, postgres.WithSnapshotName(snapshotName))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to restore/snapshot preview database: %w", err)
	}

	return pgx.Connect(ctx, dbURL)
}

func seed(ctx context.Context, conn *pgx.Conn) error {
	log.Println("applying migrations and fixtures to a fresh preview database")
	_, maintainer := db.NewRepository(conn)
	if err := maintainer.ApplyMigrations(ctx); err != nil {
		return err
	}
	if _, err := conn.Exec(ctx, fixturesSQL); err != nil {
		return fmt.Errorf("failed to apply fixtures: %w", err)
	}
	return nil
}

// schemaHash identifies the migrations + fixtures a snapshot was built from.
func schemaHash() (string, error) {
	h := sha256.New()
	if err := db.WriteMigrations(h); err != nil {
		return "", err
	}
	h.Write([]byte(fixturesSQL))
	return hex.EncodeToString(h.Sum(nil)), nil
}

// removeStaleContainer drops the preview container when it was built from
// different migrations or fixtures, so WithReuseByName creates a fresh one.
func removeStaleContainer(ctx context.Context, hash string) error {
	cli, err := testcontainers.NewDockerClientWithOpts(ctx)
	if err != nil {
		return err
	}
	defer cli.Close()

	info, err := cli.ContainerInspect(ctx, containerName)
	if err != nil || info.Config.Labels[hashLabel] == hash {
		return nil
	}
	log.Println("migrations or fixtures changed, recreating preview container")
	return cli.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true})
}
