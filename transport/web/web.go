// Package web wires every browser-facing route (the cookie-session login
// flow, the /web/* dashboards, and DB-backed /money/import) onto a gin
// router, the same way transport/mcp wires MCP tools onto an mcp.Server.
// main.go calls Register with the real repository; cmd/webui-preview calls
// it with a mock one so the pages can be eyeballed without Postgres.
package web

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"personal/action/auth"
	"personal/action/money"
	"personal/action/progress"
	"personal/action/webui"
	"personal/gateways"
)

// Register mounts the web routes onto router. authDisabled mirrors the
// AUTH_DISABLED env var main.go already reads for /app/mcp: when true,
// webAuth is a no-op, otherwise it's the cookie-session middleware from
// action/auth. Callers are responsible for calling auth.InitializeAuth()
// themselves beforehand when authDisabled is false.
func Register(router gin.IRouter, db gateways.DB, authDisabled bool) {
	webAuth := func(c *gin.Context) { c.Next() }
	if !authDisabled {
		webAuth = auth.WebMiddleware()
	}

	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/web/design-system")
	})

	router.GET("/web/login", auth.WebLoginPageHandler)
	router.POST("/web/login", auth.WebLoginHandler)
	router.GET("/web/logout", auth.WebLogoutHandler)

	router.GET("/web/progress", dbMiddleware(db), progress.DashboardWebHandler)
	router.GET("/web/design-system", webAuth, webui.DesignSystemHandler)

	// Progress browse view — new, free-scrolling, full-color pages built on
	// the webui design system. Separate from /web/progress above, which
	// stays a purpose-built black-and-white screenshot dashboard.
	router.GET("/web/progress/browse", webAuth, dbMiddleware(db), progress.BrowseWebHandler)
	router.GET("/web/progress/browse/finished", webAuth, dbMiddleware(db), progress.BrowseFinishedWebHandler)
	router.GET("/web/progress/browse/future", webAuth, dbMiddleware(db), progress.BrowseFutureWebHandler)
	router.GET("/web/progress/browse/:id", webAuth, dbMiddleware(db), progress.BrowseDetailWebHandler)

	// Money CSV import — protected by the shared web session cookie.
	moneyImport := router.Group("/money", webAuth, dbMiddleware(db))
	moneyImport.GET("/import", money.ImportGETHandler)
	moneyImport.POST("/import", money.ImportPOSTHandler)
}

func dbMiddleware(db gateways.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := gateways.WithDB(c.Request.Context(), db)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
