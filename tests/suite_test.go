package tests

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"personal/gateways"
	"personal/gateways/db"
)

type IntegrationTestSuite struct {
	suite.Suite
	pgContainer *postgres.PostgresContainer
	conn        *pgx.Conn

	repo         gateways.DB
	dbMaintainer gateways.DBMaintainer
}

func (s *IntegrationTestSuite) SetupSuite() {
	homeDir, err := os.UserHomeDir()
	s.Require().NoError(err)

	os.Setenv("DOCKER_HOST", fmt.Sprintf("unix://%s/.colima/default/docker.sock", homeDir))
	os.Setenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE", "/var/run/docker.sock")

	ctx := context.Background()

	s.pgContainer, err = postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		postgres.BasicWaitStrategies(),
	)
	s.Require().NoError(err)

	dbURL, err := s.pgContainer.ConnectionString(ctx, "sslmode=disable")
	s.Require().NoError(err)

	s.conn, err = pgx.Connect(ctx, dbURL)
	s.Require().NoError(err)

	s.Require().NoError(s.conn.Ping(ctx))

	s.repo, s.dbMaintainer = db.NewRepository(s.conn)

	// Apply migrations
	err = s.dbMaintainer.ApplyMigrations(ctx)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	ctx := context.Background()
	err := s.pgContainer.Terminate(ctx)
	s.Require().NoError(err)

	err = s.conn.Close(ctx)
	s.Require().NoError(err)
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) DB() *pgx.Conn {
	return s.conn
}

func (s *IntegrationTestSuite) Repo() gateways.DB {
	return s.repo
}
