package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"personal/action/progress"
)

func TestProgressDashboard_Rendering(t *testing.T) {
	// Setup
	router := setupRouter()
	router.GET("/web/progress", progress.DashboardWebHandler)

	// Execute
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/web/progress", nil)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))

	// Verify HTML content
	body := w.Body.String()
	assert.Contains(t, body, "<!DOCTYPE html>")
	assert.Contains(t, body, "<title>Progress Dashboard</title>")
	assert.Contains(t, body, "Overall Streak · 30 days")
}

func TestProgressDashboard_ContentValidation(t *testing.T) {
	// Setup
	router := setupRouter()
	router.GET("/web/progress", progress.DashboardWebHandler)

	// Execute
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/web/progress", nil)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify dashboard structure and content
	body := w.Body.String()

	// Check for header section with updated template
	assert.Contains(t, body, "Overall Streak · 30 days")
	assert.Contains(t, body, "streak:")
	assert.Contains(t, body, "avg gap:")

	// Check for demo activities (from getDemoData)
	assert.Contains(t, body, "Зарядка")
	assert.Contains(t, body, "daily")
	assert.Contains(t, body, "Доволен прожитыми днями?")
	assert.Contains(t, body, "weekly")
	assert.Contains(t, body, "Trainer V2")
	assert.Contains(t, body, "Статья для inDrive")
	assert.Contains(t, body, "Разговаривать с мамой")

	// Check for emojis (progress indicators)
	assert.Contains(t, body, "💪")  // habit_progress
	assert.Contains(t, body, "☀️") // mood
	assert.Contains(t, body, "🚀")  // project_progress
	assert.Contains(t, body, "✅")  // promise_state

	// Check for CSS styles
	assert.Contains(t, body, ".dashboard")
	assert.Contains(t, body, "100vw")                     // Full viewport width
	assert.Contains(t, body, "100vh")                     // Full viewport height
	assert.Contains(t, body, "filter: contrast(2)")       // High contrast for e-ink
	assert.Contains(t, body, "font-family: 'Noto Emoji'") // Noto Emoji font

	// Check for new template features
	assert.Contains(t, body, "emoji-cell today") // IsToday class
	assert.Contains(t, body, "activity-row")
}
