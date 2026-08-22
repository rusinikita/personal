// Package progress's browse view: a new, free-scrolling, full-color
// /web/progress/browse page built on the action/webui design system. It
// sits alongside — not instead of — the fixed-viewport, black-and-white,
// top-5-only screenshot dashboard in dashboard_web.go, which this file does
// not touch.
package progress

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"personal/action/webui"
	"personal/domain"
	"personal/gateways"
)

// BrowsePageSize is the fixed page size for every browse-view list (the
// three activity lists and a drill-down's progress-point history). It's a
// var, not a const, so tests can shrink it instead of seeding hundreds of
// rows to exercise pagination.
var BrowsePageSize = 20

var browseNav = webui.BuildNav(webui.NavProgress)

// browseCrossLinks is the small nav bar between the three activity lists.
// The URLs are fixed route constants, not user data, so embedding them as
// template.HTML directly (no templating) is safe.
const browseCrossLinks template.HTML = `<p><a href="/web/progress/browse">Active</a> · <a href="/web/progress/browse/finished">Finished</a> · <a href="/web/progress/browse/future">Future</a></p>`

// buildPagination computes prev/next links (each carrying ?page=N against
// baseURL) from the current page and total row count. Returns nil when
// everything fits on one page, so callers can assign it straight to
// TableData.Pagination without an extra nil check.
func buildPagination(page, totalCount int, baseURL string) *webui.PaginationData {
	return webui.BuildPagination(page, totalCount, BrowsePageSize, func(p int) string {
		return fmt.Sprintf("%s?page=%d", baseURL, p)
	})
}

// activityExtraColumn produces the value for each list's one differing
// column (last update / finished / starts).
type activityExtraColumn func(a domain.Activity) string

func buildActivityTable(activities []domain.Activity, extraLabel string, extra activityExtraColumn, pagination *webui.PaginationData) webui.TableData {
	rows := make([]webui.TableRow, 0, len(activities))
	for _, a := range activities {
		rows = append(rows, webui.TableRow{
			Cells:   []string{a.Name, progressTypeLabel(a.ProgressType), formatFrequency(a.FrequencyDays), extra(a)},
			LinkURL: fmt.Sprintf("/web/progress/browse/%d", a.ID),
		})
	}
	return webui.TableData{
		Columns: []webui.TableColumn{
			{Label: "Name"},
			{Label: "Type"},
			{Label: "Frequency"},
			{Label: extraLabel},
		},
		Rows:       rows,
		Pagination: pagination,
	}
}

func progressTypeLabel(pt domain.ProgressType) string {
	switch pt {
	case domain.ProgressTypeMood:
		return "Mood"
	case domain.ProgressTypeHabitProgress:
		return "Habit"
	case domain.ProgressTypeProjectProgress:
		return "Project"
	case domain.ProgressTypePromiseState:
		return "Promise"
	default:
		return string(pt)
	}
}

// renderActivityList is shared by the three activity list handlers
// (active/finished/future): it paginates filter, fetches the page plus the
// total count, and renders the list page.
func renderActivityList(c *gin.Context, filter domain.ActivityFilter, title, baseURL, extraLabel string, extra activityExtraColumn) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}

	filter.UserID = webui.CurrentUserID(c)

	total, err := db.CountActivities(ctx, filter)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to count activities: %v", err)
		return
	}

	page := webui.ParsePage(c)
	filter.Limit = int64(BrowsePageSize)
	filter.Offset = int64((page - 1) * BrowsePageSize)

	activities, err := db.ListActivities(ctx, filter)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to list activities: %v", err)
		return
	}

	table := buildActivityTable(activities, extraLabel, extra, buildPagination(page, total, baseURL))
	content := browseCrossLinks + webui.RenderTable(table)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:    title,
		Nav:      browseNav,
		UserName: c.GetString("user_name"),
		Content:  content,
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}

// BrowseWebHandler renders GET /web/progress/browse: every active project
// and habit (no top-5 limit, unlike dashboard_web.go's screenshot view).
func BrowseWebHandler(c *gin.Context) {
	renderActivityList(c, domain.ActivityFilter{ActiveOnly: true}, "Progress — Active", "/web/progress/browse",
		"Last update", func(a domain.Activity) string { return formatTimeAgoPtr(a.LastPointAt) })
}

// BrowseFinishedWebHandler renders GET /web/progress/browse/finished:
// activities with ended_at set.
func BrowseFinishedWebHandler(c *gin.Context) {
	renderActivityList(c, domain.ActivityFilter{ActiveOnly: false}, "Progress — Finished", "/web/progress/browse/finished",
		"Finished", func(a domain.Activity) string { return formatTimeAgoPtr(a.EndedAt) })
}

// BrowseFutureWebHandler renders GET /web/progress/browse/future:
// activities whose started_at is still in the future.
func BrowseFutureWebHandler(c *gin.Context) {
	renderActivityList(c, domain.ActivityFilter{FutureOnly: true}, "Progress — Future", "/web/progress/browse/future",
		"Starts", func(a domain.Activity) string { return a.StartedAt.Format("2006-01-02") })
}

// BrowseDetailWebHandler renders GET /web/progress/browse/{id}: a
// drill-down with trend stat tiles, a line chart of the full (unpaginated)
// value-over-time series, and a paginated table of every progress point.
func BrowseDetailWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)

	activityID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid activity id")
		return
	}

	activity, err := db.GetActivity(ctx, activityID, userID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load activity: %v", err)
		return
	}
	if activity == nil {
		c.String(http.StatusNotFound, "activity not found")
		return
	}

	now := time.Now()
	overall, err := db.GetTrendStats(ctx, activityID, userID, activity.StartedAt, now)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load trend stats: %v", err)
		return
	}
	lastMonth, err := db.GetTrendStats(ctx, activityID, userID, now.AddDate(0, 0, -30), now)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load trend stats: %v", err)
		return
	}
	lastWeek, err := db.GetTrendStats(ctx, activityID, userID, now.AddDate(0, 0, -7), now)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load trend stats: %v", err)
		return
	}

	// Full, unpaginated history for the chart — the trend is more useful
	// shown whole than sliced to the current table page.
	allPoints, err := db.ListProgress(ctx, domain.ProgressFilter{UserID: userID, ActivityID: activityID})
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load progress history: %v", err)
		return
	}
	// allPoints is newest-first (ListProgress default order); the chart
	// reads left-to-right oldest-to-newest.
	chartPoints := make([]webui.LineChartPoint, len(allPoints))
	for i, point := range allPoints {
		chartPoints[len(allPoints)-1-i] = webui.LineChartPoint{Label: point.ProgressAt.Format("2006-01-02"), Value: float64(point.Value)}
	}

	total, err := db.CountProgress(ctx, domain.ProgressFilter{UserID: userID, ActivityID: activityID})
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to count progress history: %v", err)
		return
	}

	page := webui.ParsePage(c)
	baseURL := fmt.Sprintf("/web/progress/browse/%d", activityID)
	pagePoints, err := db.ListProgress(ctx, domain.ProgressFilter{
		UserID: userID, ActivityID: activityID,
		Limit: int64(BrowsePageSize), Offset: int64((page - 1) * BrowsePageSize),
	})
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load progress history: %v", err)
		return
	}

	historyRows := make([]webui.TableRow, 0, len(pagePoints))
	for _, p := range pagePoints {
		historyRows = append(historyRows, webui.TableRow{
			Cells: []string{p.ProgressAt.Format("2006-01-02 15:04"), strconv.Itoa(p.Value), p.Note},
		})
	}

	detail := webui.DetailViewData{
		Title:    activity.Name,
		BackURL:  "/web/progress/browse",
		BackText: "Back to active list",
		Stats: []webui.StatTileData{
			{Label: "Overall", Value: fmt.Sprintf("%.1f", overall.Average), SubLabel: fmt.Sprintf("%d check-ins", overall.Count)},
			{Label: "Last 30 days", Value: fmt.Sprintf("%.1f", lastMonth.Average), SubLabel: fmt.Sprintf("%d check-ins", lastMonth.Count), Emphasis: true},
			{Label: "Last 7 days", Value: fmt.Sprintf("%.1f", lastWeek.Average), SubLabel: fmt.Sprintf("%d check-ins", lastWeek.Count)},
		},
		Table: webui.TableData{
			Columns: []webui.TableColumn{
				{Label: "Date"},
				{Label: "Value", Align: "right"},
				{Label: "Note"},
			},
			Rows:       historyRows,
			Pagination: buildPagination(page, total, baseURL),
		},
	}

	chart := webui.LineChartData{
		ID:         fmt.Sprintf("chart-activity-%d", activityID),
		Title:      activity.Name + " — value over time",
		SeriesName: "Value",
		Points:     chartPoints,
	}

	content := webui.RenderLineChart(chart) + webui.RenderDetailView(detail)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:    "Progress — " + activity.Name,
		Nav:      browseNav,
		UserName: c.GetString("user_name"),
		Content:  content,
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}
