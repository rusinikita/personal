package tests

// Covers the workout sessions web pages from docs/functions/workout-spec.md
// (backlog: 21-09-26): the history of the last 10 workouts, the "new"
// workout screen that creates nothing, and the set-adding form that creates
// the workout lazily on its first set.

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/workout"
	"personal/domain"
	"personal/gateways"
)

// workoutSessionsRouter serves the sessions routes the same way
// transport/web does, next to the records routes they share a prefix with.
func (s *IntegrationTestSuite) workoutSessionsRouter(ctx context.Context) *gin.Engine {
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
	r.GET("/web/workouts/sessions", workout.SessionsWebHandler)
	r.GET("/web/workouts/sessions/new", workout.NewSessionWebHandler)
	r.POST("/web/workouts/sessions/new/sets", workout.CreateSessionSetWebHandler)
	r.GET("/web/workouts/sessions/:id", workout.SessionWebHandler)
	r.POST("/web/workouts/sessions/:id/sets", workout.AddSessionSetWebHandler)
	r.GET("/web/workouts/:id", workout.ExerciseDetailWebHandler)
	return r
}

func (s *IntegrationTestSuite) getSessionsPage(ctx context.Context, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	s.workoutSessionsRouter(ctx).ServeHTTP(w, req)
	return w
}

func (s *IntegrationTestSuite) postSetForm(ctx context.Context, path string, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	s.workoutSessionsRouter(ctx).ServeHTTP(w, req)
	return w
}

func (s *IntegrationTestSuite) listUserWorkouts(ctx context.Context) []domain.Workout {
	workouts, err := s.Repo().ListWorkouts(ctx, gateways.UserIDFromContext(ctx))
	require.NoError(s.T(), err)
	return workouts
}

// workoutSets returns the sets of one workout from the last day.
func (s *IntegrationTestSuite) workoutSets(ctx context.Context, workoutID int64) []domain.Set {
	all, err := s.Repo().ListSets(ctx, gateways.UserIDFromContext(ctx), time.Now().Add(-24*time.Hour), time.Now().Add(time.Minute))
	require.NoError(s.T(), err)
	var sets []domain.Set
	for _, set := range all {
		if set.WorkoutID == workoutID {
			sets = append(sets, set)
		}
	}
	return sets
}

// --- GET /web/workouts/sessions ----------------------------------------------

func (s *IntegrationTestSuite) TestWorkoutSessions_History_Empty() {
	w := s.getSessionsPage(s.Context(), "/web/workouts/sessions")

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), w.Body.String(), "No workouts yet")
	assert.Contains(s.T(), w.Body.String(), `href="/web/workouts/sessions/new"`)
}

func (s *IntegrationTestSuite) TestWorkoutSessions_History() {
	ctx := s.Context()
	benchID := s.createExercise(ctx, "Bench Press", "barbell")
	squatID := s.createExercise(ctx, "Squat", "barbell")

	noonToday := time.Now().UTC().Truncate(24 * time.Hour).Add(12 * time.Hour)
	var dates []string
	for i := 11; i >= 1; i-- {
		at := noonToday.AddDate(0, 0, -i)
		wID := s.createWorkout(ctx, at)
		s.createSet(ctx, wID, benchID, 5, float64(100+i), at)
		s.createSet(ctx, wID, benchID, 5, float64(200+i), at.Add(time.Minute))
		s.createSet(ctx, wID, squatID, 3, float64(300+i), at.Add(2*time.Minute))
		dates = append(dates, at.Format("2006-01-02"))
	}

	w := s.getSessionsPage(ctx, "/web/workouts/sessions")

	require.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.NotContains(s.T(), body, dates[0], "11th newest workout must not be shown")
	prev := -1
	for i := len(dates) - 1; i >= 1; i-- {
		idx := strings.Index(body, dates[i])
		require.NotEqual(s.T(), -1, idx, "workout %s must be shown", dates[i])
		assert.Greater(s.T(), idx, prev, "workouts must be listed newest first")
		prev = idx
	}
	assert.Contains(s.T(), body, "Bench Press — 101 kg × 5")
	assert.Contains(s.T(), body, "Squat — 301 kg × 3")
	assert.NotContains(s.T(), body, "201 kg × 5", "only the first set of each exercise is shown")
}

// --- GET /web/workouts/sessions/new ------------------------------------------

func (s *IntegrationTestSuite) TestWorkoutSessions_NewScreen_CreatesNothing() {
	ctx := s.Context()
	s.createExercise(ctx, "Bench Press", "barbell")

	w := s.getSessionsPage(ctx, "/web/workouts/sessions/new")

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), w.Body.String(), `action="/web/workouts/sessions/new/sets"`)
	assert.Contains(s.T(), w.Body.String(), "Bench Press")
	assert.Contains(s.T(), w.Body.String(), `inputmode="numeric" value=""`, "new screen has nothing to prefill")
	assert.Empty(s.T(), s.listUserWorkouts(ctx))
}

func (s *IntegrationTestSuite) TestWorkoutSessions_NewScreen_ExercisesSortedByUsage() {
	ctx := s.Context()
	s.createExercise(ctx, "Never", "machine")
	rareID := s.createExercise(ctx, "Rare", "machine")
	oftenID := s.createExercise(ctx, "Often", "machine")

	at := time.Now().Add(-time.Hour)
	wID := s.createWorkout(ctx, at)
	s.createSet(ctx, wID, rareID, 10, 20, at)
	for i := 0; i < 3; i++ {
		s.createSet(ctx, wID, oftenID, 10, 20, at)
	}

	body := s.getSessionsPage(ctx, "/web/workouts/sessions/new").Body.String()

	iOften := strings.Index(body, ">Often —")
	iRare := strings.Index(body, ">Rare —")
	iNever := strings.Index(body, ">Never<")
	require.NotEqual(s.T(), -1, iOften)
	require.NotEqual(s.T(), -1, iRare)
	require.NotEqual(s.T(), -1, iNever)
	assert.Less(s.T(), iOften, iRare)
	assert.Less(s.T(), iRare, iNever)
}

func (s *IntegrationTestSuite) TestWorkoutSessions_Selector_ShowsPreviousWorkoutFirstSet() {
	ctx := s.Context()
	alphaID := s.createExercise(ctx, "Alpha", "barbell")
	betaID := s.createExercise(ctx, "Beta", "bodyweight")
	gammaID := s.createExercise(ctx, "Gamma", "machine")
	deltaID := s.createExercise(ctx, "Delta", "machine")
	fillerID := s.createExercise(ctx, "Filler", "machine")

	now := time.Now()
	// Delta was only done in a workout older than the last 10.
	deltaAt := now.AddDate(0, 0, -20)
	s.createSet(ctx, s.createWorkout(ctx, deltaAt), deltaID, 10, 40, deltaAt)
	for i := 15; i >= 8; i-- {
		at := now.AddDate(0, 0, -i)
		s.createSet(ctx, s.createWorkout(ctx, at), fillerID, 10, 20, at)
	}
	olderAt := now.AddDate(0, 0, -2)
	older := s.createWorkout(ctx, olderAt)
	s.createSet(ctx, older, alphaID, 10, 60, olderAt)
	newerAt := now.AddDate(0, 0, -1)
	newer := s.createWorkout(ctx, newerAt)
	s.createSet(ctx, newer, alphaID, 8, 80, newerAt)
	s.createSet(ctx, newer, alphaID, 6, 80, newerAt.Add(time.Minute))
	s.createSet(ctx, newer, betaID, 12, 0, newerAt.Add(2*time.Minute))

	otherCtx := s.otherUserContext(ctx)
	// The other user's set references this user's exercise, so it has to go
	// before the suite truncates this user's data.
	defer func() {
		require.NoError(s.T(), s.dbMaintainer.TruncateUserData(context.Background(), gateways.UserIDFromContext(otherCtx)))
	}()
	otherAt := now.Add(-time.Hour)
	s.createSet(otherCtx, s.createWorkout(otherCtx, otherAt), gammaID, 9, 99, otherAt)

	body := s.getSessionsPage(ctx, "/web/workouts/sessions/new").Body.String()
	assert.Contains(s.T(), body, ">Alpha — 80 kg × 8</option>", "first set of the latest workout with the exercise")
	assert.Contains(s.T(), body, ">Beta — 12 reps</option>")
	assert.Contains(s.T(), body, ">Gamma</option>", "never logged by this user")
	assert.Contains(s.T(), body, ">Delta</option>", "not done in the last 10 workouts")

	body = s.getSessionsPage(ctx, fmt.Sprintf("/web/workouts/sessions/%d", newer)).Body.String()
	assert.Contains(s.T(), body, ">Alpha — 60 kg × 10</option>", "the workout on screen is excluded")
	assert.Contains(s.T(), body, ">Beta</option>", "only logged in the workout on screen")
}

// --- POST /web/workouts/sessions/new/sets ------------------------------------

func (s *IntegrationTestSuite) TestWorkoutSessions_FirstSet_CreatesWorkoutAndRedirects() {
	ctx := s.Context()
	exID := s.createExercise(ctx, "Bench Press", "barbell")

	// An open workout from 30 minutes ago: log_workout_set would reuse it,
	// an explicit "New workout" must close it and start a new one.
	oldAt := time.Now().Add(-30 * time.Minute)
	oldID := s.createWorkout(ctx, oldAt)
	s.createSet(ctx, oldID, exID, 5, 60, oldAt)

	w := s.postSetForm(ctx, "/web/workouts/sessions/new/sets", url.Values{
		"exercise_id": {fmt.Sprint(exID)}, "weight_kg": {"80"}, "reps": {"8"},
	})

	require.Equal(s.T(), http.StatusFound, w.Code)
	workouts := s.listUserWorkouts(ctx)
	require.Len(s.T(), workouts, 2)
	newWorkout := workouts[0]
	assert.NotEqual(s.T(), oldID, newWorkout.ID)
	assert.Equal(s.T(), fmt.Sprintf("/web/workouts/sessions/%d", newWorkout.ID), w.Header().Get("Location"))

	sets := s.workoutSets(ctx, newWorkout.ID)
	require.Len(s.T(), sets, 1)
	assert.Equal(s.T(), int64(8), sets[0].Reps)
	assert.Equal(s.T(), 80.0, sets[0].WeightKg)
	assert.Equal(s.T(), exID, sets[0].ExerciseID)

	assert.NotNil(s.T(), workouts[1].CompletedAt, "previous open workout must be closed")
}

// --- POST /web/workouts/sessions/:id/sets ------------------------------------

func (s *IntegrationTestSuite) TestWorkoutSessions_AddSet_ToExistingWorkout() {
	ctx := s.Context()
	exID := s.createExercise(ctx, "Bench Press", "barbell")
	at := time.Now().Add(-10 * time.Minute)
	wID := s.createWorkout(ctx, at)
	s.createSet(ctx, wID, exID, 8, 80, at)

	w := s.postSetForm(ctx, fmt.Sprintf("/web/workouts/sessions/%d/sets", wID), url.Values{
		"exercise_id": {fmt.Sprint(exID)}, "weight_kg": {"85"}, "reps": {"6"},
	})

	require.Equal(s.T(), http.StatusFound, w.Code)
	assert.Equal(s.T(), fmt.Sprintf("/web/workouts/sessions/%d", wID), w.Header().Get("Location"))
	assert.Len(s.T(), s.listUserWorkouts(ctx), 1, "no new workout must be created")
	assert.Len(s.T(), s.workoutSets(ctx, wID), 2)
}

func (s *IntegrationTestSuite) TestWorkoutSessions_Validation() {
	ctx := s.Context()
	exID := s.createExercise(ctx, "Bench Press", "barbell")
	at := time.Now().Add(-10 * time.Minute)
	wID := s.createWorkout(ctx, at)
	foreignExID := s.createExercise(s.otherUserContext(ctx), "Foreign", "barbell")

	ex := fmt.Sprint(exID)
	tests := []struct {
		name    string
		form    url.Values
		wantErr string
	}{
		{"missing exercise", url.Values{"reps": {"8"}}, "exercise is required"},
		{"missing reps", url.Values{"exercise_id": {ex}, "weight_kg": {"80"}}, "reps must be a positive whole number"},
		{"non-numeric reps", url.Values{"exercise_id": {ex}, "reps": {"abc"}}, "reps must be a positive whole number"},
		{"non-numeric weight", url.Values{"exercise_id": {ex}, "reps": {"8"}, "weight_kg": {"x"}}, "weight must be a non-negative number"},
		{"foreign exercise", url.Values{"exercise_id": {fmt.Sprint(foreignExID)}, "reps": {"8"}}, "exercise not found"},
	}
	paths := []string{"/web/workouts/sessions/new/sets", fmt.Sprintf("/web/workouts/sessions/%d/sets", wID)}

	for _, tt := range tests {
		for _, path := range paths {
			s.Run(tt.name+" "+path, func() {
				w := s.postSetForm(ctx, path, tt.form)

				assert.Equal(s.T(), http.StatusOK, w.Code)
				assert.Contains(s.T(), w.Body.String(), tt.wantErr)
				assert.Contains(s.T(), w.Body.String(), `<form method="POST"`)
				assert.Len(s.T(), s.listUserWorkouts(ctx), 1, "no workout must be created")
				assert.Empty(s.T(), s.workoutSets(ctx, wID), "no set must be created")
			})
		}
	}
}

// --- GET /web/workouts/sessions/:id ------------------------------------------

func (s *IntegrationTestSuite) TestWorkoutSessions_WorkoutScreen() {
	ctx := s.Context()
	benchID := s.createExercise(ctx, "Bench Press", "barbell")
	squatID := s.createExercise(ctx, "Squat", "barbell")

	at1 := time.Now().Add(-2 * time.Hour)
	w1 := s.createWorkout(ctx, at1)
	s.createSet(ctx, w1, benchID, 6, 70, at1)
	s.createSet(ctx, w1, benchID, 5, 72.5, at1.Add(time.Minute))
	at2 := time.Now().Add(-time.Hour)
	w2 := s.createWorkout(ctx, at2)
	s.createSet(ctx, w2, squatID, 3, 90, at2)

	w := s.getSessionsPage(ctx, fmt.Sprintf("/web/workouts/sessions/%d", w1))

	require.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, fmt.Sprintf(`action="/web/workouts/sessions/%d/sets"`, w1))
	assert.Contains(s.T(), body, fmt.Sprintf(`<option value="%d" selected>`, benchID))
	assert.NotContains(s.T(), body, fmt.Sprintf(`<option value="%d" selected>`, squatID))
	assert.Contains(s.T(), body, `inputmode="decimal" value="72.5"`, "weight prefilled from the workout's last set")
	assert.Contains(s.T(), body, `inputmode="numeric" value="5"`, "reps prefilled from the workout's last set")
	i1 := strings.Index(body, "70 kg × 6")
	i2 := strings.Index(body, "72.5 kg × 5")
	require.NotEqual(s.T(), -1, i1)
	require.NotEqual(s.T(), -1, i2)
	assert.Less(s.T(), i1, i2, "sets must be listed in logging order")
	assert.NotContains(s.T(), body, ">90 kg × 3</td>", "sets of another workout must not be shown")
}

func (s *IntegrationTestSuite) TestWorkoutSessions_UnknownOrForeignWorkout_404s() {
	ctx := s.Context()
	exID := s.createExercise(ctx, "Bench Press", "barbell")
	foreignID := s.createWorkout(s.otherUserContext(ctx), time.Now())
	form := url.Values{"exercise_id": {fmt.Sprint(exID)}, "reps": {"8"}}

	tests := []struct {
		name string
		do   func() *httptest.ResponseRecorder
	}{
		{"GET unknown", func() *httptest.ResponseRecorder {
			return s.getSessionsPage(ctx, "/web/workouts/sessions/999999999")
		}},
		{"GET foreign", func() *httptest.ResponseRecorder {
			return s.getSessionsPage(ctx, fmt.Sprintf("/web/workouts/sessions/%d", foreignID))
		}},
		{"POST unknown", func() *httptest.ResponseRecorder {
			return s.postSetForm(ctx, "/web/workouts/sessions/999999999/sets", form)
		}},
		{"POST foreign", func() *httptest.ResponseRecorder {
			return s.postSetForm(ctx, fmt.Sprintf("/web/workouts/sessions/%d/sets", foreignID), form)
		}},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			assert.Equal(s.T(), http.StatusNotFound, tt.do().Code)
		})
	}
	assert.Empty(s.T(), s.listUserWorkouts(ctx))
}

// --- cross-links -------------------------------------------------------------

func (s *IntegrationTestSuite) TestWorkoutSessions_CrossLinks() {
	ctx := s.Context()
	for _, path := range []string{"/web/workouts", "/web/workouts/sessions"} {
		s.Run(path, func() {
			body := s.getSessionsPage(ctx, path).Body.String()
			assert.Contains(s.T(), body, `<a href="/web/workouts">Personal records</a>`)
			assert.Contains(s.T(), body, `<a href="/web/workouts/sessions">Sessions</a>`)
		})
	}
}
