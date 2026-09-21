// Package progress's browse view: a new, free-scrolling, full-color
// /web/progress/browse page built on the action/webui design system. It
// sits alongside — not instead of — the fixed-viewport, black-and-white,
// top-5-only screenshot dashboard in dashboard_web.go, which this file does
// not touch.
package progress

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"personal/action/goals"
	"personal/action/webui"
	"personal/domain"
	"personal/gateways"
)

// progressGoalTypes is which goal_types the Progress browse view's embedded
// tile grid shows (see docs/functions/goals-spec.md).
var progressGoalTypes = []domain.GoalType{domain.GoalTypeActivityOccurrenceCount, domain.GoalTypeActivityStreakCount}

// BrowsePageSize is the fixed page size for every browse-view list (the
// three activity lists and a drill-down's progress-point history). It's a
// var, not a const, so tests can shrink it instead of seeding hundreds of
// rows to exercise pagination.
var BrowsePageSize = 20

var browseNav = webui.BuildNav(webui.NavProgress)

// browseCrossLinks is the small nav bar between the four activity lists.
// The URLs are fixed route constants, not user data, so embedding them as
// template.HTML directly (no templating) is safe.
const browseCrossLinks template.HTML = `<p><a href="/web/progress/browse">Active</a> · <a href="/web/progress/browse/finished">Finished</a> · <a href="/web/progress/browse/future">Future</a> · <a href="/web/progress/browse/paused">Paused</a></p>`

// progressValueOption is one radio choice in the drill-down's "log a point"
// form: a value paired with the one emoji chosen to represent it.
type progressValueOption struct {
	Value int
	Emoji string
}

// progressValueOptions returns the radio group choices for pt, worst to
// best, using one fixed emoji metaphor per progress_type (not the full
// get_progress_type_examples set — a web form needs exactly one label per
// value, see progress-spec.md Best Practices). promise_state has no ±2
// anywhere in the domain, so it gets 3 options instead of 5.
func progressValueOptions(pt domain.ProgressType) []progressValueOption {
	switch pt {
	case domain.ProgressTypeMood:
		return []progressValueOption{{-2, "⛈️"}, {-1, "🌧️"}, {0, "☁️"}, {1, "⛅"}, {2, "☀️"}}
	case domain.ProgressTypeHabitProgress:
		return []progressValueOption{{-2, "❌"}, {-1, "😔"}, {0, "🤔"}, {1, "👍"}, {2, "💪"}}
	case domain.ProgressTypeProjectProgress:
		return []progressValueOption{{-2, "🔄"}, {-1, "↩️"}, {0, "⏸️"}, {1, "➡️"}, {2, "🚀"}}
	case domain.ProgressTypePromiseState:
		return []progressValueOption{{-1, "🤷"}, {0, "💭"}, {1, "✅"}}
	default:
		return []progressValueOption{{-2, "-2"}, {-1, "-1"}, {0, "0"}, {1, "1"}, {2, "2"}}
	}
}

// pointFormStepOption is one open-step checkbox in the "log a point" form.
type pointFormStepOption struct {
	ID   int64
	Name string
}

// pointFormData feeds pointFormTemplate for one activity's drill-down page.
type pointFormData struct {
	ActionURL string
	Options   []progressValueOption
	Steps     []pointFormStepOption
	Error     string
}

// pointFormContentSrc is the "log a point" form on the standalone
// /web/progress/browse/{id}/points/new page — a local html/template
// constant, not a shared webui component (action/webui has no form
// components at all), same convention as goalsRefreshFormSrc
// (action/goals/dashboard_web.go) and importFormContentSrc
// (action/money/import_web.go).
const pointFormContentSrc = `{{if .Error}}<p style="color: var(--pico-del-color)">{{.Error}}</p>{{end}}
<form method="POST" action="{{.ActionURL}}">
    <fieldset>
        <legend>Value</legend>
        {{range .Options}}<label style="display:inline-block; margin-right: 1em"><input type="radio" name="value" value="{{.Value}}" required> {{.Emoji}} ({{.Value}})</label>
        {{end}}
    </fieldset>
    <label for="point-note">Note</label>
    <textarea id="point-note" name="note" rows="3" placeholder="optional"></textarea>
    {{if .Steps}}<fieldset>
        <legend>Close steps</legend>
        {{range .Steps}}<label style="display:block"><input type="checkbox" name="close_step_ids" value="{{.ID}}"> {{.Name}}</label>
        {{end}}
    </fieldset>{{end}}
    <details>
        <summary>More options</summary>
        <label for="point-hours-left">Hours left</label>
        <input type="number" id="point-hours-left" name="hours_left" step="0.5" placeholder="optional, for projects">
        <label for="point-progress-at">When</label>
        <input type="datetime-local" id="point-progress-at" name="progress_at" placeholder="optional, defaults to now">
        <label for="point-new-one-time-steps">New one-time steps</label>
        <input type="text" id="point-new-one-time-steps" name="new_one_time_steps" placeholder="separate multiple with ;">
        <label for="point-new-repeatable-steps">New repeatable steps</label>
        <input type="text" id="point-new-repeatable-steps" name="new_repeatable_steps" placeholder="separate multiple with ;">
    </details>
    <button type="submit">Log point</button>
</form>`

var pointFormTemplate = template.Must(template.New("pointForm").Parse(pointFormContentSrc))

func renderPointForm(data pointFormData) (template.HTML, error) {
	var b strings.Builder
	if err := pointFormTemplate.Execute(&b, data); err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil
}

// newPointHeaderData feeds newPointHeaderTemplate — activity name and
// progress_type go through html/template's auto-escaping (not
// template.HTML) since they're user-entered data, unlike the fixed BackURL.
type newPointHeaderData struct {
	BackURL      string
	ActivityName string
	ProgressType string
}

// newPointHeaderSrc is the standalone "log a point" page's header: which
// activity the point is being logged for, so it's unambiguous away from the
// drill-down page's own context.
const newPointHeaderSrc = `<a class="webui-detail-back" href="{{.BackURL}}">← Back to {{.ActivityName}}</a>
<h2>Log a point — {{.ActivityName}}</h2>
<p>{{.ProgressType}}</p>`

var newPointHeaderTemplate = template.Must(template.New("newPointHeader").Parse(newPointHeaderSrc))

func renderNewPointHeader(data newPointHeaderData) (template.HTML, error) {
	var b strings.Builder
	if err := newPointHeaderTemplate.Execute(&b, data); err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil
}

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

// stepsCompactCell formats an activity's open steps for their own "Steps"
// table column — TableRow.Cells is plain []string (no webui component
// renders rich HTML per-cell), so this stays plain text: one line per step,
// joined with "\n" (components/table.html renders cells with
// white-space:pre-line so the breaks actually show instead of collapsing
// into one run of text), ☑️ marking each and 🔁 added only for repeatable
// ones (one_time is the unmarked default).
func stepsCompactCell(steps []domain.Step) string {
	if len(steps) == 0 {
		return ""
	}
	names := make([]string, len(steps))
	for i, st := range steps {
		name := "☑️ " + st.Name
		if st.Type == domain.StepTypeRepeatable {
			name += " 🔁"
		}
		names[i] = name
	}
	return strings.Join(names, "\n")
}

// renderStepsList renders an activity's open steps as a labeled list for the
// drill-down page (unlike stepsCompactCell's one-line browse summary) —
// a heading plus one row per step marked with ☑️, styled by .webui-steps* in
// layout.html so it reads as a real checklist instead of blending into the
// description paragraph. one_time is the unmarked default (most steps are),
// so only repeatable gets a 🔁 marker — no badge text, since the emoji alone
// reads clearly next to a short step name. Empty when there are no open
// steps, so it's safe to append unconditionally.
func renderStepsList(steps []domain.Step) template.HTML {
	if len(steps) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<div class="webui-steps-heading">Open steps</div><ul class="webui-steps">`)
	for _, st := range steps {
		b.WriteString("<li>☑️ ")
		b.WriteString(template.HTMLEscapeString(st.Name))
		if st.Type == domain.StepTypeRepeatable {
			b.WriteString(` <span class="webui-step-type" title="repeatable">🔁</span>`)
		}
		b.WriteString("</li>")
	}
	b.WriteString("</ul>")
	return template.HTML(b.String())
}

// buildActivityTable builds one list's table. includeType is false for the
// active list's per-progress_type sections (see activeSectionOrder), where
// the section heading already says the type — a Type column there would
// just repeat it. The still-combined finished/future lists pass true.
// stepsByActivity being non-nil (only the active list passes one — see
// stepsCompactCell) adds a trailing "Steps" column; passing nil (finished/
// future/paused) omits it entirely, since those activities never have
// visible steps.
func buildActivityTable(activities []domain.Activity, extraLabel string, extra activityExtraColumn, includeType bool, stepsByActivity map[int64][]domain.Step, pagination *webui.PaginationData) webui.TableData {
	showSteps := stepsByActivity != nil
	rows := make([]webui.TableRow, 0, len(activities))
	for _, a := range activities {
		cells := []string{a.Name, a.Description}
		if includeType {
			cells = append(cells, progressTypeLabel(a.ProgressType))
		}
		cells = append(cells, formatFrequency(a.FrequencyDays), extra(a))
		if showSteps {
			cells = append(cells, stepsCompactCell(stepsByActivity[a.ID]))
		}
		rows = append(rows, webui.TableRow{
			Cells:   cells,
			LinkURL: fmt.Sprintf("/web/progress/browse/%d", a.ID),
		})
	}
	columns := []webui.TableColumn{{Label: "Name"}, {Label: "Description"}}
	if includeType {
		columns = append(columns, webui.TableColumn{Label: "Type"})
	}
	columns = append(columns, webui.TableColumn{Label: "Frequency"}, webui.TableColumn{Label: extraLabel})
	if showSteps {
		columns = append(columns, webui.TableColumn{Label: "Steps"})
	}
	return webui.TableData{
		Columns:    columns,
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

// renderActivityList is shared by the finished/future list handlers: it
// paginates filter, fetches the page plus the total count, and renders the
// list page as one combined table across all progress_types (see
// activeSectionOrder for why the active list, handled separately by
// BrowseWebHandler, does not use this).
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

	table := buildActivityTable(activities, extraLabel, extra, true, nil, buildPagination(page, total, baseURL))
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

// activeSectionOrder is the active list's fixed per-progress_type section
// order and heading text (see progress-spec.md). A type with zero active
// activities still renders its heading with an empty table, so the
// four-section layout stays predictable.
var activeSectionOrder = []struct {
	Type    domain.ProgressType
	Heading string
}{
	{domain.ProgressTypeHabitProgress, "Habits"},
	{domain.ProgressTypePromiseState, "Promises"},
	{domain.ProgressTypeProjectProgress, "Projects"},
	{domain.ProgressTypeMood, "Mood"},
}

// BrowseWebHandler renders GET /web/progress/browse: an activity goal tile
// grid, then every active activity split into four unpaginated tables (one
// per progress_type, activeSectionOrder), unlike dashboard_web.go's
// top-5-only screenshot view.
func BrowseWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)

	goalTiles, err := goals.BuildGoalTiles(ctx, db, userID, time.Now().UTC(), progressGoalTypes)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load goals: %v", err)
		return
	}

	extra := func(a domain.Activity) string { return formatTimeAgoPtr(a.LastPointAt) }
	content := browseCrossLinks + webui.RenderGoalTiles(webui.GoalTilesData{Tiles: goalTiles})
	for _, section := range activeSectionOrder {
		activities, err := db.ListActivities(ctx, domain.ActivityFilter{
			UserID:       userID,
			Statuses:     []domain.ActivityStatus{domain.ActivityStatusActive},
			ProgressType: section.Type,
		})
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to list activities: %v", err)
			return
		}
		stepsByActivity := make(map[int64][]domain.Step, len(activities))
		for _, a := range activities {
			steps, err := db.ListSteps(ctx, domain.StepFilter{UserID: userID, ActivityID: a.ID, Statuses: []domain.StepStatus{domain.StepStatusActive}})
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to list steps: %v", err)
				return
			}
			if len(steps) > 0 {
				stepsByActivity[a.ID] = steps
			}
		}
		table := buildActivityTable(activities, "Last update", extra, false, stepsByActivity, nil)
		content += template.HTML(fmt.Sprintf("<h3>%s</h3>", section.Heading)) + webui.RenderTable(table)
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:    "Progress — Active",
		Nav:      browseNav,
		UserName: c.GetString("user_name"),
		Content:  content,
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}

// BrowseFinishedWebHandler renders GET /web/progress/browse/finished:
// activities with status finished or dropped.
func BrowseFinishedWebHandler(c *gin.Context) {
	renderActivityList(c, domain.ActivityFilter{Statuses: []domain.ActivityStatus{domain.ActivityStatusFinished, domain.ActivityStatusDropped}},
		"Progress — Finished", "/web/progress/browse/finished",
		"Finished", func(a domain.Activity) string { return formatTimeAgoPtr(a.EndedAt) })
}

// BrowseFutureWebHandler renders GET /web/progress/browse/future:
// activities whose started_at is still in the future.
func BrowseFutureWebHandler(c *gin.Context) {
	renderActivityList(c, domain.ActivityFilter{FutureOnly: true, Statuses: []domain.ActivityStatus{domain.ActivityStatusActive}},
		"Progress — Future", "/web/progress/browse/future",
		"Starts", func(a domain.Activity) string { return a.StartedAt.Format("2006-01-02") })
}

// BrowsePausedWebHandler renders GET /web/progress/browse/paused:
// activities with status paused, all progress_types combined (same shape
// as Finished/Future), with a "Deferred until" column instead of
// Finished/Starts.
func BrowsePausedWebHandler(c *gin.Context) {
	renderActivityList(c, domain.ActivityFilter{Statuses: []domain.ActivityStatus{domain.ActivityStatusPaused}},
		"Progress — Paused", "/web/progress/browse/paused",
		"Deferred until", func(a domain.Activity) string {
			if a.DeferredUntil == nil {
				return "—"
			}
			return a.DeferredUntil.Format("2006-01-02")
		})
}

// BrowseDetailWebHandler renders GET /web/progress/browse/{id}: a
// drill-down with a "+ Add" button to the standalone log-a-point page, trend
// stat tiles, a line chart of the full (unpaginated) value-over-time
// series, and a paginated table of every progress point.
func BrowseDetailWebHandler(c *gin.Context) {
	activityID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid activity id")
		return
	}
	renderBrowseDetail(c, activityID)
}

// BrowseNewPointWebHandler renders GET /web/progress/browse/{id}/points/new:
// the standalone "log a point" page reached via the drill-down's "+ Add"
// button. Its header names the activity the point is being logged for, so
// it stays unambiguous away from the drill-down's own context.
func BrowseNewPointWebHandler(c *gin.Context) {
	activityID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid activity id")
		return
	}
	renderNewPointPage(c, activityID, "")
}

// BrowseCreatePointWebHandler handles POST /web/progress/browse/{id}/points:
// logs a progress point from the standalone log-a-point page's form,
// instead of requiring the MCP tool create_progress_point. Validation is
// shared with that tool via createProgressPoint
// (create_progress_point_mcp.go), so the two entry points can't drift
// apart. On success it redirects to the GET drill-down (write-then-redirect,
// same pattern as goals.RefreshWebHandler); on failure it re-renders the
// same log-a-point page in place with an inline error, same pattern as
// money's CSV import form.
func BrowseCreatePointWebHandler(c *gin.Context) {
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

	value, err := strconv.Atoi(c.PostForm("value"))
	if err != nil {
		renderNewPointPage(c, activityID, "value is required")
		return
	}

	var hoursLeft *float64
	if raw := strings.TrimSpace(c.PostForm("hours_left")); raw != "" {
		hl, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			renderNewPointPage(c, activityID, "hours_left must be a number")
			return
		}
		hoursLeft = &hl
	}

	var progressAt time.Time
	if raw := strings.TrimSpace(c.PostForm("progress_at")); raw != "" {
		progressAt, err = time.ParseInLocation("2006-01-02T15:04", raw, time.Local)
		if err != nil {
			renderNewPointPage(c, activityID, "when must be a valid date/time")
			return
		}
	}

	point, err := createProgressPoint(ctx, db, userID, activityID, value, c.PostForm("note"), hoursLeft, progressAt)
	if err != nil {
		renderNewPointPage(c, activityID, err.Error())
		return
	}

	if err := applyStepFormActions(ctx, db, userID, activityID, point.ID,
		c.PostFormArray("close_step_ids"), c.PostForm("new_one_time_steps"), c.PostForm("new_repeatable_steps")); err != nil {
		renderNewPointPage(c, activityID, err.Error())
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/web/progress/browse/%d", activityID))
}

// applyStepFormActions closes every checked step and queues new steps from
// the two ";"-split text fields, after a progress point is logged from the
// web form — one form submission, one redirect, same requirement as the
// point itself (see progress-spec.md Best Practices).
func applyStepFormActions(ctx context.Context, db gateways.DB, userID, activityID, pointID int64, closeStepIDs []string, newOneTime, newRepeatable string) error {
	now := time.Now()
	for _, raw := range closeStepIDs {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid step id %q", raw)
		}
		step, err := db.GetStep(ctx, id, userID)
		if err != nil {
			return fmt.Errorf("database error: %w", err)
		}
		if step == nil {
			return fmt.Errorf("step not found")
		}
		step.Status = domain.StepStatusFinished
		step.ClosedAt = &now
		step.CompletedByProgressPointID = &pointID
		if err := db.UpdateStep(ctx, step); err != nil {
			return fmt.Errorf("failed to close step: %w", err)
		}
	}

	if err := queueNewSteps(ctx, db, userID, activityID, pointID, newOneTime, domain.StepTypeOneTime); err != nil {
		return err
	}
	return queueNewSteps(ctx, db, userID, activityID, pointID, newRepeatable, domain.StepTypeRepeatable)
}

// queueNewSteps splits raw on ";" and creates one active step per non-empty
// entry, tied back to pointID via CreatedByProgressPointID.
func queueNewSteps(ctx context.Context, db gateways.DB, userID, activityID, pointID int64, raw string, stepType domain.StepType) error {
	for _, name := range strings.Split(raw, ";") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, err := db.CreateStep(ctx, &domain.Step{
			UserID:                   userID,
			ActivityID:               activityID,
			Name:                     name,
			Type:                     stepType,
			CreatedByProgressPointID: &pointID,
		}); err != nil {
			return fmt.Errorf("failed to create step: %w", err)
		}
	}
	return nil
}

// renderNewPointPage renders the standalone log-a-point page for
// activityID, shared by BrowseNewPointWebHandler and the POST handler's
// re-render-on-error path. formError is shown inline above the form when
// non-empty.
func renderNewPointPage(c *gin.Context, activityID int64, formError string) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)

	activity, err := db.GetActivity(ctx, activityID, userID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load activity: %v", err)
		return
	}
	if activity == nil {
		c.String(http.StatusNotFound, "activity not found")
		return
	}

	backURL := fmt.Sprintf("/web/progress/browse/%d", activityID)
	header, err := renderNewPointHeader(newPointHeaderData{
		BackURL:      backURL,
		ActivityName: activity.Name,
		ProgressType: progressTypeLabel(activity.ProgressType),
	})
	if err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
		return
	}

	var stepOptions []pointFormStepOption
	if activity.Status == domain.ActivityStatusActive {
		steps, err := db.ListSteps(ctx, domain.StepFilter{UserID: userID, ActivityID: activityID, Statuses: []domain.StepStatus{domain.StepStatusActive}})
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load steps: %v", err)
			return
		}
		for _, st := range steps {
			stepOptions = append(stepOptions, pointFormStepOption{ID: st.ID, Name: st.Name})
		}
	}

	form, err := renderPointForm(pointFormData{
		ActionURL: fmt.Sprintf("/web/progress/browse/%d/points", activityID),
		Options:   progressValueOptions(activity.ProgressType),
		Steps:     stepOptions,
		Error:     formError,
	})
	if err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:    "Progress — Log point — " + activity.Name,
		Nav:      browseNav,
		UserName: c.GetString("user_name"),
		Content:  header + form,
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}

// renderBrowseDetail renders the drill-down page for activityID.
func renderBrowseDetail(c *gin.Context, activityID int64) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)

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

	var stepsHTML template.HTML
	if activity.Status == domain.ActivityStatusActive {
		steps, err := db.ListSteps(ctx, domain.StepFilter{UserID: userID, ActivityID: activityID, Statuses: []domain.StepStatus{domain.StepStatusActive}})
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load steps: %v", err)
			return
		}
		stepsHTML = renderStepsList(steps)
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
		Title:       activity.Name,
		Description: renderBoldMarkdown(activity.Description) + stepsHTML,
		BackURL:     "/web/progress/browse",
		BackText:    "Back to active list",
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

	addPointButton := template.HTML(fmt.Sprintf(`<p><a href="/web/progress/browse/%d/points/new" role="button">+ Add</a></p>`, activityID))

	content := addPointButton + webui.RenderLineChart(chart) + webui.RenderDetailView(detail)

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
