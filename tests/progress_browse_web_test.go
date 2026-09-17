package tests

// Covers the "Web interface for Progress" browse view from
// docs/functions/progress-spec.md (backlog: 19-08-26). Separate, additional
// pages built on the webui design system, alongside the untouched
// screenshot dashboard covered by progress_dashboard_test.go.

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/progress"
	"personal/domain"
	"personal/gateways"
)

// browseRouter builds a minimal gin engine wired to the test suite's DB and
// user_id, serving the four browse routes the same way transport/web does.
func (s *IntegrationTestSuite) browseRouter(ctx context.Context) *gin.Engine {
	userID := gateways.UserIDFromContext(ctx)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		reqCtx := gateways.WithDB(c.Request.Context(), s.Repo())
		reqCtx = gateways.WithUserID(reqCtx, userID)
		c.Request = c.Request.WithContext(reqCtx)
		c.Next()
	})
	r.GET("/web/progress/browse", progress.BrowseWebHandler)
	r.GET("/web/progress/browse/finished", progress.BrowseFinishedWebHandler)
	r.GET("/web/progress/browse/future", progress.BrowseFutureWebHandler)
	r.GET("/web/progress/browse/paused", progress.BrowsePausedWebHandler)
	r.GET("/web/progress/browse/:id", progress.BrowseDetailWebHandler)
	r.GET("/web/progress/browse/:id/points/new", progress.BrowseNewPointWebHandler)
	r.POST("/web/progress/browse/:id/points", progress.BrowseCreatePointWebHandler)
	return r
}

// finishActivity marks an activity finished the way edit_activity does now
// that repository.FinishActivity is gone — status is the source of truth,
// ended_at is set alongside it.
func (s *IntegrationTestSuite) finishActivity(ctx context.Context, activityID, userID int64, endedAt time.Time) {
	a, err := s.Repo().GetActivity(ctx, activityID, userID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), a)
	a.Status = domain.ActivityStatusFinished
	a.EndedAt = &endedAt
	require.NoError(s.T(), s.Repo().UpdateActivity(ctx, a))
}

// withBrowsePageSize temporarily shrinks progress.BrowsePageSize so
// pagination tests don't need to seed dozens of rows, restoring the default
// afterward.
func withBrowsePageSize(t *testing.T, size int) {
	original := progress.BrowsePageSize
	progress.BrowsePageSize = size
	t.Cleanup(func() { progress.BrowsePageSize = original })
}

func (s *IntegrationTestSuite) createActivity(ctx context.Context, name string, progressType domain.ProgressType, startedAt time.Time) int64 {
	id, err := s.Repo().CreateActivity(ctx, &domain.Activity{
		UserID:        gateways.UserIDFromContext(ctx),
		Name:          name,
		ProgressType:  progressType,
		FrequencyDays: 1,
		StartedAt:     startedAt,
	})
	require.NoError(s.T(), err)
	return id
}

// --- GET /web/progress/browse ------------------------------------------------

func (s *IntegrationTestSuite) TestBrowse_ListsActiveActivities() {
	ctx := s.Context()
	now := time.Now()
	s.createActivity(ctx, "Ship personal tracker", domain.ProgressTypeProjectProgress, now.AddDate(0, 0, -10))
	s.createActivity(ctx, "Gym", domain.ProgressTypeHabitProgress, now.AddDate(0, 0, -30))

	r := s.browseRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/progress/browse", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, "Ship personal tracker")
	assert.Contains(s.T(), body, "Gym")
	assert.Contains(s.T(), body, `href="/web/progress/browse/`, "rows must link to the drill-down")
}

func (s *IntegrationTestSuite) TestBrowse_ExcludesFinishedAndFutureActivities() {
	ctx := s.Context()
	now := time.Now()

	s.createActivity(ctx, "Active one", domain.ProgressTypeProjectProgress, now.AddDate(0, 0, -5))
	finishedID := s.createActivity(ctx, "Finished one", domain.ProgressTypeProjectProgress, now.AddDate(0, 0, -20))
	s.finishActivity(ctx, finishedID, gateways.UserIDFromContext(ctx), now.AddDate(0, 0, -1))
	s.createActivity(ctx, "Future one", domain.ProgressTypeProjectProgress, now.AddDate(0, 0, 5))

	r := s.browseRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/progress/browse", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(s.T(), body, "Active one")
	assert.NotContains(s.T(), body, "Finished one")
	assert.NotContains(s.T(), body, "Future one")
}

func (s *IntegrationTestSuite) TestBrowse_ShowsActivityDescription() {
	ctx := s.Context()
	now := time.Now()
	_, err := s.Repo().CreateActivity(ctx, &domain.Activity{
		UserID:        gateways.UserIDFromContext(ctx),
		Name:          "Ship personal tracker",
		Description:   "Track weekly progress on the side project",
		ProgressType:  domain.ProgressTypeProjectProgress,
		FrequencyDays: 1,
		StartedAt:     now.AddDate(0, 0, -10),
	})
	require.NoError(s.T(), err)

	r := s.browseRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/progress/browse", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Contains(s.T(), w.Body.String(), "Track weekly progress on the side project")
}

func (s *IntegrationTestSuite) TestBrowse_HasCrossLinksToFinishedAndFuture() {
	r := s.browseRouter(s.Context())
	req := httptest.NewRequest(http.MethodGet, "/web/progress/browse", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(s.T(), body, `href="/web/progress/browse/finished"`)
	assert.Contains(s.T(), body, `href="/web/progress/browse/future"`)
	assert.Contains(s.T(), body, `href="/web/progress/browse/paused"`)
}

// TestBrowse_SplitsActiveActivitiesByProgressTypeSections covers the
// four-section active list (Habits, Promises, Projects, Mood, in that
// order) from progress-spec.md: each section is headed by its type name,
// has no Type column (redundant with the heading), and is unpaginated.
func (s *IntegrationTestSuite) TestBrowse_SplitsActiveActivitiesByProgressTypeSections() {
	ctx := s.Context()
	now := time.Now()
	s.createActivity(ctx, "Gym", domain.ProgressTypeHabitProgress, now.AddDate(0, 0, -30))
	s.createActivity(ctx, "Call mom", domain.ProgressTypePromiseState, now.AddDate(0, 0, -10))
	s.createActivity(ctx, "Ship personal tracker", domain.ProgressTypeProjectProgress, now.AddDate(0, 0, -14))
	s.createActivity(ctx, "Daily mood", domain.ProgressTypeMood, now.AddDate(0, 0, -1))

	r := s.browseRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/progress/browse", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	habitsIdx := strings.Index(body, "<h3>Habits</h3>")
	promisesIdx := strings.Index(body, "<h3>Promises</h3>")
	projectsIdx := strings.Index(body, "<h3>Projects</h3>")
	moodIdx := strings.Index(body, "<h3>Mood</h3>")
	require.True(s.T(), habitsIdx >= 0 && promisesIdx >= 0 && projectsIdx >= 0 && moodIdx >= 0, "all four section headings must render")
	assert.True(s.T(), habitsIdx < promisesIdx && promisesIdx < projectsIdx && projectsIdx < moodIdx, "sections must render in Habits, Promises, Projects, Mood order")

	assert.Contains(s.T(), body, "Gym")
	assert.Contains(s.T(), body, "Call mom")
	assert.Contains(s.T(), body, "Ship personal tracker")
	assert.Contains(s.T(), body, "Daily mood")
	assert.NotContains(s.T(), body, ">Type</th>", "Type column must not appear on the active list")
}

// TestBrowse_ActiveSectionRendersEmptyTableWhenNoActivitiesOfThatType covers
// the "still show the heading" rule for a progress_type with zero active
// activities.
func (s *IntegrationTestSuite) TestBrowse_ActiveSectionRendersEmptyTableWhenNoActivitiesOfThatType() {
	ctx := s.Context()
	now := time.Now()
	s.createActivity(ctx, "Gym", domain.ProgressTypeHabitProgress, now.AddDate(0, 0, -30))

	r := s.browseRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/progress/browse", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(s.T(), body, "<h3>Promises</h3>")
	assert.Contains(s.T(), body, "<h3>Projects</h3>")
	assert.Contains(s.T(), body, "<h3>Mood</h3>")
}

// TestBrowse_ActiveListIsNotPaginated covers the "active list shows
// everything" decision (unlike finished/future, which stay paginated).
func (s *IntegrationTestSuite) TestBrowse_ActiveListIsNotPaginated() {
	withBrowsePageSize(s.T(), 2)
	ctx := s.Context()
	now := time.Now()
	s.createActivity(ctx, "Habit A", domain.ProgressTypeHabitProgress, now.AddDate(0, 0, -1))
	s.createActivity(ctx, "Habit B", domain.ProgressTypeHabitProgress, now.AddDate(0, 0, -2))
	s.createActivity(ctx, "Habit C", domain.ProgressTypeHabitProgress, now.AddDate(0, 0, -3))

	r := s.browseRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/progress/browse", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(s.T(), body, "Habit A")
	assert.Contains(s.T(), body, "Habit B")
	assert.Contains(s.T(), body, "Habit C")
	assert.NotContains(s.T(), body, "Page 1 of")
}

func (s *IntegrationTestSuite) TestBrowseFinished_Pagination() {
	withBrowsePageSize(s.T(), 2)
	ctx := s.Context()
	now := time.Now()
	userID := gateways.UserIDFromContext(ctx)
	for _, name := range []string{"Activity A", "Activity B", "Activity C"} {
		id := s.createActivity(ctx, name, domain.ProgressTypeProjectProgress, now.AddDate(0, 0, -10))
		s.finishActivity(ctx, id, userID, now.AddDate(0, 0, -1))
	}

	r := s.browseRouter(ctx)

	// Page 1: two rows, a next link, no prev link.
	req := httptest.NewRequest(http.MethodGet, "/web/progress/browse/finished", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	body := w.Body.String()
	assert.Contains(s.T(), body, "Page 1 of 2")
	assert.Contains(s.T(), body, `href="/web/progress/browse/finished?page=2"`)
	assert.NotContains(s.T(), body, "page=0")

	// Page 2: remaining row, a prev link, no next link.
	req = httptest.NewRequest(http.MethodGet, "/web/progress/browse/finished?page=2", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	body = w.Body.String()
	assert.Contains(s.T(), body, "Page 2 of 2")
	assert.Contains(s.T(), body, `href="/web/progress/browse/finished?page=1"`)
	assert.NotContains(s.T(), body, "Next →</a>")
}

// --- GET /web/progress/browse/finished --------------------------------------

func (s *IntegrationTestSuite) TestBrowseFinished_ListsOnlyFinishedActivities() {
	ctx := s.Context()
	now := time.Now()
	userID := gateways.UserIDFromContext(ctx)

	s.createActivity(ctx, "Still going", domain.ProgressTypeProjectProgress, now.AddDate(0, 0, -5))
	finishedID := s.createActivity(ctx, "Wrapped up", domain.ProgressTypeProjectProgress, now.AddDate(0, 0, -20))
	s.finishActivity(ctx, finishedID, userID, now.AddDate(0, 0, -1))

	r := s.browseRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/progress/browse/finished", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, "Wrapped up")
	assert.NotContains(s.T(), body, "Still going")
}

// --- GET /web/progress/browse/future -----------------------------------------

func (s *IntegrationTestSuite) TestBrowseFuture_ListsOnlyNotYetStartedActivities() {
	ctx := s.Context()
	now := time.Now()

	s.createActivity(ctx, "Already running", domain.ProgressTypeProjectProgress, now.AddDate(0, 0, -5))
	s.createActivity(ctx, "Starts next week", domain.ProgressTypeProjectProgress, now.AddDate(0, 0, 7))

	r := s.browseRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/progress/browse/future", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, "Starts next week")
	assert.NotContains(s.T(), body, "Already running")
}

// --- GET /web/progress/browse/paused -----------------------------------------

func (s *IntegrationTestSuite) TestBrowsePaused_ListsOnlyPausedActivitiesWithDeferredUntilColumn() {
	ctx := s.Context()
	db := s.Repo()
	userID := gateways.UserIDFromContext(ctx)
	now := time.Now()

	s.createActivity(ctx, "Still going", domain.ProgressTypeHabitProgress, now.AddDate(0, 0, -5))

	pausedID := s.createActivity(ctx, "On hold", domain.ProgressTypeHabitProgress, now.AddDate(0, 0, -20))
	a, err := db.GetActivity(ctx, pausedID, userID)
	require.NoError(s.T(), err)
	deferredUntil := now.AddDate(0, 0, 14)
	a.Status = domain.ActivityStatusPaused
	a.DeferredUntil = &deferredUntil
	require.NoError(s.T(), db.UpdateActivity(ctx, a))

	r := s.browseRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/progress/browse/paused", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, "On hold")
	assert.NotContains(s.T(), body, "Still going")
	assert.Contains(s.T(), body, "Deferred until")
	assert.Contains(s.T(), body, deferredUntil.Format("2006-01-02"))
}

// --- GET /web/progress/browse/{id} -------------------------------------------

func (s *IntegrationTestSuite) TestBrowseDetail_ShowsHistoryNoteAndChart() {
	ctx := s.Context()
	now := time.Now()
	userID := gateways.UserIDFromContext(ctx)

	activityID := s.createActivity(ctx, "Architecture strategy", domain.ProgressTypeProjectProgress, now.AddDate(0, 0, -30))
	_, err := s.Repo().CreateProgress(ctx, &domain.ActivityPoint{
		ActivityID: activityID, UserID: userID, Value: 2, Note: "major breakthrough", ProgressAt: now.AddDate(0, 0, -1),
	})
	require.NoError(s.T(), err)

	r := s.browseRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/web/progress/browse/%d", activityID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, "Architecture strategy")
	assert.Contains(s.T(), body, "major breakthrough")
	assert.Contains(s.T(), body, "<canvas id=", "must render a value-over-time chart")
	assert.Contains(s.T(), body, `href="/web/progress/browse"`, "must have a back link")
}

func (s *IntegrationTestSuite) TestBrowseDetail_ShowsActivityDescription() {
	ctx := s.Context()
	now := time.Now()
	activityID, err := s.Repo().CreateActivity(ctx, &domain.Activity{
		UserID:        gateways.UserIDFromContext(ctx),
		Name:          "Architecture strategy",
		Description:   "Long-term plan for the service split",
		ProgressType:  domain.ProgressTypeProjectProgress,
		FrequencyDays: 1,
		StartedAt:     now.AddDate(0, 0, -30),
	})
	require.NoError(s.T(), err)

	r := s.browseRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/web/progress/browse/%d", activityID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Contains(s.T(), w.Body.String(), "Long-term plan for the service split")
}

func (s *IntegrationTestSuite) TestBrowseDetail_UnknownOrForeignActivity_404s() {
	r := s.browseRouter(s.Context())
	req := httptest.NewRequest(http.MethodGet, "/web/progress/browse/999999999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusNotFound, w.Code)
}

func (s *IntegrationTestSuite) TestBrowseDetail_ChartShowsFullHistoryDespitePagination() {
	withBrowsePageSize(s.T(), 2)
	ctx := s.Context()
	now := time.Now()
	userID := gateways.UserIDFromContext(ctx)

	activityID := s.createActivity(ctx, "Daily mood", domain.ProgressTypeMood, now.AddDate(0, 0, -10))
	for i := 0; i < 5; i++ {
		_, err := s.Repo().CreateProgress(ctx, &domain.ActivityPoint{
			ActivityID: activityID, UserID: userID, Value: 1, Note: fmt.Sprintf("note-%d", i), ProgressAt: now.AddDate(0, 0, -i),
		})
		require.NoError(s.T(), err)
	}

	r := s.browseRouter(ctx)

	// The chart's embedded label/value arrays must carry all 5 points even
	// though the history table's page size is 2.
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/web/progress/browse/%d", activityID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	body := w.Body.String()
	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), body, `data: [1,1,1,1,1]`, "chart series must include all 5 points, not just the current table page")

	// History table page 1 only shows its own 2 rows.
	assert.Contains(s.T(), body, "note-0")
	assert.Contains(s.T(), body, "note-1")
	assert.NotContains(s.T(), body, "note-2")
	assert.Contains(s.T(), body, "Page 1 of 3")

	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/web/progress/browse/%d?page=3", activityID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Contains(s.T(), w.Body.String(), "Page 3 of 3")
}

// --- misc ---------------------------------------------------------------------

func (s *IntegrationTestSuite) TestBrowse_ShowsTrendStatTilesInDetail() {
	ctx := s.Context()
	now := time.Now()
	userID := gateways.UserIDFromContext(ctx)

	activityID := s.createActivity(ctx, "Call mom", domain.ProgressTypePromiseState, now.AddDate(0, 0, -10))
	_, err := s.Repo().CreateProgress(ctx, &domain.ActivityPoint{
		ActivityID: activityID, UserID: userID, Value: 1, ProgressAt: now,
	})
	require.NoError(s.T(), err)

	r := s.browseRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/web/progress/browse/%d", activityID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(s.T(), body, "Overall")
	assert.Contains(s.T(), body, "Last 30 days")
	assert.Contains(s.T(), body, "Last 7 days")
	assert.Contains(s.T(), body, "webui-stat-tile")
}

// TestBrowseDetail_HasAddPointButtonLinkingToStandalonePage covers the
// drill-down page's "+ Add" button, which links to the standalone
// log-a-point page instead of embedding the form inline.
func (s *IntegrationTestSuite) TestBrowseDetail_HasAddPointButtonLinkingToStandalonePage() {
	ctx := s.Context()
	now := time.Now()
	activityID := s.createActivity(ctx, "Gym", domain.ProgressTypeHabitProgress, now.AddDate(0, 0, -5))

	r := s.browseRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/web/progress/browse/%d", activityID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(s.T(), body, fmt.Sprintf(`href="/web/progress/browse/%d/points/new"`, activityID))
	assert.Contains(s.T(), body, "+ Add")
	assert.NotContains(s.T(), body, `name="value"`, "the form must no longer be embedded on the drill-down page")
}

// --- GET /web/progress/browse/{id}/points/new --------------------------------

// TestBrowseNewPoint_ShowsWhichActivityAndEmojiRadiosPerProgressType covers
// the standalone page's header (naming the activity) and its radio group:
// 5 options for mood/habit_progress/project_progress, 3 for promise_state
// (no ±2 in that domain).
func (s *IntegrationTestSuite) TestBrowseNewPoint_ShowsWhichActivityAndEmojiRadiosPerProgressType() {
	ctx := s.Context()
	now := time.Now()

	moodID := s.createActivity(ctx, "Daily mood", domain.ProgressTypeMood, now.AddDate(0, 0, -5))
	promiseID := s.createActivity(ctx, "Call mom", domain.ProgressTypePromiseState, now.AddDate(0, 0, -5))

	r := s.browseRouter(ctx)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/web/progress/browse/%d/points/new", moodID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	body := w.Body.String()
	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), body, "Daily mood", "header must name which activity the point is for")
	assert.Contains(s.T(), body, `action="/web/progress/browse/`+fmt.Sprint(moodID)+`/points"`)
	assert.Contains(s.T(), body, fmt.Sprintf(`href="/web/progress/browse/%d"`, moodID), "must link back to the drill-down page")
	assert.Equal(s.T(), 5, strings.Count(body, `name="value"`), "mood must offer 5 radio options")
	assert.Contains(s.T(), body, "☀️")

	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/web/progress/browse/%d/points/new", promiseID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	body = w.Body.String()
	assert.Contains(s.T(), body, "Call mom")
	assert.Equal(s.T(), 3, strings.Count(body, `name="value"`), "promise_state must offer only 3 radio options (no ±2)")
}

func (s *IntegrationTestSuite) TestBrowseNewPoint_UnknownOrForeignActivity_404s() {
	r := s.browseRouter(s.Context())
	req := httptest.NewRequest(http.MethodGet, "/web/progress/browse/999999999/points/new", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusNotFound, w.Code)
}

// --- POST /web/progress/browse/{id}/points -----------------------------------

func (s *IntegrationTestSuite) TestBrowseCreatePoint_LogsPointAndRedirectsToDetail() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	now := time.Now()
	activityID := s.createActivity(ctx, "Gym", domain.ProgressTypeHabitProgress, now.AddDate(0, 0, -5))

	r := s.browseRouter(ctx)
	form := url.Values{"value": {"2"}, "note": {"crushed it"}}
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/web/progress/browse/%d/points", activityID), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusFound, w.Code)
	assert.Equal(s.T(), fmt.Sprintf("/web/progress/browse/%d", activityID), w.Header().Get("Location"))

	points, err := s.Repo().ListProgress(ctx, domain.ProgressFilter{UserID: userID, ActivityID: activityID})
	require.NoError(s.T(), err)
	require.Len(s.T(), points, 1)
	assert.Equal(s.T(), 2, points[0].Value)
	assert.Equal(s.T(), "crushed it", points[0].Note)
}

func (s *IntegrationTestSuite) TestBrowseCreatePoint_RejectsOutOfRangeValue() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	now := time.Now()
	activityID := s.createActivity(ctx, "Gym", domain.ProgressTypeHabitProgress, now.AddDate(0, 0, -5))

	r := s.browseRouter(ctx)
	form := url.Values{"value": {"5"}}
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/web/progress/browse/%d/points", activityID), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code, "invalid input re-renders the detail page instead of redirecting")
	assert.Contains(s.T(), w.Body.String(), "value must be between -2 and")

	points, err := s.Repo().ListProgress(ctx, domain.ProgressFilter{UserID: userID, ActivityID: activityID})
	require.NoError(s.T(), err)
	assert.Empty(s.T(), points, "no point must be created on validation failure")
}

func (s *IntegrationTestSuite) TestBrowseCreatePoint_UnknownOrForeignActivity_ShowsError() {
	r := s.browseRouter(s.Context())
	form := url.Values{"value": {"1"}}
	req := httptest.NewRequest(http.MethodPost, "/web/progress/browse/999999999/points", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusNotFound, w.Code, "renderNewPointPage 404s the same way the GET handler does for an unowned/missing activity")
	assert.Contains(s.T(), w.Body.String(), "activity not found")
}

func (s *IntegrationTestSuite) TestBrowseCreatePoint_AcceptsBackdatedProgressAt() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	now := time.Now()
	activityID := s.createActivity(ctx, "Gym", domain.ProgressTypeHabitProgress, now.AddDate(0, 0, -5))

	r := s.browseRouter(ctx)
	form := url.Values{"value": {"1"}, "progress_at": {"2026-01-02T15:04"}}
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/web/progress/browse/%d/points", activityID), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusFound, w.Code)

	points, err := s.Repo().ListProgress(ctx, domain.ProgressFilter{UserID: userID, ActivityID: activityID})
	require.NoError(s.T(), err)
	require.Len(s.T(), points, 1)
	assert.Equal(s.T(), "2026-01-02 15:04", points[0].ProgressAt.Format("2006-01-02 15:04"))
}
