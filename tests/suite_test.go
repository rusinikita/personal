package tests

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type IntegrationTestSuite struct {
	suite.Suite
	pgContainer *postgres.PostgresContainer
	db          *sql.DB
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

	s.db, err = sql.Open("pgx", dbURL)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	ctx := context.Background()
	err := s.pgContainer.Terminate(ctx)
	s.Require().NoError(err)

	err = s.db.Close()
	s.Require().NoError(err)
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) DB() *sql.DB {
	return s.db
}
