package tests

// Covers GET /web/ideas, POST /web/ideas, GET /web/ideas/spike and
// GET /web/ideas/search from
// docs/functions/ideas-spec.md.

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
	r.GET("/web/ideas/spike", ideas.SpikeIdeasWebHandler)
	r.GET("/web/ideas/search", ideas.SearchIdeasWebHandler)
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

func (s *IntegrationTestSuite) getIdeasPage(r *gin.Engine, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func (s *IntegrationTestSuite) TestIdeasWeb() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	now := time.Now()
	r := s.ideasRouter(ctx)

	s.Run("GET shows inbox ideas as cards, newest first", func() {
		s.createIdea(ctx, "older inbox thought", now.Add(-4*time.Hour))
		newer := s.createIdea(ctx, "newer inbox thought", now.Add(-3*time.Hour))
		someday := s.createIdea(ctx, "someday thought", now.Add(-2*time.Hour))
		s.setIdeaStatus(ctx, someday, domain.IdeaStatusSomeday)
		spike := s.createIdea(ctx, "spike thought", now.Add(-time.Hour))
		s.setIdeaStatus(ctx, spike, domain.IdeaStatusSpike)
		dropped := s.createIdea(ctx, "dropped thought", now)
		s.resolveIdea(ctx, dropped, domain.IdeaResolutionDropped, now)

		duplicate := s.createIdea(ctx, "newer inbox thought again", now)
		_, _, err := ideas.ResolveIdea(ctx, nil, ideas.ResolveIdeaInput{IdeaID: duplicate, Resolution: "merged", MergedIntoID: &newer})
		require.NoError(s.T(), err)

		w := s.getIdeasPage(r, "/web/ideas")

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.Contains(s.T(), html, `action="/web/ideas"`)
		assert.Contains(s.T(), html, `href="/web/ideas/spike"`)
		assert.Contains(s.T(), html, `href="/web/ideas/search"`)
		assert.Contains(s.T(), html, `<div class="grid">`)

		newerIdx := strings.Index(html, "<p>newer inbox thought</p>")
		olderIdx := strings.Index(html, "<p>older inbox thought</p>")
		require.Greater(s.T(), newerIdx, -1)
		require.Greater(s.T(), olderIdx, newerIdx)
		assert.Contains(s.T(), html, "<span>×2</span><span>"+now.Add(-3*time.Hour).Format("2006-01-02")+"</span></footer>")

		assert.NotContains(s.T(), html, "someday thought")
		assert.NotContains(s.T(), html, "spike thought")
		assert.NotContains(s.T(), html, "dropped thought")
		assert.NotContains(s.T(), html, "inbox thought again")
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

	s.Run("card keeps line breaks of a body captured via the form", func() {
		w := s.postIdea(r, "multiline first\r\nmultiline second\r\n\r\nmultiline paragraph")
		require.Equal(s.T(), http.StatusSeeOther, w.Code)

		html := s.getIdeasPage(r, "/web/ideas").Body.String()

		assert.Contains(s.T(), html, "<p>multiline first<br>multiline second</p>\n<p>multiline paragraph</p>")
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

func (s *IntegrationTestSuite) TestIdeasSpikeWeb() {
	ctx := s.Context()
	now := time.Now()
	r := s.ideasRouter(ctx)

	older := s.createIdea(ctx, "older spike thought", now.Add(-2*time.Hour))
	s.setIdeaStatus(ctx, older, domain.IdeaStatusSpike)
	newer := s.createIdea(ctx, "newer spike thought", now.Add(-time.Hour))
	s.setIdeaStatus(ctx, newer, domain.IdeaStatusSpike)
	s.createIdea(ctx, "inbox thought", now)
	someday := s.createIdea(ctx, "someday thought", now)
	s.setIdeaStatus(ctx, someday, domain.IdeaStatusSomeday)
	blocked := s.createIdea(ctx, "blocked thought", now)
	s.setIdeaStatus(ctx, blocked, domain.IdeaStatusSpike)
	s.resolveIdea(ctx, blocked, domain.IdeaResolutionBlocked, now)
	otherCtx := s.otherUserContext(ctx)
	foreign := s.createIdea(otherCtx, "other user's spike", now)
	s.setIdeaStatus(otherCtx, foreign, domain.IdeaStatusSpike)

	w := s.getIdeasPage(r, "/web/ideas/spike")

	require.Equal(s.T(), http.StatusOK, w.Code)
	html := w.Body.String()
	assert.Contains(s.T(), html, `href="/web/ideas"`)
	assert.Equal(s.T(), 2, strings.Count(html, "<article>"))
	assert.Contains(s.T(), html, "<span></span><span>"+now.Add(-time.Hour).Format("2006-01-02")+"</span></footer>")
	newerIdx := strings.Index(html, "<p>newer spike thought</p>")
	require.Greater(s.T(), newerIdx, -1)
	assert.Greater(s.T(), strings.Index(html, "<p>older spike thought</p>"), newerIdx)
	assert.NotContains(s.T(), html, "inbox thought")
	assert.NotContains(s.T(), html, "someday thought")
	assert.NotContains(s.T(), html, "blocked thought")
	assert.NotContains(s.T(), html, "other user")
}

func (s *IntegrationTestSuite) TestIdeasSearchWeb() {
	ctx := s.Context()
	now := time.Now()
	r := s.ideasRouter(ctx)

	s.Run("empty query shows the 50 newest ideas in any status", func() {
		for i := 0; i <= 50; i++ {
			s.createIdea(ctx, fmt.Sprintf("latest idea %02d", i), now.Add(time.Duration(i+1)*time.Hour))
		}
		blocked := s.createIdea(ctx, "blocked latest idea", now.Add(100*time.Hour))
		s.setIdeaStatus(ctx, blocked, domain.IdeaStatusSpike)
		s.resolveIdea(ctx, blocked, domain.IdeaResolutionBlocked, now)

		w := s.getIdeasPage(r, "/web/ideas/search")

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.Contains(s.T(), html, `<fieldset role="group">`)
		assert.Contains(s.T(), html, `href="/web/ideas"`)
		assert.Equal(s.T(), 50, strings.Count(html, "<article>"))
		assert.Contains(s.T(), html, "<p>blocked latest idea</p>\n<footer style=\"display: flex; justify-content: space-between\"><span>resolved · blocked</span>")
		assert.Contains(s.T(), html, "<p>latest idea 50</p>\n<footer style=\"display: flex; justify-content: space-between\"><span>inbox</span>")
		assert.Contains(s.T(), html, "latest idea 02")
		assert.NotContains(s.T(), html, "latest idea 01")
		assert.NotContains(s.T(), html, "latest idea 00")
		assert.Less(s.T(), strings.Index(html, "latest idea 50"), strings.Index(html, "latest idea 49"))
	})

	s.Run("comma-separated phrases match any status, case-insensitive", func() {
		s.createIdea(ctx, "Learn to SAIL a boat", now)
		dropped := s.createIdea(ctx, "yacht trip", now)
		s.resolveIdea(ctx, dropped, domain.IdeaResolutionDropped, now)
		both := s.createIdea(ctx, "sail the yacht around", now.Add(-time.Hour))
		duplicate := s.createIdea(ctx, "cruise somewhere", now)
		_, _, err := ideas.ResolveIdea(ctx, nil, ideas.ResolveIdeaInput{IdeaID: duplicate, Resolution: "merged", MergedIntoID: &both})
		require.NoError(s.T(), err)
		s.createIdea(ctx, "unrelated thought", now)
		s.createIdea(s.otherUserContext(ctx), "other user's sail", now)

		w := s.getIdeasPage(r, "/web/ideas/search?"+url.Values{"q": {" sail , , yacht "}}.Encode())

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.Contains(s.T(), html, `value=" sail , , yacht "`)
		assert.Equal(s.T(), 3, strings.Count(html, "<article>"))
		assert.Contains(s.T(), html, "<p>yacht trip</p>\n<footer style=\"display: flex; justify-content: space-between\"><span>resolved · dropped</span>")
		assert.Contains(s.T(), html, "<p>sail the yacht around</p>\n<footer style=\"display: flex; justify-content: space-between\"><span>inbox · ×2</span>")
		assert.NotContains(s.T(), html, "unrelated thought")
		assert.NotContains(s.T(), html, "other user&#39;s sail")
		// matched by both phrases, so ranked first despite being oldest
		assert.Less(s.T(), strings.Index(html, "sail the yacht around"), strings.Index(html, "Learn to SAIL a boat"))
	})

	s.Run("more than 5 phrases shows an error", func() {
		w := s.getIdeasPage(r, "/web/ideas/search?"+url.Values{"q": {"a,b,c,d,e,f"}}.Encode())

		assert.Equal(s.T(), http.StatusBadRequest, w.Code)
		assert.Contains(s.T(), w.Body.String(), "maximum 5 comma-separated phrases allowed")
		assert.NotContains(s.T(), w.Body.String(), "<article>")
	})
}
