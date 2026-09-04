// Package goals' web dashboard: the read-mostly GET /web/goals page (every
// goal type, active goals as a tile grid, past/completed goals as a plain
// table) plus its one write route, POST /web/goals/refresh — the web
// equivalent of the refresh_goals MCP tool. Creating/editing goals and
// logging manual progress stay MCP-only.
package goals

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"personal/action/webui"
	"personal/domain"
	"personal/gateways"
)

var goalsNav = webui.BuildNav(webui.NavGoals)

// goalsRefreshFormSrc is a plain POST form with one submit button — the
// dashboard's only write action, rendered locally (not a shared webui
// component), same convention as money-spec.md's export checkbox form.
const goalsRefreshFormSrc = `<form method="POST" action="/web/goals/refresh"><button type="submit">Refresh</button></form>`

var goalsRefreshFormTemplate = template.Must(template.New("goalsRefreshForm").Parse(goalsRefreshFormSrc))

func renderRefreshForm() (template.HTML, error) {
	var b strings.Builder
	if err := goalsRefreshFormTemplate.Execute(&b, nil); err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil
}

func goalTypeLabel(t domain.GoalType) string {
	switch t {
	case domain.GoalTypeMoneySaving:
		return "Money — Saving"
	case domain.GoalTypeMoneySpend:
		return "Money — Spend limit"
	case domain.GoalTypeExerciseMaxWeight:
		return "Exercise — Max weight"
	case domain.GoalTypeExerciseTotalVolume:
		return "Exercise — Total volume"
	case domain.GoalTypeActivityOccurrenceCount:
		return "Activity — Occurrence count"
	case domain.GoalTypeActivityStreakCount:
		return "Activity — Streak"
	case domain.GoalTypeManual:
		return "Manual"
	default:
		return string(t)
	}
}

func formatDeadline(t *time.Time) string {
	if t == nil {
		return "—"
	}
	return t.Format("2006-01-02")
}

// buildPastGoalsTable builds the past/completed section's table — final
// target/deadline only, no progress bar, since the window already closed
// (see goals-spec.md Best Practices).
func buildPastGoalsTable(goalList []domain.Goal) webui.TableData {
	rows := make([]webui.TableRow, 0, len(goalList))
	for _, g := range goalList {
		rows = append(rows, webui.TableRow{
			Cells: []string{g.Name, goalTypeLabel(g.GoalType), formatNumber(g.TargetValue), formatDeadline(g.EndsAt)},
		})
	}
	return webui.TableData{
		Columns: []webui.TableColumn{
			{Label: "Name"},
			{Label: "Type"},
			{Label: "Target", Align: "right"},
			{Label: "Deadline"},
		},
		Rows: rows,
	}
}

// DashboardWebHandler renders GET /web/goals: every active goal as a tile
// grid, then past/completed goals (ends_at before now) as a table below.
func DashboardWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)
	now := time.Now().UTC()

	tiles, err := BuildGoalTiles(ctx, db, userID, now, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load goals: %v", err)
		return
	}

	allGoals, err := db.ListGoals(ctx, domain.GoalFilter{UserID: userID, ActiveOnly: false})
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load goals: %v", err)
		return
	}
	var past []domain.Goal
	for _, g := range allGoals {
		if g.EndsAt != nil && g.EndsAt.Before(now) {
			past = append(past, g)
		}
	}

	refreshForm, err := renderRefreshForm()
	if err != nil {
		c.String(http.StatusInternalServerError, "template error: %v", err)
		return
	}

	content := refreshForm +
		webui.RenderGoalTiles(webui.GoalTilesData{Tiles: tiles, EmptyMessage: "No active goals yet"}) +
		webui.RenderTable(buildPastGoalsTable(past))

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:    "Goals",
		Nav:      goalsNav,
		UserName: c.GetString("user_name"),
		Content:  content,
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}

// RefreshWebHandler handles POST /web/goals/refresh: the web equivalent of
// the refresh_goals MCP tool with no goal_id (refreshes every active goal
// owned by the user), then redirects back to GET /web/goals.
func RefreshWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)

	if _, err := refreshAllActive(ctx, db, userID, time.Now().UTC()); err != nil {
		c.String(http.StatusInternalServerError, "Failed to refresh goals: %v", err)
		return
	}

	c.Redirect(http.StatusFound, "/web/goals")
}
