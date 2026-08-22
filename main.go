package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	sloggin "github.com/samber/slog-gin"

	"personal/action/auth"
	"personal/gateways/db"
	"personal/gateways/telegram"
	mcp2 "personal/transport/mcp"
	"personal/transport/web"
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

	err = conn.Ping(context.Background())
	if err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Create database repository
	repo, dbMaintainer := db.NewRepository(conn)

	// Apply migrations
	if err := dbMaintainer.ApplyMigrations(context.Background()); err != nil {
		log.Printf("Warning: Failed to apply migrations: %v", err)
	}

	// Initialize Telegram gateway
	telegramBotToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramBotToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable not set")
	}
	telegramChatIDStr := os.Getenv("TELEGRAM_CHAT_ID")
	if telegramChatIDStr == "" {
		log.Fatal("TELEGRAM_CHAT_ID environment variable not set")
	}
	telegramChatID, err := strconv.ParseInt(telegramChatIDStr, 10, 64)
	if err != nil {
		log.Fatal("TELEGRAM_CHAT_ID format err", err)
	}
	telegramClient, err := telegram.NewClient(telegramBotToken, telegramChatID)
	if err != nil {
		log.Fatalf("Failed to create telegram client: %v", err)
	}

	server := mcp2.Server(repo, telegramClient)

	// Create the streamable HTTP handler.
	handler := mcp.NewStreamableHTTPHandler(
		func(req *http.Request) *mcp.Server {
			return server
		},
		&mcp.StreamableHTTPOptions{
			Stateless:                    true,
			JSONResponse:                 true,
			PropagateRequestCancellation: true,
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

	// Initialize OAuth users and JWT secret from environment variables
	auth.InitializeAuth()

	authRequired := router.Group("/app")

	authDisabled := os.Getenv("AUTH_DISABLED") != ""

	if !authDisabled {
		router.GET("/.well-known/oauth-authorization-server", auth.WellKnownHandler)
		router.GET("/.well-known/oauth-authorization-server/*path", auth.WellKnownHandler)
		router.GET("/.well-known/oauth-protected-resource/*path", auth.WellKnownHandler)
		router.Any("/oauth/authorize", auth.AuthorizeHandler)
		router.POST("/oauth/token", auth.TokenHandler)
		router.POST("/oauth/register", auth.RegisterClientHandler)

		authRequired.Use(auth.Middleware())
	}

	authRequired.Any("/mcp", func(ctx *gin.Context) {
		handler.ServeHTTP(ctx.Writer, ctx.Request)
	})

	// All /web/* dashboards, web session login, and /money/import — see
	// transport/web.
	web.Register(router, repo, authDisabled)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	err = router.Run(":" + port)
	if err != nil {
		log.Fatal(err)
	}
}
