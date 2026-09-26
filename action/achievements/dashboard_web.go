// Package achievements' web dashboard: the read-mostly GET /web/achievements page (every
// achievement type, active achievements as a tile grid, past/completed achievements as a plain
// table) plus its one write route, POST /web/achievements/refresh — the web
// equivalent of the refresh_achievements MCP tool. Creating/editing achievements and
// logging manual progress stay MCP-only.
package achievements

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

var achievementsNav = webui.BuildNav(webui.NavAchievements)

// achievementsRefreshFormSrc is a plain POST form with one submit button — the
// dashboard's only write action, rendered locally (not a shared webui
// component), same convention as money-spec.md's export checkbox form.
const achievementsRefreshFormSrc = `<form method="POST" action="/web/achievements/refresh"><button type="submit">Refresh</button></form>`

var achievementsRefreshFormTemplate = template.Must(template.New("achievementsRefreshForm").Parse(achievementsRefreshFormSrc))

func renderRefreshForm() (template.HTML, error) {
	var b strings.Builder
	if err := achievementsRefreshFormTemplate.Execute(&b, nil); err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil
}

func achievementTypeLabel(t domain.AchievementType) string {
	switch t {
	case domain.AchievementTypeMoneySaving:
		return "Money — Saving"
	case domain.AchievementTypeMoneySpend:
		return "Money — Spend limit"
	case domain.AchievementTypeExerciseMaxWeight:
		return "Exercise — Max weight"
	case domain.AchievementTypeExerciseTotalVolume:
		return "Exercise — Total volume"
	case domain.AchievementTypeActivityOccurrenceCount:
		return "Activity — Occurrence count"
	case domain.AchievementTypeActivityStreakCount:
		return "Activity — Streak"
	case domain.AchievementTypeManual:
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

// buildPastAchievementsTable builds the past/completed section's table — final
// target/deadline only, no progress bar, since the window already closed
// (see achievements-spec.md Best Practices).
func buildPastAchievementsTable(achievementList []domain.Achievement) webui.TableData {
	rows := make([]webui.TableRow, 0, len(achievementList))
	for _, g := range achievementList {
		rows = append(rows, webui.TableRow{
			Cells: []string{g.Name, achievementTypeLabel(g.AchievementType), formatNumber(g.TargetValue), formatDeadline(g.EndsAt)},
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

// DashboardWebHandler renders GET /web/achievements: every active achievement as a tile
// grid, then past/completed achievements (ends_at before now) as a table below.
func DashboardWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)
	now := time.Now().UTC()

	tiles, err := BuildAchievementTiles(ctx, db, userID, now, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load achievements: %v", err)
		return
	}

	allAchievements, err := db.ListAchievements(ctx, domain.AchievementFilter{UserID: userID, ActiveOnly: false})
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load achievements: %v", err)
		return
	}
	var past []domain.Achievement
	for _, g := range allAchievements {
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
		webui.RenderAchievementTiles(webui.AchievementTilesData{Tiles: tiles, EmptyMessage: "No active achievements yet"}) +
		webui.RenderTable(buildPastAchievementsTable(past))

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:    "Achievements",
		Nav:      achievementsNav,
		UserName: c.GetString("user_name"),
		Content:  content,
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}

// RefreshWebHandler handles POST /web/achievements/refresh: the web equivalent of
// the refresh_achievements MCP tool with no achievement_id (refreshes every active achievement
// owned by the user), then redirects back to GET /web/achievements.
func RefreshWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)

	if _, err := refreshAllActive(ctx, db, userID, time.Now().UTC()); err != nil {
		c.String(http.StatusInternalServerError, "Failed to refresh achievements: %v", err)
		return
	}

	c.Redirect(http.StatusFound, "/web/achievements")
}
