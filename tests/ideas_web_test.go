package tests

// Covers GET /web/ideas and POST /web/ideas from docs/functions/ideas-spec.md.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/ideas"
	"personal/domain"
	"personal/gateways"
)

func (s *IntegrationTestSuite) ideasRouter(ctx context.Context) *gin.Engine {
	userID := gateways.UserIDFromContext(ctx)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		reqCtx := gateways.WithDB(c.Request.Context(), s.Repo())
		reqCtx = gateways.WithUserID(reqCtx, userID)
		c.Request = c.Request.WithContext(reqCtx)
		c.Next()
	})
	r.GET("/web/ideas", ideas.IdeasWebHandler)
	r.POST("/web/ideas", ideas.CreateIdeaWebHandler)
	return r
}

func (s *IntegrationTestSuite) postIdea(r *gin.Engine, body string) *httptest.ResponseRecorder {
	form := url.Values{"body": {body}}
	req := httptest.NewRequest(http.MethodPost, "/web/ideas", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func (s *IntegrationTestSuite) TestIdeasWeb() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	now := time.Now()
	r := s.ideasRouter(ctx)

	s.Run("GET shows open ideas grouped by status and the blocked ones", func() {
		s.createIdea(ctx, "inbox thought", now.Add(-4*time.Hour))
		someday := s.createIdea(ctx, "someday thought", now.Add(-3*time.Hour))
		s.setIdeaStatus(ctx, someday, domain.IdeaStatusSomeday)
		spike := s.createIdea(ctx, "spike thought", now.Add(-2*time.Hour))
		s.setIdeaStatus(ctx, spike, domain.IdeaStatusSpike)
		blocked := s.createIdea(ctx, "blocked thought", now.Add(-time.Hour))
		s.setIdeaStatus(ctx, blocked, domain.IdeaStatusSpike)
		s.resolveIdea(ctx, blocked, domain.IdeaResolutionBlocked, now)
		dropped := s.createIdea(ctx, "dropped thought", now)
		s.resolveIdea(ctx, dropped, domain.IdeaResolutionDropped, now)

		duplicate := s.createIdea(ctx, "inbox thought again", now)
		_, _, err := ideas.ResolveIdea(ctx, nil, ideas.ResolveIdeaInput{IdeaID: duplicate, Resolution: "merged", MergedIntoID: &someday})
		require.NoError(s.T(), err)

		req := httptest.NewRequest(http.MethodGet, "/web/ideas", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.Contains(s.T(), html, `action="/web/ideas"`)
		assert.Contains(s.T(), html, `href="/web/ideas"`)

		order := []string{">spike<", "spike thought", ">inbox<", "inbox thought", ">someday<", "someday thought", ">blocked<", "blocked thought"}
		last := -1
		for _, part := range order {
			idx := strings.Index(html, part)
			require.Greater(s.T(), idx, last, part)
			last = idx
		}
		assert.NotContains(s.T(), html, "dropped thought")
		assert.NotContains(s.T(), html, "inbox thought again")
		assert.Contains(s.T(), html, "×2")
	})

	s.Run("POST creates an idea and redirects", func() {
		w := s.postIdea(r, "captured from web")

		assert.Equal(s.T(), http.StatusSeeOther, w.Code)
		assert.Equal(s.T(), "/web/ideas", w.Header().Get("Location"))

		found, err := s.Repo().SearchIdeas(ctx, domain.IdeaSearchFilter{UserID: userID, Query: "captured from web"})
		require.NoError(s.T(), err)
		require.Len(s.T(), found, 1)
		assert.Equal(s.T(), domain.IdeaStatusInbox, found[0].Status)
	})

	s.Run("POST with empty body re-renders with an error", func() {
		before, err := s.Repo().ListIdeas(ctx, domain.IdeaFilter{UserID: userID})
		require.NoError(s.T(), err)

		w := s.postIdea(r, "  ")

		assert.Equal(s.T(), http.StatusBadRequest, w.Code)
		assert.Contains(s.T(), w.Body.String(), "body is required")
		after, err := s.Repo().ListIdeas(ctx, domain.IdeaFilter{UserID: userID})
		require.NoError(s.T(), err)
		assert.Len(s.T(), after, len(before))
	})
}
