// Package web wires every browser-facing route (the cookie-session login
// flow, the /web/* dashboards, and DB-backed /money/import) onto a gin
// router, the same way transport/mcp wires MCP tools onto an mcp.Server.
// main.go calls Register with the production database; cmd/webui-preview
// calls it with a local Postgres testcontainer seeded with fixtures.
package web

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"personal/action/achievements"
	"personal/action/auth"
	"personal/action/docs"
	"personal/action/ideas"
	"personal/action/money"
	"personal/action/progress"
	"personal/action/webui"
	"personal/action/workout"
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

	redirectToDesignSystem := func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/web/design-system")
	}
	router.GET("/", redirectToDesignSystem)
	router.GET("/web", redirectToDesignSystem)
	router.GET("/web/", redirectToDesignSystem)

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
	router.GET("/web/progress/browse/paused", webAuth, dbMiddleware(db), progress.BrowsePausedWebHandler)
	router.GET("/web/progress/browse/:id", webAuth, dbMiddleware(db), progress.BrowseDetailWebHandler)
	router.GET("/web/progress/browse/:id/points/new", webAuth, dbMiddleware(db), progress.BrowseNewPointWebHandler)
	router.POST("/web/progress/browse/:id/points", webAuth, dbMiddleware(db), progress.BrowseCreatePointWebHandler)

	// Workouts dashboard — read-only personal records list + per-exercise
	// drill-down, plus the sessions pages for logging sets, built on the
	// webui design system.
	router.GET("/web/workouts", webAuth, dbMiddleware(db), workout.PersonalRecordsWebHandler)
	router.GET("/web/workouts/sessions", webAuth, dbMiddleware(db), workout.SessionsWebHandler)
	router.GET("/web/workouts/sessions/new", webAuth, dbMiddleware(db), workout.NewSessionWebHandler)
	router.POST("/web/workouts/sessions/new/sets", webAuth, dbMiddleware(db), workout.CreateSessionSetWebHandler)
	router.GET("/web/workouts/sessions/:id", webAuth, dbMiddleware(db), workout.SessionWebHandler)
	router.POST("/web/workouts/sessions/:id/sets", webAuth, dbMiddleware(db), workout.AddSessionSetWebHandler)
	router.GET("/web/workouts/:id", webAuth, dbMiddleware(db), workout.ExerciseDetailWebHandler)

	// Money dashboard — read-only overview, transaction list, calendar, and
	// spending export, built on the webui design system. Adding/editing
	// transactions stays out of scope; see the CSV import routes below.
	router.GET("/web/money", webAuth, dbMiddleware(db), money.MoneyDashboardWebHandler)
	router.GET("/web/money/transactions", webAuth, dbMiddleware(db), money.TransactionsWebHandler)
	router.GET("/web/money/calendar", webAuth, dbMiddleware(db), money.CalendarWebHandler)
	router.GET("/web/money/export", webAuth, dbMiddleware(db), money.ExportWebHandler)
	router.GET("/web/money/export/download", webAuth, dbMiddleware(db), money.ExportDownloadWebHandler)

	// Achievements dashboard — every achievement type, read-mostly plus one Refresh write
	// route, built on the webui design system. Creating/editing achievements and
	// logging manual progress stay MCP-only.
	router.GET("/web/achievements", webAuth, dbMiddleware(db), achievements.DashboardWebHandler)
	router.POST("/web/achievements/refresh", webAuth, dbMiddleware(db), achievements.RefreshWebHandler)

	// Achievements e-ink dashboard — purpose-built fixed-viewport, black-and-white
	// screenshot page for a physical always-on display, unauthenticated
	// like /web/progress above (see achievements.EinkDashboardWebHandler).
	router.GET("/web/achievements/eink", dbMiddleware(db), achievements.EinkDashboardWebHandler)
	// Pre-rename e-ink URL, kept because the physical display still points at it.
	router.GET("/web/goals/eink", achievements.LegacyEinkRedirectWebHandler)

	// Ideas — quick capture form plus the inbox cards, the spike ideas, and
	// a search over every status; reviews and resolutions stay MCP-only.
	router.GET("/web/ideas", webAuth, dbMiddleware(db), ideas.IdeasWebHandler)
	router.POST("/web/ideas", webAuth, dbMiddleware(db), ideas.CreateIdeaWebHandler)
	router.GET("/web/ideas/spike", webAuth, dbMiddleware(db), ideas.SpikeIdeasWebHandler)
	router.GET("/web/ideas/search", webAuth, dbMiddleware(db), ideas.SearchIdeasWebHandler)

	// Docs — hand-written convention documents embedded in action/docs,
	// read-only, no DB access.
	router.GET("/web/docs", webAuth, docs.IndexWebHandler)
	router.GET("/web/docs/:topic", webAuth, docs.DocWebHandler)

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
