// Package workout's web dashboard: read-only /web/workouts pages built on
// the action/webui design system, for reviewing personal records and
// per-exercise trends in a browser. Logging/editing workouts stays
// MCP/Telegram-bot-only — this file adds no write routes.
package workout

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"personal/action/webui"
	"personal/domain"
	"personal/gateways"
)

// historyLimit bounds the exercise-history query used to build the
// drill-down's trend charts. A single-user personal tool won't log anywhere
// near this many sessions for one exercise, so this is effectively
// "unlimited" while still satisfying GetExerciseHistory's required limit
// param (a literal 0 would mean "return zero rows", not "no limit").
const historyLimit = 10000

var workoutsNav = []webui.NavItem{
	{Label: "Home", URL: "/web"},
	{Label: "Money", URL: "/web/money"},
	{Label: "Progress", URL: "/web/progress/browse"},
	{Label: "Workouts", URL: "/web/workouts", Active: true},
	{Label: "Design System", URL: "/web/design-system"},
}

// currentUserID reads the session user id set by auth.WebMiddleware,
// falling back to 1 the same way action/progress's browse view does for
// AUTH_DISABLED / webui-preview use without a real session.
func currentUserID(c *gin.Context) int64 {
	userID := gateways.UserIDFromContext(c.Request.Context())
	if userID == 0 {
		userID = 1
	}
	return userID
}

// formatWeight renders a nullable weight record as "82.5" or "—".
func formatWeight(rec *domain.SetRecord) string {
	if rec == nil {
		return "—"
	}
	return fmt.Sprintf("%.1f", rec.WeightKg)
}

// formatReps renders a nullable reps record as "8" or "—".
func formatReps(rec *domain.SetRecord) string {
	if rec == nil {
		return "—"
	}
	return strconv.FormatInt(rec.Reps, 10)
}

// estimated1RM applies the Epley formula (same one get_personal_records_mcp.go
// uses) to a max-weight record, or "—" when there is none to estimate from.
func estimated1RM(maxWeight *domain.SetRecord) string {
	if maxWeight == nil {
		return "—"
	}
	return fmt.Sprintf("%.1f", maxWeight.WeightKg*(1+float64(maxWeight.Reps)/30))
}

// PersonalRecordsWebHandler renders GET /web/workouts: every exercise ever
// logged, sorted by times performed (set count) descending, with each row's
// personal records and estimated 1RM.
func PersonalRecordsWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := currentUserID(c)

	records, err := db.ListPersonalRecords(ctx, userID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load personal records: %v", err)
		return
	}

	rows := make([]webui.TableRow, 0, len(records))
	for _, r := range records {
		rows = append(rows, webui.TableRow{
			Cells: []string{
				r.Exercise.Name,
				string(r.Exercise.EquipmentType),
				strconv.FormatInt(r.SetCount, 10),
				formatWeight(r.Records.MaxWeight),
				formatReps(r.Records.MaxReps),
				estimated1RM(r.Records.MaxWeight),
			},
			LinkURL: fmt.Sprintf("/web/workouts/%d", r.Exercise.ID),
		})
	}

	table := webui.TableData{
		Columns: []webui.TableColumn{
			{Label: "Name"},
			{Label: "Equipment"},
			{Label: "Times performed", Align: "right"},
			{Label: "Max weight (kg)", Align: "right"},
			{Label: "Max reps", Align: "right"},
			{Label: "Est. 1RM (kg)", Align: "right"},
		},
		Rows: rows,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:    "Workouts — Personal Records",
		Nav:      workoutsNav,
		UserName: c.GetString("user_name"),
		Content:  webui.RenderTable(table),
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}

// ExerciseDetailWebHandler renders GET /web/workouts/{id}: a drill-down with
// personal-record stat tiles plus weight-over-time and reps-over-time trend
// charts built from every set ever logged for the exercise.
func ExerciseDetailWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := currentUserID(c)

	exerciseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid exercise id")
		return
	}

	exercise, err := db.GetExercise(ctx, exerciseID, userID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load exercise: %v", err)
		return
	}
	if exercise == nil {
		c.String(http.StatusNotFound, "exercise not found")
		return
	}

	records, err := db.GetPersonalRecords(ctx, userID, exerciseID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load personal records: %v", err)
		return
	}

	workouts, err := db.GetExerciseHistory(ctx, userID, exerciseID, historyLimit, 0)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load exercise history: %v", err)
		return
	}
	workoutIDs := make([]int64, len(workouts))
	for i, w := range workouts {
		workoutIDs[i] = w.ID
	}
	// ListSetsByExerciseAndWorkouts orders by created_at ASC regardless of
	// workoutIDs order, so the sets below are already oldest-to-newest —
	// exactly the order the trend charts need.
	sets, err := db.ListSetsByExerciseAndWorkouts(ctx, userID, exerciseID, workoutIDs)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load sets: %v", err)
		return
	}

	weightPoints := make([]webui.LineChartPoint, 0, len(sets))
	repPoints := make([]webui.LineChartPoint, 0, len(sets))
	for _, s := range sets {
		label := s.CreatedAt.Format("2006-01-02")
		if s.WeightKg > 0 {
			weightPoints = append(weightPoints, webui.LineChartPoint{Label: label, Value: s.WeightKg})
		}
		if s.Reps > 0 {
			repPoints = append(repPoints, webui.LineChartPoint{Label: label, Value: float64(s.Reps)})
		}
	}

	setCount := int64(len(sets))

	stats := []webui.StatTileData{
		{Label: "Max weight (kg)", Value: formatWeight(records.MaxWeight)},
		{Label: "Max reps", Value: formatReps(records.MaxReps)},
		{Label: "Est. 1RM (kg)", Value: estimated1RM(records.MaxWeight), Emphasis: true},
		{Label: "Times performed", Value: strconv.FormatInt(setCount, 10)},
	}

	detail := webui.DetailViewData{
		Title:    exercise.Name,
		BackURL:  "/web/workouts",
		BackText: "Back to personal records",
		Stats:    stats,
	}

	weightChart := webui.LineChartData{
		ID:         fmt.Sprintf("chart-weight-%d", exerciseID),
		Title:      exercise.Name + " — weight over time",
		SeriesName: "Weight (kg)",
		Points:     weightPoints,
	}
	repChart := webui.LineChartData{
		ID:         fmt.Sprintf("chart-reps-%d", exerciseID),
		Title:      exercise.Name + " — reps over time",
		SeriesName: "Reps",
		Points:     repPoints,
	}

	content := webui.RenderDetailView(detail) + webui.RenderLineChart(weightChart) + webui.RenderLineChart(repChart)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:    "Workouts — " + exercise.Name,
		Nav:      workoutsNav,
		UserName: c.GetString("user_name"),
		Content:  content,
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}
