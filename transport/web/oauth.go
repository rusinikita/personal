package web

import (
	"github.com/gin-gonic/gin"

	"personal/action/auth"
	"personal/gateways"
)

// RegisterOAuth mounts the OAuth authorization server routes that protect
// /app/mcp (see docs/functions/auth-spec.md). Callers call
// auth.InitializeAuth() beforehand.
func RegisterOAuth(router gin.IRouter, db gateways.DB) {
	router.GET("/.well-known/oauth-authorization-server", auth.WellKnownHandler)
	router.GET("/.well-known/oauth-authorization-server/*path", auth.WellKnownHandler)
	router.GET("/.well-known/oauth-protected-resource", auth.ProtectedResourceHandler)
	router.GET("/.well-known/oauth-protected-resource/*path", auth.ProtectedResourceHandler)
	router.Any("/oauth/authorize", dbMiddleware(db), auth.AuthorizeHandler)
	router.POST("/oauth/token", dbMiddleware(db), auth.TokenHandler)
}
