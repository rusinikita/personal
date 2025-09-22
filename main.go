package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	sloggin "github.com/samber/slog-gin"

	"personal/action/auth"
	"personal/action/log_food"
	"personal/action/say_hi"
)

func main() {
	// Create a server with a single tool.
	server := mcp.NewServer(&mcp.Implementation{Name: "greeter", Version: "v1.0.0"}, nil)
	mcp.AddTool(server, &say_hi.MCPDefinition, say_hi.SayHi)
	// TODO: Add log_food tool with database connection
	_ = log_food.MCPDefinition

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

	err := router.Run(":8081")
	if err != nil {
		log.Fatal(err)
	}
}
