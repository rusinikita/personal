package tests

import (
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/progress"
	"personal/domain"
	"personal/gateways"
)

func (s *IntegrationTestSuite) TestProgressDashboard_Rendering() {
	ctx := s.Context()
	db := s.Repo()
	userID := s.UserID()
	now := time.Now()

	// Create test activities with progress
	activity1 := &domain.Activity{
		UserID:        userID,
		Name:          "Зарядка",
		ProgressType:  domain.ProgressTypeHabitProgress,
		FrequencyDays: 1,
		StartedAt:     now.AddDate(0, 0, -10),
	}
	id1, err := db.CreateActivity(ctx, activity1)
	require.NoError(s.T(), err)

	// Add some progress points
	for i := 0; i < 5; i++ {
		point := &domain.ActivityPoint{
			ActivityID: id1,
			UserID:     userID,
			Value:      2,
			ProgressAt: now.AddDate(0, 0, -i),
		}
		_, err = db.CreateProgress(ctx, point)
		require.NoError(s.T(), err)
	}

	// Setup router with DB and UserID middleware
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Add DB and UserID middleware
	contextMiddleware := func(db gateways.DB, userID int64) gin.HandlerFunc {
		return func(c *gin.Context) {
			ctx := gateways.WithDB(c.Request.Context(), db)
			ctx = gateways.WithUserID(ctx, userID)
			c.Request = c.Request.WithContext(ctx)
			c.Next()
		}
	}

	router.GET("/web/progress", contextMiddleware(db, userID), progress.DashboardWebHandler)

	// Execute
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/web/progress", nil)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Equal(s.T(), "text/html; charset=utf-8", w.Header().Get("Content-Type"))

	// Verify HTML content
	body := w.Body.String()
	assert.Contains(s.T(), body, "<!DOCTYPE html>")
	assert.Contains(s.T(), body, "<title>Progress Dashboard</title>")
	assert.Contains(s.T(), body, "Overall Streak · 30 days")
}

func (s *IntegrationTestSuite) TestProgressDashboard_ContentValidation() {
	ctx := s.Context()
	db := s.Repo()
	userID := s.UserID()
	now := time.Now()

	// Create multiple test activities with different types
	activities := []struct {
		name      string
		progType  domain.ProgressType
		frequency int
	}{
		{"Зарядка", domain.ProgressTypeHabitProgress, 1},
		{"Доволен прожитыми днями?", domain.ProgressTypeMood, 7},
		{"Trainer V2", domain.ProgressTypeProjectProgress, 7},
		{"Статья для inDrive", domain.ProgressTypePromiseState, 5},
		{"Разговаривать с мамой", domain.ProgressTypeHabitProgress, 4},
	}

	for _, act := range activities {
		activity := &domain.Activity{
			UserID:        userID,
			Name:          act.name,
			ProgressType:  act.progType,
			FrequencyDays: act.frequency,
			StartedAt:     now.AddDate(0, 0, -10),
		}
		actID, err := db.CreateActivity(ctx, activity)
		require.NoError(s.T(), err)

		// Add progress points with different values
		values := []int{2, 1, 2, 0, -1}
		for i, val := range values {
			point := &domain.ActivityPoint{
				ActivityID: actID,
				UserID:     userID,
				Value:      val,
				ProgressAt: now.AddDate(0, 0, -i),
			}
			_, err = db.CreateProgress(ctx, point)
			require.NoError(s.T(), err)
		}
	}

	// Setup router with DB and UserID middleware
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	contextMiddleware := func(db gateways.DB, userID int64) gin.HandlerFunc {
		return func(c *gin.Context) {
			ctx := gateways.WithDB(c.Request.Context(), db)
			ctx = gateways.WithUserID(ctx, userID)
			c.Request = c.Request.WithContext(ctx)
			c.Next()
		}
	}

	router.GET("/web/progress", contextMiddleware(db, userID), progress.DashboardWebHandler)

	// Execute
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/web/progress", nil)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(s.T(), http.StatusOK, w.Code)

	// Verify dashboard structure and content
	body := w.Body.String()

	// Check for header section
	assert.Contains(s.T(), body, "Overall Streak · 30 days")
	assert.Contains(s.T(), body, "streak:")
	assert.Contains(s.T(), body, "avg gap:")

	// Check for activities (top 5 by urgency)
	assert.Contains(s.T(), body, "Зарядка")
	assert.Contains(s.T(), body, "each 1d")
	assert.Contains(s.T(), body, "Доволен прожитыми днями?")
	assert.Contains(s.T(), body, "each 7d")
	assert.Contains(s.T(), body, "Trainer V2")
	assert.Contains(s.T(), body, "Статья для inDrive")
	assert.Contains(s.T(), body, "Разговаривать с мамой")

	// Check for emojis (progress indicators)
	assert.Contains(s.T(), body, "💪")  // habit_progress
	assert.Contains(s.T(), body, "☀️") // mood
	assert.Contains(s.T(), body, "🚀")  // project_progress
	assert.Contains(s.T(), body, "✅")  // promise_state

	// Check for CSS styles
	assert.Contains(s.T(), body, ".dashboard")
	assert.Contains(s.T(), body, "100vw")                     // Full viewport width
	assert.Contains(s.T(), body, "100vh")                     // Full viewport height
	assert.Contains(s.T(), body, "filter: contrast(2)")       // High contrast for e-ink
	assert.Contains(s.T(), body, "font-family: 'Noto Emoji'") // Noto Emoji font

	// Check for new template features
	assert.Contains(s.T(), body, "emoji-cell today") // IsToday class
	assert.Contains(s.T(), body, "activity-row")
}
