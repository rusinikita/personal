package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	sloggin "github.com/samber/slog-gin"

	"personal/action/add_food"
	"personal/action/auth"
	"personal/action/log_food"
	"personal/action/say_hi"
	"personal/gateways"
	"personal/gateways/db"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file", err)
	}

	// Initialize database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close(context.Background())

	// Create database repository
	repo, dbMaintainer := db.NewRepository(conn)

	// Apply migrations
	if err := dbMaintainer.ApplyMigrations(context.Background()); err != nil {
		log.Printf("Warning: Failed to apply migrations: %v", err)
	}

	// Create a server with a single tool.
	server := mcp.NewServer(&mcp.Implementation{Name: "greeter", Version: "v1.0.0"}, nil)

	server.AddReceivingMiddleware(func(handler mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (result mcp.Result, err error) {
			// Add database to context
			ctx = gateways.WithDB(ctx, repo)
			return handler(ctx, method, req)
		}
	})

	mcp.AddTool(server, &say_hi.MCPDefinition, say_hi.SayHi)
	mcp.AddTool(server, &add_food.MCPDefinition, add_food.AddFood)
	mcp.AddTool(server, &log_food.MCPDefinition, log_food.LogFood)

	// Create the streamable HTTP handler.
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, nil)

	url := "127.0.0.1:8081"
	log.Printf("MCP server listening on %s", url)

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

	auth.BaseAuthURL = "https://splendid-chicken-42.telebit.io"

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

	err = router.Run(":8081")
	if err != nil {
		log.Fatal(err)
	}
}
