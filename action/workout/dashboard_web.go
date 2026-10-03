// Package workout's web dashboard: read-only /web/workouts pages built on
// the action/webui design system, for reviewing personal records and
// per-exercise trends in a browser. Logging sets from the browser lives in
// sessions_web.go.
package workout

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"personal/action/achievements"
	"personal/action/webui"
	"personal/domain"
	"personal/gateways"
)

// workoutAchievementTypes is which achievement_types the Workouts embedded tile grid
// shows (see docs/functions/achievements-spec.md).
var workoutAchievementTypes = []domain.AchievementType{domain.AchievementTypeExerciseMaxWeight, domain.AchievementTypeExerciseTotalVolume}

// exerciseHistoryWorkoutsLimit is how many newest workouts containing the
// exercise the drill-down loads sets from, for its chart and history table.
const exerciseHistoryWorkoutsLimit = 100

var workoutsNav = webui.BuildNav(webui.NavWorkouts)

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
	userID := webui.CurrentUserID(c)

	achievementTiles, err := achievements.BuildAchievementTiles(ctx, db, userID, time.Now().UTC(), workoutAchievementTypes)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load achievements: %v", err)
		return
	}

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
				r.Exercise.Description,
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
			{Label: "Description"},
			{Label: "Times performed", Align: "right"},
			{Label: "Max weight (kg)", Align: "right"},
			{Label: "Max reps", Align: "right"},
			{Label: "Est. 1RM (kg)", Align: "right"},
		},
		Rows: rows,
	}

	content := workoutsCrossLinks + webui.RenderAchievementTiles(webui.AchievementTilesData{Tiles: achievementTiles}) + webui.RenderTable(table)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:    "Workouts — Personal Records",
		Nav:      workoutsNav,
		UserName: c.GetString("user_name"),
		Content:  content,
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}

// ExerciseDetailWebHandler renders GET /web/workouts/{id}: a drill-down with
// personal-record stat tiles, one dual-axis chart (weight left, reps right,
// one point per workout — its first set) and the set history table, built
// from the exercise's sets in its latest exerciseHistoryWorkoutsLimit workouts.
func ExerciseDetailWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)

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

	// ListExerciseSets returns the sets oldest-to-newest — exactly the order
	// the chart needs.
	sets, err := db.ListExerciseSets(ctx, userID, exerciseID, exerciseHistoryWorkoutsLimit)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load sets: %v", err)
		return
	}

	// setsByWorkout keeps each workout's sets in logging order; workoutOrder
	// is the workouts oldest-to-newest.
	setsByWorkout := map[int64][]domain.Set{}
	var workoutOrder []int64
	points := make([]webui.DualAxisChartPoint, 0, len(sets))
	for _, s := range sets {
		if _, seen := setsByWorkout[s.WorkoutID]; !seen {
			workoutOrder = append(workoutOrder, s.WorkoutID)
		}
		setsByWorkout[s.WorkoutID] = append(setsByWorkout[s.WorkoutID], s)
		// Only the workout's first set becomes a chart point.
		if len(setsByWorkout[s.WorkoutID]) > 1 {
			continue
		}
		if s.WeightKg <= 0 && s.Reps <= 0 {
			continue
		}
		point := webui.DualAxisChartPoint{Label: s.CreatedAt.Format("2006-01-02")}
		if s.WeightKg > 0 {
			weight := s.WeightKg
			point.Left = &weight
		}
		if s.Reps > 0 {
			reps := float64(s.Reps)
			point.Right = &reps
		}
		points = append(points, point)
	}

	setCount := int64(len(sets))

	stats := []webui.StatTileData{
		{Label: "Max weight (kg)", Value: formatWeight(records.MaxWeight)},
		{Label: "Max reps", Value: formatReps(records.MaxReps)},
		{Label: "Est. 1RM (kg)", Value: estimated1RM(records.MaxWeight), Emphasis: true},
		{Label: "Times performed", Value: strconv.FormatInt(setCount, 10)},
	}

	detail := webui.DetailViewData{
		Title:       exercise.Name,
		Description: template.HTML(template.HTMLEscapeString(exercise.Description)),
		BackURL:     "/web/workouts",
		BackText:    "Back to personal records",
		Stats:       stats,
	}

	chart := webui.DualAxisChartData{
		ID:              fmt.Sprintf("chart-weight-reps-%d", exerciseID),
		Title:           exercise.Name + " — weight and reps of the first set per workout",
		LeftSeriesName:  "Weight (kg)",
		RightSeriesName: "Reps",
		Points:          points,
	}

	content := webui.RenderDetailView(detail) + webui.RenderDualAxisChart(chart)

	if len(sets) > 0 {
		rows := make([]webui.TableRow, 0, len(sets))
		for i := len(workoutOrder) - 1; i >= 0; i-- {
			for _, s := range setsByWorkout[workoutOrder[i]] {
				rows = append(rows, webui.TableRow{
					Cells:   []string{s.CreatedAt.Format("2006-01-02"), formatSet(s)},
					LinkURL: fmt.Sprintf("/web/workouts/sessions/%d", s.WorkoutID),
				})
			}
		}
		content += webui.RenderTable(webui.TableData{
			Columns: []webui.TableColumn{{Label: "Date"}, {Label: "Set"}},
			Rows:    rows,
		})
	}

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
