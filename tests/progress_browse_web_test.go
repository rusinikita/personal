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
	r.GET("/web/progress/browse/:id", progress.BrowseDetailWebHandler)
	return r
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
	require.NoError(s.T(), s.Repo().FinishActivity(ctx, finishedID, gateways.UserIDFromContext(ctx), now.AddDate(0, 0, -1)))
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
		require.NoError(s.T(), s.Repo().FinishActivity(ctx, id, userID, now.AddDate(0, 0, -1)))
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
	require.NoError(s.T(), s.Repo().FinishActivity(ctx, finishedID, userID, now.AddDate(0, 0, -1)))

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
