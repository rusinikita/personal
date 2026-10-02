// Package workout's sessions web pages: /web/workouts/sessions is the
// history of the last workouts, /web/workouts/sessions/new and
// /web/workouts/sessions/{id} are the workout screen with a set-adding form.
// A workout row is created lazily, only by the first set posted from the
// new screen (see docs/functions/workout-spec.md).
package workout

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"personal/action/webui"
	"personal/domain"
	"personal/gateways"
)

// historyWorkoutsLimit is how many newest workouts the history page shows.
const historyWorkoutsLimit = 10

// workoutsCrossLinks links the two Workouts sub-sections, shown on top of
// both the personal records list and the sessions history.
const workoutsCrossLinks template.HTML = `<p><a href="/web/workouts">Personal records</a> · <a href="/web/workouts/sessions">Sessions</a></p>`

type historyExercise struct {
	Name     string
	FirstSet string
}

type historyWorkout struct {
	Date      string
	URL       string
	Exercises []historyExercise
}

// historySrc is the sessions history — a local html/template constant, same
// convention as pointFormContentSrc (action/progress).
const historySrc = `<p><a href="/web/workouts/sessions/new" role="button">New workout</a></p>
{{range .}}<article>
<header><a href="{{.URL}}">{{.Date}}</a></header>
{{if .Exercises}}<ul>
{{range .Exercises}}<li>{{.Name}} — {{.FirstSet}}</li>
{{end}}</ul>{{else}}<p>No sets</p>{{end}}
</article>
{{else}}<p>No workouts yet</p>
{{end}}`

var historyTemplate = template.Must(template.New("workoutHistory").Parse(historySrc))

type exerciseOption struct {
	ID       int64
	Name     string
	Selected bool
}

// setFormData feeds setFormTemplate. WeightKg and Reps are the raw inputs,
// kept as typed when the form is re-rendered with an error.
type setFormData struct {
	Title     string
	Error     string
	ActionURL string
	Exercises []exerciseOption
	WeightKg  string
	Reps      string
}

const setFormSrc = `<p><a href="/web/workouts/sessions">← Sessions</a></p>
<h2>{{.Title}}</h2>
{{if .Error}}<p style="color: var(--pico-del-color)">{{.Error}}</p>{{end}}
<form method="POST" action="{{.ActionURL}}">
    <label for="exercise-id">Exercise</label>
    <select id="exercise-id" name="exercise_id" required>
        <option value="">Choose exercise</option>
        {{range .Exercises}}<option value="{{.ID}}"{{if .Selected}} selected{{end}}>{{.Name}}</option>
        {{end}}
    </select>
    <div class="grid">
        <label>Weight / difficulty (kg)
            <input type="number" name="weight_kg" min="0" step="0.5" inputmode="decimal" value="{{.WeightKg}}">
        </label>
        <label>Reps
            <input type="number" name="reps" min="1" step="1" inputmode="numeric" value="{{.Reps}}" required>
        </label>
    </div>
    <button type="submit">Add set</button>
</form>`

var setFormTemplate = template.Must(template.New("workoutSetForm").Parse(setFormSrc))

func executeTemplate(t *template.Template, data any) (template.HTML, error) {
	var b strings.Builder
	if err := t.Execute(&b, data); err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil
}

// formatSet renders a set as "80 kg × 8", "12 reps" or "60 s".
func formatSet(s domain.Set) string {
	switch {
	case s.WeightKg > 0 && s.Reps > 0:
		return fmt.Sprintf("%g kg × %d", s.WeightKg, s.Reps)
	case s.Reps > 0:
		return fmt.Sprintf("%d reps", s.Reps)
	default:
		return fmt.Sprintf("%d s", s.DurationSeconds)
	}
}

func writeWorkoutsPage(c *gin.Context, status int, title string, content template.HTML) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(status)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:    title,
		Nav:      workoutsNav,
		UserName: c.GetString("user_name"),
		Content:  content,
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}

// SessionsWebHandler renders GET /web/workouts/sessions: the last 10
// workouts, each with its exercises and the first set logged for each.
// Built from the same ListSets + GetExercisesByIDs calls list_workouts makes.
func SessionsWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)

	workouts, err := db.ListWorkouts(ctx, userID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load workouts: %v", err)
		return
	}
	if len(workouts) > historyWorkoutsLimit {
		workouts = workouts[:historyWorkoutsLimit]
	}

	setsByWorkout := map[int64][]domain.Set{}
	exerciseNames := map[int64]string{}
	if len(workouts) > 0 {
		oldest := workouts[len(workouts)-1].StartedAt
		sets, err := db.ListSets(ctx, userID, oldest, time.Now())
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load sets: %v", err)
			return
		}
		var exerciseIDs []int64
		// ListSets returns newest first; walk it backwards to keep logging order.
		for i := len(sets) - 1; i >= 0; i-- {
			s := sets[i]
			setsByWorkout[s.WorkoutID] = append(setsByWorkout[s.WorkoutID], s)
			if _, ok := exerciseNames[s.ExerciseID]; !ok {
				exerciseNames[s.ExerciseID] = ""
				exerciseIDs = append(exerciseIDs, s.ExerciseID)
			}
		}
		exercises, err := db.GetExercisesByIDs(ctx, userID, exerciseIDs)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load exercises: %v", err)
			return
		}
		for _, ex := range exercises {
			exerciseNames[ex.ID] = ex.Name
		}
	}

	history := make([]historyWorkout, 0, len(workouts))
	for _, w := range workouts {
		hw := historyWorkout{
			Date: w.StartedAt.Format("2006-01-02"),
			URL:  fmt.Sprintf("/web/workouts/sessions/%d", w.ID),
		}
		seen := map[int64]bool{}
		for _, s := range setsByWorkout[w.ID] {
			if seen[s.ExerciseID] {
				continue
			}
			seen[s.ExerciseID] = true
			hw.Exercises = append(hw.Exercises, historyExercise{Name: exerciseNames[s.ExerciseID], FirstSet: formatSet(s)})
		}
		history = append(history, hw)
	}

	content, err := executeTemplate(historyTemplate, history)
	if err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
		return
	}
	writeWorkoutsPage(c, http.StatusOK, "Workouts — Sessions", workoutsCrossLinks+content)
}

// NewSessionWebHandler renders GET /web/workouts/sessions/new: an empty
// workout screen. It creates nothing — the workout appears only with its
// first set.
func NewSessionWebHandler(c *gin.Context) {
	renderSessionPage(c, nil, setFormData{})
}

// SessionWebHandler renders GET /web/workouts/sessions/{id}: the set-adding
// form for an existing workout plus the sets already logged in it.
func SessionWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	w, ok := loadSessionWorkout(c, db)
	if !ok {
		return
	}
	renderSessionPage(c, w, setFormData{})
}

// CreateSessionSetWebHandler handles POST /web/workouts/sessions/new/sets:
// the first set from the new screen creates the workout. An explicit "New
// workout" always starts a new one, so any still-open workout is closed at
// its last set's time first, same as log_workout_set does past its 2-hour
// window. On success redirects to the new workout's screen.
func CreateSessionSetWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)

	input, errMsg := parseSetForm(ctx, c, db, userID)
	if errMsg != "" {
		renderSessionPage(c, nil, setFormDataFromPost(c, errMsg))
		return
	}

	lastSet, err := db.GetLastSet(ctx, userID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load last set: %v", err)
		return
	}
	if lastSet != nil && lastSet.Workout.CompletedAt == nil {
		if err := db.CloseWorkout(ctx, lastSet.Workout.ID, lastSet.Set.CreatedAt); err != nil {
			c.String(http.StatusInternalServerError, "Failed to close previous workout: %v", err)
			return
		}
	}

	now := time.Now()
	workoutID, err := db.CreateWorkout(ctx, &domain.Workout{UserID: userID, StartedAt: now})
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to create workout: %v", err)
		return
	}
	if err := createWebSet(ctx, db, userID, workoutID, input, now); err != nil {
		c.String(http.StatusInternalServerError, "Failed to create set: %v", err)
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/web/workouts/sessions/%d", workoutID))
}

// AddSessionSetWebHandler handles POST /web/workouts/sessions/{id}/sets: adds
// a set straight into the given workout (no 2-hour rule) and redirects back
// to its screen, where the form is prefilled with this set.
func AddSessionSetWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)

	w, ok := loadSessionWorkout(c, db)
	if !ok {
		return
	}

	input, errMsg := parseSetForm(ctx, c, db, userID)
	if errMsg != "" {
		renderSessionPage(c, w, setFormDataFromPost(c, errMsg))
		return
	}

	if err := createWebSet(ctx, db, userID, w.ID, input, time.Now()); err != nil {
		c.String(http.StatusInternalServerError, "Failed to create set: %v", err)
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/web/workouts/sessions/%d", w.ID))
}

// loadSessionWorkout resolves the :id param to the current user's workout,
// writing a 400/404/500 response and returning false when it can't.
func loadSessionWorkout(c *gin.Context, db gateways.DB) (*domain.Workout, bool) {
	workoutID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid workout id")
		return nil, false
	}
	workouts, err := db.GetWorkoutsByIDs(c.Request.Context(), webui.CurrentUserID(c), []int64{workoutID})
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load workout: %v", err)
		return nil, false
	}
	if len(workouts) == 0 {
		c.String(http.StatusNotFound, "workout not found")
		return nil, false
	}
	return &workouts[0], true
}

// parseSetForm reads the set form into a LogWorkoutSetInput, validated the
// same way log_workout_set validates it. A non-empty message is a
// user-facing validation error.
func parseSetForm(ctx context.Context, c *gin.Context, db gateways.DB, userID int64) (LogWorkoutSetInput, string) {
	var input LogWorkoutSetInput

	exerciseID, err := strconv.ParseInt(c.PostForm("exercise_id"), 10, 64)
	if err != nil {
		return input, "exercise is required"
	}
	input.ExerciseID = exerciseID

	reps, err := strconv.ParseInt(strings.TrimSpace(c.PostForm("reps")), 10, 64)
	if err != nil || reps <= 0 {
		return input, "reps must be a positive whole number"
	}
	input.Reps = reps

	if raw := strings.TrimSpace(c.PostForm("weight_kg")); raw != "" {
		weight, err := strconv.ParseFloat(raw, 64)
		if err != nil || weight < 0 {
			return input, "weight must be a non-negative number"
		}
		input.WeightKg = weight
	}

	if err := validateLogWorkoutSetInput(input); err != nil {
		return input, err.Error()
	}

	exercise, err := db.GetExercise(ctx, input.ExerciseID, userID)
	if err != nil {
		return input, fmt.Sprintf("failed to load exercise: %v", err)
	}
	if exercise == nil {
		return input, "exercise not found"
	}

	return input, ""
}

func createWebSet(ctx context.Context, db gateways.DB, userID, workoutID int64, input LogWorkoutSetInput, at time.Time) error {
	_, err := db.CreateSet(ctx, &domain.Set{
		UserID:     userID,
		WorkoutID:  workoutID,
		ExerciseID: input.ExerciseID,
		Reps:       input.Reps,
		WeightKg:   input.WeightKg,
		CreatedAt:  at,
	})
	return err
}

// setFormDataFromPost keeps the submitted values so a re-rendered form with
// an error doesn't lose what was typed.
func setFormDataFromPost(c *gin.Context, errMsg string) setFormData {
	return setFormData{
		Error:    errMsg,
		WeightKg: c.PostForm("weight_kg"),
		Reps:     c.PostForm("reps"),
	}
}

// renderSessionPage renders the workout screen: the set form, then (for an
// existing workout) its sets in logging order. w is nil for the new screen.
// On a re-render with an error the form keeps the posted values; otherwise
// it is prefilled with the workout's last set (exercise, weight, reps), so
// the next set of the same exercise is a single submit.
func renderSessionPage(c *gin.Context, w *domain.Workout, form setFormData) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)

	exercises, err := db.ListExercisesByUsage(ctx, userID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load exercises: %v", err)
		return
	}

	var workoutSets []domain.Set
	if w != nil {
		sets, err := db.ListSets(ctx, userID, w.StartedAt, time.Now())
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load sets: %v", err)
			return
		}
		// ListSets returns newest first; walk it backwards to keep logging order.
		for i := len(sets) - 1; i >= 0; i-- {
			if sets[i].WorkoutID == w.ID {
				workoutSets = append(workoutSets, sets[i])
			}
		}
	}

	selected := c.PostForm("exercise_id")
	if form.Error == "" && len(workoutSets) > 0 {
		last := workoutSets[len(workoutSets)-1]
		selected = strconv.FormatInt(last.ExerciseID, 10)
		if last.WeightKg > 0 {
			form.WeightKg = strconv.FormatFloat(last.WeightKg, 'f', -1, 64)
		}
		if last.Reps > 0 {
			form.Reps = strconv.FormatInt(last.Reps, 10)
		}
	}

	exerciseNames := make(map[int64]string, len(exercises))
	form.Exercises = make([]exerciseOption, 0, len(exercises))
	for _, ex := range exercises {
		exerciseNames[ex.ID] = ex.Name
		form.Exercises = append(form.Exercises, exerciseOption{
			ID:       ex.ID,
			Name:     ex.Name,
			Selected: strconv.FormatInt(ex.ID, 10) == selected,
		})
	}

	form.Title = "New workout"
	form.ActionURL = "/web/workouts/sessions/new/sets"
	if w != nil {
		form.Title = "Workout " + w.StartedAt.Format("2006-01-02")
		form.ActionURL = fmt.Sprintf("/web/workouts/sessions/%d/sets", w.ID)
	}

	content, err := executeTemplate(setFormTemplate, form)
	if err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
		return
	}

	if len(workoutSets) > 0 {
		rows := make([]webui.TableRow, 0, len(workoutSets))
		for _, s := range workoutSets {
			rows = append(rows, webui.TableRow{Cells: []string{exerciseNames[s.ExerciseID], formatSet(s)}})
		}
		content += webui.RenderTable(webui.TableData{
			Columns: []webui.TableColumn{{Label: "Exercise"}, {Label: "Set"}},
			Rows:    rows,
		})
	}

	writeWorkoutsPage(c, http.StatusOK, "Workouts — "+form.Title, content)
}
