package tests

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"runtime"
	"strings"
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

func (s *IntegrationTestSuite) UserID(skip ...int) int64 {
	skipI := 2
	if len(skip) > 0 {
		skipI = skip[0]
	}

	name := testName(skipI)

	return hashTestName(name)
}

func (s *IntegrationTestSuite) Context() context.Context {
	ctx := context.Background()
	ctx = gateways.WithUserID(ctx, s.UserID(3))
	ctx = gateways.WithDB(ctx, s.repo)

	return ctx
}

func (s *IntegrationTestSuite) AfterTest(_, testName string) {
	id := hashTestName(testName)

	s.Require().NoError(s.dbMaintainer.TruncateUserData(context.Background(), id))
}

func (s *IntegrationTestSuite) SetupSuite() {
	homeDir, err := os.UserHomeDir()
	s.Require().NoError(err)

	os.Setenv("DOCKER_HOST", fmt.Sprintf("unix://%s/.colima/default/docker.sock", homeDir))
	os.Setenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE", "/var/run/docker.sock")

	ctx := s.Context()

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
	ctx := s.Context()
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

func testName(skip int) string {
	pc, _, _, _ := runtime.Caller(skip)
	funcStr := runtime.FuncForPC(pc).Name()

	name := strings.TrimPrefix(funcStr, "personal/tests.(*IntegrationTestSuite).")

	return name
}

func hashTestName(name string) int64 {
	h := fnv.New64a()
	h.Write([]byte(name))
	return int64(h.Sum64())
}
