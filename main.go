package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	sloggin "github.com/samber/slog-gin"

	"personal/action/auth"
	"personal/gateways/db"
	"personal/transport"
)

func main() {
	err := godotenv.Load(".env.local")
	if err != nil {
		log.Println("Error loading .env.local file", err)
	}

	// Initialize database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	pgxPoolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatal("DATABASE_URL format err", err)
	}

	conn, err := pgxpool.NewWithConfig(context.Background(), pgxPoolConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close()

	// Create database repository
	repo, dbMaintainer := db.NewRepository(conn)

	// Apply migrations
	if err := dbMaintainer.ApplyMigrations(context.Background()); err != nil {
		log.Printf("Warning: Failed to apply migrations: %v", err)
	}

	server := transport.MCPServer(repo)

	// Create the streamable HTTP handler.
	handler := mcp.NewStreamableHTTPHandler(
		func(req *http.Request) *mcp.Server {
			return server
		},
		&mcp.StreamableHTTPOptions{
			Stateless:    true,
			JSONResponse: true,
		},
	)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	config := sloggin.Config{
		WithRequestID:      true,
		WithRequestBody:    true,
		WithRequestHeader:  true,
		WithResponseBody:   true,
		WithResponseHeader: true,
	}

	router := gin.New()
	router.Use(sloggin.NewWithConfig(logger, config), gin.Recovery())

	gin.SetMode(gin.DebugMode)

	auth.BaseAuthURL = os.Getenv("BASE_URL")
	if auth.BaseAuthURL == "" {
		log.Fatal("BASE_URL environment variable not set")
	}

	router.GET("/.well-known/oauth-authorization-server", auth.WellKnownHandler)
	router.GET("/.well-known/oauth-authorization-server/*path", auth.WellKnownHandler)
	router.GET("/.well-known/oauth-protected-resource/*path", auth.WellKnownHandler)
	router.Any("/oauth/authorize", auth.AuthorizeHandler)
	router.POST("/oauth/token", auth.TokenHandler)
	router.POST("/oauth/register", auth.RegisterClientHandler)

	authRequired := router.Group("/app")
	authRequired.Use(auth.Middleware())

	authRequired.Any("/mcp", func(ctx *gin.Context) {
		handler.ServeHTTP(ctx.Writer, ctx.Request)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	err = router.Run(":" + port)
	if err != nil {
		log.Fatal(err)
	}
}
