package tests

// Covers the "Web interface for Workouts" dashboard from
// docs/functions/workout-spec.md (backlog: 19-08-26). Read-only pages built
// on the webui design system: a personal-records list sorted by times
// performed (set count), and a per-exercise drill-down with weight/reps
// trend charts. Logging/editing workouts stays MCP/bot-only.

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/workout"
	"personal/domain"
	"personal/gateways"
)

// workoutDashboardRouter builds a minimal gin engine wired to the test
// suite's DB and user_id, serving the two workout dashboard routes the same
// way transport/web does.
func (s *IntegrationTestSuite) workoutDashboardRouter(ctx context.Context) *gin.Engine {
	userID := gateways.UserIDFromContext(ctx)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		reqCtx := gateways.WithDB(c.Request.Context(), s.Repo())
		reqCtx = gateways.WithUserID(reqCtx, userID)
		c.Request = c.Request.WithContext(reqCtx)
		c.Next()
	})
	r.GET("/web/workouts", workout.PersonalRecordsWebHandler)
	r.GET("/web/workouts/:id", workout.ExerciseDetailWebHandler)
	return r
}

func (s *IntegrationTestSuite) createExercise(ctx context.Context, name, equipmentType string) int64 {
	_, ex, err := workout.CreateExercise(ctx, nil, workout.CreateExerciseInput{Name: name, EquipmentType: equipmentType})
	require.NoError(s.T(), err)
	return ex.ID
}

func (s *IntegrationTestSuite) createWorkout(ctx context.Context, startedAt time.Time) int64 {
	id, err := s.Repo().CreateWorkout(ctx, &domain.Workout{UserID: gateways.UserIDFromContext(ctx), StartedAt: startedAt})
	require.NoError(s.T(), err)
	return id
}

func (s *IntegrationTestSuite) createSet(ctx context.Context, workoutID, exerciseID int64, reps int64, weightKg float64, at time.Time) {
	_, err := s.Repo().CreateSet(ctx, &domain.Set{
		UserID: gateways.UserIDFromContext(ctx), WorkoutID: workoutID, ExerciseID: exerciseID,
		Reps: reps, WeightKg: weightKg, CreatedAt: at,
	})
	require.NoError(s.T(), err)
}

// --- GET /web/workouts -------------------------------------------------------

func (s *IntegrationTestSuite) TestPersonalRecordsList_SortedByTimesPerformed() {
	ctx := s.Context()
	now := time.Now()

	oneSetID := s.createExercise(ctx, "Once Exercise", "barbell")
	w1 := s.createWorkout(ctx, now.AddDate(0, 0, -1))
	s.createSet(ctx, w1, oneSetID, 5, 50, now.AddDate(0, 0, -1))

	threeSetsID := s.createExercise(ctx, "Thrice Exercise", "barbell")
	w2 := s.createWorkout(ctx, now.AddDate(0, 0, -2))
	s.createSet(ctx, w2, threeSetsID, 5, 50, now.AddDate(0, 0, -2))
	s.createSet(ctx, w2, threeSetsID, 5, 55, now.AddDate(0, 0, -2))
	w3 := s.createWorkout(ctx, now.AddDate(0, 0, -1))
	s.createSet(ctx, w3, threeSetsID, 5, 60, now.AddDate(0, 0, -1))

	twoSetsID := s.createExercise(ctx, "Twice Exercise", "barbell")
	s.createSet(ctx, w3, twoSetsID, 5, 50, now.AddDate(0, 0, -1))
	s.createSet(ctx, w3, twoSetsID, 5, 55, now.AddDate(0, 0, -1))

	r := s.workoutDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/workouts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()

	iThrice := strings.Index(body, "Thrice Exercise")
	iTwice := strings.Index(body, "Twice Exercise")
	iOnce := strings.Index(body, "Once Exercise")
	require.NotEqual(s.T(), -1, iThrice)
	require.NotEqual(s.T(), -1, iTwice)
	require.NotEqual(s.T(), -1, iOnce)
	assert.Less(s.T(), iThrice, iTwice, "exercise with more sets must be listed first")
	assert.Less(s.T(), iTwice, iOnce, "exercise with more sets must be listed first")
}

func (s *IntegrationTestSuite) TestPersonalRecordsList_ExcludesNeverPerformedExercises() {
	ctx := s.Context()
	s.createExercise(ctx, "Never Done", "dumbbells")

	usedID := s.createExercise(ctx, "Done Once", "barbell")
	w := s.createWorkout(ctx, time.Now())
	s.createSet(ctx, w, usedID, 5, 50, time.Now())

	r := s.workoutDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/workouts", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	body := rec.Body.String()
	assert.Contains(s.T(), body, "Done Once")
	assert.NotContains(s.T(), body, "Never Done")
}

func (s *IntegrationTestSuite) TestPersonalRecordsList_ShowsRecordsAndEstimated1RM() {
	ctx := s.Context()
	now := time.Now()

	exID := s.createExercise(ctx, "Bench Press", "barbell")
	w1 := s.createWorkout(ctx, now.AddDate(0, 0, -2))
	s.createSet(ctx, w1, exID, 8, 60, now.AddDate(0, 0, -2)) // max reps
	w2 := s.createWorkout(ctx, now.AddDate(0, 0, -1))
	s.createSet(ctx, w2, exID, 5, 100, now.AddDate(0, 0, -1)) // max weight

	r := s.workoutDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/workouts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, "Bench Press")
	assert.Contains(s.T(), body, "barbell")
	assert.Contains(s.T(), body, "100.0", "max weight column")
	assert.Contains(s.T(), body, "8", "max reps column")
	// estimated_1rm: 100 * (1 + 5/30) = 116.7
	assert.Contains(s.T(), body, "116.7", "estimated 1RM column")
}

func (s *IntegrationTestSuite) TestPersonalRecordsList_RowsLinkToDrillDown() {
	ctx := s.Context()
	exID := s.createExercise(ctx, "Squat", "barbell")
	w := s.createWorkout(ctx, time.Now())
	s.createSet(ctx, w, exID, 5, 80, time.Now())

	r := s.workoutDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/workouts", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Contains(s.T(), rec.Body.String(), fmt.Sprintf(`href="/web/workouts/%d"`, exID))
}

// --- GET /web/workouts/{id} ---------------------------------------------------

func (s *IntegrationTestSuite) TestExerciseDetail_ShowsStatTilesAndTrendCharts() {
	ctx := s.Context()
	now := time.Now()

	exID := s.createExercise(ctx, "Deadlift", "barbell")
	w1 := s.createWorkout(ctx, now.AddDate(0, 0, -14))
	s.createSet(ctx, w1, exID, 8, 80, now.AddDate(0, 0, -14))
	w2 := s.createWorkout(ctx, now.AddDate(0, 0, -7))
	s.createSet(ctx, w2, exID, 5, 100, now.AddDate(0, 0, -7))

	r := s.workoutDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/web/workouts/%d", exID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, "Deadlift")
	assert.Contains(s.T(), body, "webui-stat-tile")
	assert.Equal(s.T(), 1, strings.Count(body, "<canvas id="), "must render one dual-axis chart for weight and reps")
	assert.Contains(s.T(), body, "data: [80,100]", "weight line must be oldest-to-newest")
	assert.Contains(s.T(), body, "data: [8,5]", "reps line must be oldest-to-newest")
	assert.Contains(s.T(), body, `yAxisID: "y1"`, "reps line must be on the right Y axis")
	assert.Contains(s.T(), body, `href="/web/workouts"`, "must have a back link to the list")
}

func (s *IntegrationTestSuite) TestExerciseDetail_SetWithoutWeight_GapInWeightLine() {
	ctx := s.Context()
	now := time.Now()

	exID := s.createExercise(ctx, "Pull-up", "bodyweight")
	w1 := s.createWorkout(ctx, now.AddDate(0, 0, -14))
	s.createSet(ctx, w1, exID, 8, 80, now.AddDate(0, 0, -14))
	w2 := s.createWorkout(ctx, now.AddDate(0, 0, -7))
	s.createSet(ctx, w2, exID, 12, 0, now.AddDate(0, 0, -7))

	r := s.workoutDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/web/workouts/%d", exID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, "data: [80,null]", "set without weight must be a gap in the weight line")
	assert.Contains(s.T(), body, "data: [8,12]", "set without weight must still show on the reps line")
}

func (s *IntegrationTestSuite) TestExerciseDetail_Chart_FirstSetPerWorkoutOnly() {
	ctx := s.Context()
	now := time.Now()

	exID := s.createExercise(ctx, "Bench Press", "barbell")
	at1 := now.AddDate(0, 0, -14)
	w1 := s.createWorkout(ctx, at1)
	s.createSet(ctx, w1, exID, 10, 60, at1)
	s.createSet(ctx, w1, exID, 8, 60, at1.Add(time.Minute))
	at2 := now.AddDate(0, 0, -7)
	w2 := s.createWorkout(ctx, at2)
	s.createSet(ctx, w2, exID, 8, 80, at2)
	s.createSet(ctx, w2, exID, 6, 80, at2.Add(time.Minute))

	r := s.workoutDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/web/workouts/%d", exID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, "data: [60,80]", "one weight point per workout, its first set")
	assert.Contains(s.T(), body, "data: [10,8]", "one reps point per workout, its first set")
	assert.Contains(s.T(), body, ">60 kg × 8</td>", "follow-up sets stay in the history table")
	assert.Contains(s.T(), body, ">80 kg × 6</td>", "follow-up sets stay in the history table")
}

func (s *IntegrationTestSuite) TestWorkoutsDashboard_ExerciseDetail_SetHistory() {
	ctx := s.Context()
	now := time.Now()

	exID := s.createExercise(ctx, "Bench Press", "barbell")
	otherExID := s.createExercise(ctx, "Squat", "barbell")

	var newest int64
	for i := 101; i >= 1; i-- {
		at := now.AddDate(0, 0, -i)
		wID := s.createWorkout(ctx, at)
		switch i {
		case 101:
			s.createSet(ctx, wID, exID, 3, 33, at)
		case 1:
			newest = wID
			s.createSet(ctx, wID, exID, 8, 80, at)
			s.createSet(ctx, wID, exID, 6, 80, at.Add(time.Minute))
			s.createSet(ctx, wID, exID, 12, 0, at.Add(2*time.Minute))
			s.createSet(ctx, wID, otherExID, 7, 77, at.Add(3*time.Minute))
		default:
			s.createSet(ctx, wID, exID, 5, 50, at)
		}
	}

	otherCtx := s.otherUserContext(ctx)
	// The other user's set references this user's exercise, so it has to go
	// before the suite truncates this user's data.
	defer func() {
		require.NoError(s.T(), s.dbMaintainer.TruncateUserData(context.Background(), gateways.UserIDFromContext(otherCtx)))
	}()
	s.createSet(otherCtx, s.createWorkout(otherCtx, now), exID, 6, 66, now)

	r := s.workoutDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/web/workouts/%d", exID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Equal(s.T(), 99, strings.Count(body, ">50 kg × 5</td>"), "every set of the newest 100 workouts is listed")
	assert.NotContains(s.T(), body, "33 kg × 3", "the 101st newest workout is outside the window")
	assert.NotContains(s.T(), body, "77 kg × 7", "another exercise's sets must not be shown")
	assert.NotContains(s.T(), body, "66 kg × 6", "another user's sets must not be shown")

	i1 := strings.Index(body, ">80 kg × 8</td>")
	i2 := strings.Index(body, ">80 kg × 6</td>")
	i3 := strings.Index(body, ">12 reps</td>")
	i4 := strings.Index(body, ">50 kg × 5</td>")
	require.NotEqual(s.T(), -1, i1)
	require.NotEqual(s.T(), -1, i2)
	require.NotEqual(s.T(), -1, i3)
	assert.Less(s.T(), i1, i2, "sets of a workout must be in logging order")
	assert.Less(s.T(), i2, i3, "sets of a workout must be in logging order")
	assert.Less(s.T(), i3, i4, "newest workout must come first")
	assert.Contains(s.T(), body, fmt.Sprintf(`<a href="/web/workouts/sessions/%d">`, newest), "rows link to their workout screen")
}

func (s *IntegrationTestSuite) TestExerciseDetail_UnknownOrForeignExercise_404s() {
	r := s.workoutDashboardRouter(s.Context())
	req := httptest.NewRequest(http.MethodGet, "/web/workouts/999999999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusNotFound, w.Code)
}

func (s *IntegrationTestSuite) TestExerciseDetail_NoSetsYet_RendersWithoutError() {
	ctx := s.Context()
	exID := s.createExercise(ctx, "Fresh Exercise", "bodyweight")

	r := s.workoutDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/web/workouts/%d", exID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), w.Body.String(), "Fresh Exercise")
	assert.NotContains(s.T(), w.Body.String(), "<table", "no sets means no history table")
}
