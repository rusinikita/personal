package tests

// Covers the /web/personal-docs routes from
// docs/functions/personaldocs-spec.md.

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/personaldocs"
	"personal/gateways"
	"personal/util"
)

func (s *IntegrationTestSuite) personalDocsRouter(ctx context.Context) *gin.Engine {
	userID := gateways.UserIDFromContext(ctx)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		reqCtx := gateways.WithDB(c.Request.Context(), s.Repo())
		reqCtx = gateways.WithUserID(reqCtx, userID)
		c.Request = c.Request.WithContext(reqCtx)
		c.Next()
	})
	r.GET("/web/personal-docs", personaldocs.PersonalDocsWebHandler)
	r.GET("/web/personal-docs/new", personaldocs.NewPersonalDocWebHandler)
	r.POST("/web/personal-docs", personaldocs.CreatePersonalDocWebHandler)
	r.GET("/web/personal-docs/:id", personaldocs.PersonalDocWebHandler)
	r.POST("/web/personal-docs/:id", personaldocs.EditPersonalDocWebHandler)
	r.POST("/web/personal-docs/:id/chunks", personaldocs.CreatePersonalDocChunkWebHandler)
	r.POST("/web/personal-docs/:id/chunks/:chunk_id", personaldocs.SavePersonalDocChunkWebHandler)
	r.POST("/web/personal-docs/:id/chunks/:chunk_id/move", personaldocs.MovePersonalDocChunkWebHandler)
	r.POST("/web/personal-docs/:id/chunks/:chunk_id/revert", personaldocs.RevertPersonalDocChunkWebHandler)
	r.POST("/web/personal-docs/:id/chunks/:chunk_id/delete", personaldocs.DeletePersonalDocChunkWebHandler)
	return r
}

func personalDocsGet(r *gin.Engine, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func personalDocsPost(r *gin.Engine, target string, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// editPersonalDocChunkBody gives a seeded chunk a previous body.
func (s *IntegrationTestSuite) editPersonalDocChunkBody(ctx context.Context, chunkID int64, body string) {
	chunk := s.getPersonalDocChunk(ctx, chunkID)
	chunk.PreviousBody = util.Ptr(chunk.Body)
	chunk.Body = body
	require.NoError(s.T(), s.Repo().UpdatePersonalDocChunk(ctx, chunk))
}

func (s *IntegrationTestSuite) TestPersonalDocsWeb() {
	ctx := s.Context()
	r := s.personalDocsRouter(ctx)

	s.Run("no documents", func() {
		w := personalDocsGet(r, "/web/personal-docs")

		require.Equal(s.T(), http.StatusOK, w.Code)
		assert.Contains(s.T(), w.Body.String(), `href="/web/personal-docs/new"`)
		assert.Contains(s.T(), w.Body.String(), "No documents yet.")
	})

	s.Run("lists the user's documents", func() {
		first := s.createPersonalDoc(ctx, "Development plan")
		second := s.createPersonalDoc(ctx, "Personal <instructions>")
		foreign := s.createPersonalDoc(s.otherUserContext(ctx), "Foreign doc")

		w := personalDocsGet(r, "/web/personal-docs")

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.Contains(s.T(), html, fmt.Sprintf(`<a href="/web/personal-docs/%d">Development plan</a>`, first))
		assert.Contains(s.T(), html, "Development plan description")
		assert.Contains(s.T(), html, fmt.Sprintf(`<a href="/web/personal-docs/%d">Personal &lt;instructions&gt;</a>`, second))
		assert.Contains(s.T(), html, `href="/web/personal-docs/new"`)
		assert.Contains(s.T(), html, "New document")
		assert.Contains(s.T(), html, `<a href="/web/personal-docs" aria-current="page">Personal docs</a>`)
		assert.NotContains(s.T(), html, "Foreign doc")
		assert.NotContains(s.T(), html, fmt.Sprintf(`/web/personal-docs/%d"`, foreign))
	})
}

func (s *IntegrationTestSuite) TestPersonalDocsCreateWeb() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	r := s.personalDocsRouter(ctx)

	s.Run("GET shows the form", func() {
		w := personalDocsGet(r, "/web/personal-docs/new")

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.Contains(s.T(), html, `<form method="POST" action="/web/personal-docs">`)
		assert.Contains(s.T(), html, `name="title"`)
		assert.Contains(s.T(), html, `name="description"`)
	})

	s.Run("POST creates the document and redirects to it", func() {
		w := personalDocsPost(r, "/web/personal-docs", url.Values{"title": {"Plan"}, "description": {"Where I am going"}})

		require.Equal(s.T(), http.StatusSeeOther, w.Code)
		docs, err := s.Repo().ListPersonalDocs(ctx, userID)
		require.NoError(s.T(), err)
		require.Len(s.T(), docs, 1)
		assert.Equal(s.T(), "Plan", docs[0].Title)
		assert.Equal(s.T(), "Where I am going", docs[0].Description)
		assert.Equal(s.T(), fmt.Sprintf("/web/personal-docs/%d", docs[0].ID), w.Header().Get("Location"))
	})

	failures := []struct {
		name        string
		title       string
		description string
		wantErr     string
	}{
		{name: "empty title", title: "", description: "kept description", wantErr: "title is required"},
		{name: "empty description", title: "kept title", description: " ", wantErr: "description is required"},
	}

	for _, tt := range failures {
		s.Run(tt.name+" re-renders with an error", func() {
			before, err := s.Repo().ListPersonalDocs(ctx, userID)
			require.NoError(s.T(), err)

			w := personalDocsPost(r, "/web/personal-docs", url.Values{"title": {tt.title}, "description": {tt.description}})

			assert.Equal(s.T(), http.StatusBadRequest, w.Code)
			html := w.Body.String()
			assert.Contains(s.T(), html, tt.wantErr)
			assert.Contains(s.T(), html, fmt.Sprintf(`name="title" value="%s"`, tt.title))
			assert.Contains(s.T(), html, fmt.Sprintf(`name="description" value="%s"`, tt.description))
			after, err := s.Repo().ListPersonalDocs(ctx, userID)
			require.NoError(s.T(), err)
			assert.Len(s.T(), after, len(before))
		})
	}
}

func (s *IntegrationTestSuite) TestPersonalDocViewWeb() {
	ctx := s.Context()
	r := s.personalDocsRouter(ctx)

	s.Run("chunks rendered in order with their buttons", func() {
		docID := s.createPersonalDoc(ctx, "Plan")
		ids := s.createPersonalDocChunks(ctx, docID, "# First heading", "second **bold**\n\n<script>alert(1)</script>", "third text")
		s.editPersonalDocChunkBody(ctx, ids[2], "third edited")

		w := personalDocsGet(r, fmt.Sprintf("/web/personal-docs/%d", docID))

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.Contains(s.T(), html, "<h2>Plan</h2>")
		assert.Contains(s.T(), html, "Plan description")
		assert.Contains(s.T(), html, "<h1>First heading</h1>")
		assert.Contains(s.T(), html, "<strong>bold</strong>")
		assert.NotContains(s.T(), html, "<script>alert(1)</script>")
		assert.Contains(s.T(), html, "third edited")
		assert.NotContains(s.T(), html, "<textarea")

		var anchors []int
		for _, id := range ids {
			idx := strings.Index(html, fmt.Sprintf(`<article id="chunk-%d">`, id))
			require.Greater(s.T(), idx, -1)
			anchors = append(anchors, idx)
			assert.Contains(s.T(), html, fmt.Sprintf(`href="/web/personal-docs/%d?edit=%d#chunk-%d"`, docID, id, id))
			assert.Contains(s.T(), html, fmt.Sprintf(`href="/web/personal-docs/%d?delete=%d#chunk-%d"`, docID, id, id))
			assert.Contains(s.T(), html, fmt.Sprintf(`action="/web/personal-docs/%d/chunks/%d/move"`, docID, id))
		}
		assert.Less(s.T(), anchors[0], anchors[1])
		assert.Less(s.T(), anchors[1], anchors[2])

		// only the edited chunk can be reverted
		assert.Contains(s.T(), html, fmt.Sprintf(`?revert=%d#chunk-%d"`, ids[2], ids[2]))
		assert.NotContains(s.T(), html, fmt.Sprintf(`?revert=%d#`, ids[0]))
		assert.NotContains(s.T(), html, fmt.Sprintf(`?revert=%d#`, ids[1]))

		// first chunk has no up, last chunk has no down
		first := html[anchors[0]:anchors[1]]
		middle := html[anchors[1]:anchors[2]]
		last := html[anchors[2]:]
		assert.NotContains(s.T(), first, `title="Move up"`)
		assert.Contains(s.T(), first, `title="Move down"`)
		assert.Contains(s.T(), middle, `title="Move up"`)
		assert.Contains(s.T(), middle, `title="Move down"`)
		assert.Contains(s.T(), last, `title="Move up"`)
		assert.NotContains(s.T(), last, `title="Move down"`)

		assert.Contains(s.T(), html, fmt.Sprintf(`href="/web/personal-docs/%d?new=1#chunk-new"`, docID))
		assert.Contains(s.T(), html, fmt.Sprintf(`href="/web/personal-docs/%d?edit_doc=1"`, docID))
	})

	s.Run("single chunk has no move buttons", func() {
		docID := s.createPersonalDoc(ctx, "Single")
		s.createPersonalDocChunks(ctx, docID, "only")

		html := personalDocsGet(r, fmt.Sprintf("/web/personal-docs/%d", docID)).Body.String()

		assert.NotContains(s.T(), html, `title="Move up"`)
		assert.NotContains(s.T(), html, `title="Move down"`)
		assert.Contains(s.T(), html, `title="Edit"`)
		assert.Contains(s.T(), html, `title="Delete"`)
	})

	s.Run("empty document shows only Add chunk", func() {
		docID := s.createPersonalDoc(ctx, "Empty")

		w := personalDocsGet(r, fmt.Sprintf("/web/personal-docs/%d", docID))

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.NotContains(s.T(), html, `id="chunk-`)
		assert.Contains(s.T(), html, fmt.Sprintf(`href="/web/personal-docs/%d?new=1#chunk-new"`, docID))
	})

	foreign := s.createPersonalDoc(s.otherUserContext(ctx), "Foreign")
	notFound := []struct {
		name   string
		target string
	}{
		{name: "unknown document", target: "/web/personal-docs/999999999"},
		{name: "not a number", target: "/web/personal-docs/abc"},
		{name: "other user's document", target: fmt.Sprintf("/web/personal-docs/%d", foreign)},
	}
	for _, tt := range notFound {
		s.Run(tt.name+" is 404", func() {
			assert.Equal(s.T(), http.StatusNotFound, personalDocsGet(r, tt.target).Code)
		})
	}
}

func (s *IntegrationTestSuite) TestPersonalDocEditDocWeb() {
	ctx := s.Context()
	otherCtx := s.otherUserContext(ctx)
	r := s.personalDocsRouter(ctx)

	s.Run("edit_doc mode shows a prefilled form", func() {
		docID := s.createPersonalDoc(ctx, "Plan")
		s.createPersonalDocChunks(ctx, docID, "chunk text")

		w := personalDocsGet(r, fmt.Sprintf("/web/personal-docs/%d?edit_doc=1", docID))

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.Contains(s.T(), html, fmt.Sprintf(`<form method="POST" action="/web/personal-docs/%d">`, docID))
		assert.Contains(s.T(), html, `name="title" value="Plan"`)
		assert.Contains(s.T(), html, `name="description" value="Plan description"`)
		assert.Contains(s.T(), html, "<p>chunk text</p>")
		assert.Contains(s.T(), html, `title="Edit"`)
	})

	s.Run("POST saves both and redirects", func() {
		docID := s.createPersonalDoc(ctx, "Plan")

		w := personalDocsPost(r, fmt.Sprintf("/web/personal-docs/%d", docID), url.Values{"title": {"New plan"}, "description": {"New description"}})

		require.Equal(s.T(), http.StatusSeeOther, w.Code)
		assert.Equal(s.T(), fmt.Sprintf("/web/personal-docs/%d", docID), w.Header().Get("Location"))
		doc := s.getPersonalDoc(ctx, docID)
		assert.Equal(s.T(), "New plan", doc.Title)
		assert.Equal(s.T(), "New description", doc.Description)
	})

	failures := []struct {
		name        string
		title       string
		description string
		wantErr     string
		wantValue   string
	}{
		{name: "empty title", title: "", description: "entered description", wantErr: "title is required", wantValue: `name="description" value="entered description"`},
		{name: "multi-line description", title: "entered title", description: "one\r\ntwo", wantErr: "description must be one line", wantValue: `name="title" value="entered title"`},
	}
	for _, tt := range failures {
		s.Run(tt.name+" re-renders in document edit mode", func() {
			docID := s.createPersonalDoc(ctx, "Plan")
			s.createPersonalDocChunks(ctx, docID, "chunk text")

			w := personalDocsPost(r, fmt.Sprintf("/web/personal-docs/%d", docID), url.Values{"title": {tt.title}, "description": {tt.description}})

			assert.Equal(s.T(), http.StatusBadRequest, w.Code)
			html := w.Body.String()
			assert.Contains(s.T(), html, tt.wantErr)
			assert.Contains(s.T(), html, tt.wantValue)
			assert.Contains(s.T(), html, "<p>chunk text</p>")
			doc := s.getPersonalDoc(ctx, docID)
			assert.Equal(s.T(), "Plan", doc.Title)
			assert.Equal(s.T(), "Plan description", doc.Description)
		})
	}

	s.Run("unknown document is 404", func() {
		w := personalDocsPost(r, "/web/personal-docs/999999999", url.Values{"title": {"x"}, "description": {"y"}})

		assert.Equal(s.T(), http.StatusNotFound, w.Code)
	})

	s.Run("other user's document is 404 and unchanged", func() {
		foreign := s.createPersonalDoc(otherCtx, "Foreign")

		w := personalDocsPost(r, fmt.Sprintf("/web/personal-docs/%d", foreign), url.Values{"title": {"x"}, "description": {"y"}})

		assert.Equal(s.T(), http.StatusNotFound, w.Code)
		assert.Equal(s.T(), "Foreign", s.getPersonalDoc(otherCtx, foreign).Title)
	})
}

func (s *IntegrationTestSuite) TestPersonalDocEditModeWeb() {
	ctx := s.Context()
	otherCtx := s.otherUserContext(ctx)
	r := s.personalDocsRouter(ctx)

	docID := s.createPersonalDoc(ctx, "Plan")
	ids := s.createPersonalDocChunks(ctx, docID, "first text", "# Raw <b>heading</b>\n\nsecond", "third text")

	s.Run("shows the chunk as a form", func() {
		w := personalDocsGet(r, fmt.Sprintf("/web/personal-docs/%d?edit=%d", docID, ids[1]))

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.Equal(s.T(), 1, strings.Count(html, "<textarea"))
		assert.Contains(s.T(), html, fmt.Sprintf(`<form method="POST" action="/web/personal-docs/%d/chunks/%d">`, docID, ids[1]))
		assert.Contains(s.T(), html, "# Raw &lt;b&gt;heading&lt;/b&gt;\n\nsecond</textarea>")
		assert.NotContains(s.T(), html, "<h1>Raw")
		assert.Contains(s.T(), html, `<button type="submit">Save</button>`)
		assert.Contains(s.T(), html, fmt.Sprintf(`href="/web/personal-docs/%d#chunk-%d">Cancel</a>`, docID, ids[1]))

		// the other chunks stay rendered with their buttons
		assert.Contains(s.T(), html, "<p>first text</p>")
		assert.Contains(s.T(), html, "<p>third text</p>")
		assert.Contains(s.T(), html, fmt.Sprintf(`?edit=%d#chunk-%d"`, ids[0], ids[0]))
		assert.Contains(s.T(), html, fmt.Sprintf(`?edit=%d#chunk-%d"`, ids[2], ids[2]))
		assert.NotContains(s.T(), html, fmt.Sprintf(`?edit=%d#chunk-%d"`, ids[1], ids[1]))
	})

	otherDoc := s.createPersonalDoc(ctx, "Other")
	otherDocChunk := s.createPersonalDocChunks(ctx, otherDoc, "other doc chunk")[0]
	foreignChunk := s.createPersonalDocChunks(otherCtx, s.createPersonalDoc(otherCtx, "Foreign"), "foreign")[0]

	notFound := []struct {
		name  string
		chunk string
	}{
		{name: "unknown chunk", chunk: "999999999"},
		{name: "not a number", chunk: "abc"},
		{name: "chunk of another document", chunk: fmt.Sprint(otherDocChunk)},
		{name: "other user's chunk", chunk: fmt.Sprint(foreignChunk)},
	}
	for _, tt := range notFound {
		s.Run(tt.name+" is 404", func() {
			w := personalDocsGet(r, fmt.Sprintf("/web/personal-docs/%d?edit=%s", docID, tt.chunk))

			assert.Equal(s.T(), http.StatusNotFound, w.Code)
		})
	}
}

func (s *IntegrationTestSuite) TestPersonalDocConfirmModesWeb() {
	ctx := s.Context()
	otherCtx := s.otherUserContext(ctx)
	r := s.personalDocsRouter(ctx)

	docID := s.createPersonalDoc(ctx, "Plan")
	ids := s.createPersonalDocChunks(ctx, docID, "never edited", "old <text>")
	s.editPersonalDocChunkBody(ctx, ids[1], "new text")

	s.Run("delete confirmation changes nothing", func() {
		w := personalDocsGet(r, fmt.Sprintf("/web/personal-docs/%d?delete=%d", docID, ids[0]))

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.Contains(s.T(), html, "This cannot be undone.")
		assert.Contains(s.T(), html, fmt.Sprintf(`<form method="POST" action="/web/personal-docs/%d/chunks/%d/delete">`, docID, ids[0]))
		assert.Contains(s.T(), html, "<p>never edited</p>")
		assert.NotContains(s.T(), html, fmt.Sprintf(`/chunks/%d/delete"`, ids[1]))
		s.assertPersonalDocChunks(ctx, docID, "never edited", "new text")
	})

	s.Run("revert confirmation changes nothing", func() {
		w := personalDocsGet(r, fmt.Sprintf("/web/personal-docs/%d?revert=%d", docID, ids[1]))

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.Contains(s.T(), html, "<pre>\nold &lt;text&gt;</pre>")
		assert.Contains(s.T(), html, fmt.Sprintf(`<form method="POST" action="/web/personal-docs/%d/chunks/%d/revert">`, docID, ids[1]))
		assert.Contains(s.T(), html, "<p>new text</p>")
		chunk := s.getPersonalDocChunk(ctx, ids[1])
		assert.Equal(s.T(), "new text", chunk.Body)
		assert.Equal(s.T(), util.Ptr("old <text>"), chunk.PreviousBody)
	})

	s.Run("revert of a never edited chunk is 400", func() {
		w := personalDocsGet(r, fmt.Sprintf("/web/personal-docs/%d?revert=%d", docID, ids[0]))

		assert.Equal(s.T(), http.StatusBadRequest, w.Code)
	})

	otherDoc := s.createPersonalDoc(ctx, "Other")
	otherDocChunk := s.createPersonalDocChunks(ctx, otherDoc, "other doc chunk")[0]
	s.editPersonalDocChunkBody(ctx, otherDocChunk, "other doc chunk edited")
	foreignChunk := s.createPersonalDocChunks(otherCtx, s.createPersonalDoc(otherCtx, "Foreign"), "foreign")[0]
	s.editPersonalDocChunkBody(otherCtx, foreignChunk, "foreign edited")

	for _, mode := range []string{"delete", "revert"} {
		for name, chunk := range map[string]string{
			"unknown chunk":             "999999999",
			"chunk of another document": fmt.Sprint(otherDocChunk),
			"other user's chunk":        fmt.Sprint(foreignChunk),
		} {
			s.Run(mode+" with "+name+" is 404", func() {
				w := personalDocsGet(r, fmt.Sprintf("/web/personal-docs/%d?%s=%s", docID, mode, chunk))

				assert.Equal(s.T(), http.StatusNotFound, w.Code)
			})
		}
	}
}

func (s *IntegrationTestSuite) TestPersonalDocNewChunkModeWeb() {
	ctx := s.Context()
	r := s.personalDocsRouter(ctx)

	docID := s.createPersonalDoc(ctx, "Plan")
	ids := s.createPersonalDocChunks(ctx, docID, "first text", "second text")

	w := personalDocsGet(r, fmt.Sprintf("/web/personal-docs/%d?new=1", docID))

	require.Equal(s.T(), http.StatusOK, w.Code)
	html := w.Body.String()
	formIdx := strings.Index(html, `<article id="chunk-new">`)
	require.Greater(s.T(), formIdx, -1)
	assert.Greater(s.T(), formIdx, strings.Index(html, fmt.Sprintf(`<article id="chunk-%d">`, ids[1])))
	assert.Contains(s.T(), html, fmt.Sprintf(`<form method="POST" action="/web/personal-docs/%d/chunks">`, docID))
	assert.Contains(s.T(), html, "required>\n</textarea>")
	assert.Contains(s.T(), html, "<p>first text</p>")
	assert.Contains(s.T(), html, "<p>second text</p>")
	s.assertPersonalDocChunks(ctx, docID, "first text", "second text")
}

func (s *IntegrationTestSuite) TestPersonalDocChunkCreateWeb() {
	ctx := s.Context()
	r := s.personalDocsRouter(ctx)

	s.Run("adds the chunk at the end and redirects to its anchor", func() {
		docID := s.createPersonalDoc(ctx, "Plan")
		s.createPersonalDocChunks(ctx, docID, "first")

		w := personalDocsPost(r, fmt.Sprintf("/web/personal-docs/%d/chunks", docID), url.Values{"body": {"line one\r\nline two\r\n\r\nparagraph"}})

		require.Equal(s.T(), http.StatusSeeOther, w.Code)
		s.assertPersonalDocChunks(ctx, docID, "first", "line one\nline two\n\nparagraph")
		chunks := s.listPersonalDocChunks(ctx, docID)
		assert.Equal(s.T(), fmt.Sprintf("/web/personal-docs/%d#chunk-%d", docID, chunks[1].ID), w.Header().Get("Location"))
	})

	failures := []struct {
		name     string
		body     string
		wantErr  string
		wantText string
	}{
		{name: "too many lines", body: personalDocLines(personalDocMaxLines+1, "entered line"), wantErr: "body has 61 lines", wantText: "entered line\nentered line"},
		{name: "too long line", body: "entered " + strings.Repeat("a", personalDocMaxLineLength), wantErr: "has 208 characters", wantText: "entered aaaa"},
		{name: "empty body", body: "  ", wantErr: "body is required", wantText: "</textarea>"},
	}
	for _, tt := range failures {
		s.Run(tt.name+" re-renders in new chunk mode", func() {
			docID := s.createPersonalDoc(ctx, "Plan")
			s.createPersonalDocChunks(ctx, docID, "first")

			w := personalDocsPost(r, fmt.Sprintf("/web/personal-docs/%d/chunks", docID), url.Values{"body": {tt.body}})

			assert.Equal(s.T(), http.StatusBadRequest, w.Code)
			html := w.Body.String()
			assert.Contains(s.T(), html, tt.wantErr)
			assert.Contains(s.T(), html, `<article id="chunk-new">`)
			assert.Contains(s.T(), html, tt.wantText)
			s.assertPersonalDocChunks(ctx, docID, "first")
		})
	}

	s.Run("unknown document is 404", func() {
		w := personalDocsPost(r, "/web/personal-docs/999999999/chunks", url.Values{"body": {"text"}})

		assert.Equal(s.T(), http.StatusNotFound, w.Code)
	})
}

// personalDocChunkNotFoundCases seeds the chunks a POST to
// /web/personal-docs/{docID}/chunks/{chunk} must answer 404 for, and returns
// their request paths by case name.
func (s *IntegrationTestSuite) personalDocChunkNotFoundCases(ctx context.Context, docID int64, ownChunk int64) (map[string]string, func()) {
	otherCtx := s.otherUserContext(ctx)
	otherDoc := s.createPersonalDoc(ctx, "Other")
	otherDocChunk := s.createPersonalDocChunks(ctx, otherDoc, "other one", "other two")[1]
	s.editPersonalDocChunkBody(ctx, otherDocChunk, "other two edited")
	foreignDoc := s.createPersonalDoc(otherCtx, "Foreign")
	foreignChunk := s.createPersonalDocChunks(otherCtx, foreignDoc, "foreign one", "foreign two")[1]
	s.editPersonalDocChunkBody(otherCtx, foreignChunk, "foreign two edited")

	cases := map[string]string{
		"unknown document":          fmt.Sprintf("/web/personal-docs/999999999/chunks/%d", ownChunk),
		"unknown chunk":             fmt.Sprintf("/web/personal-docs/%d/chunks/999999999", docID),
		"chunk of another document": fmt.Sprintf("/web/personal-docs/%d/chunks/%d", docID, otherDocChunk),
		"other user's chunk":        fmt.Sprintf("/web/personal-docs/%d/chunks/%d", docID, foreignChunk),
		"other user's document":     fmt.Sprintf("/web/personal-docs/%d/chunks/%d", foreignDoc, foreignChunk),
	}
	assertUnchanged := func() {
		s.assertPersonalDocChunks(ctx, otherDoc, "other one", "other two edited")
		s.assertPersonalDocChunks(otherCtx, foreignDoc, "foreign one", "foreign two edited")
		assert.Equal(s.T(), util.Ptr("other two"), s.getPersonalDocChunk(ctx, otherDocChunk).PreviousBody)
		assert.Equal(s.T(), util.Ptr("foreign two"), s.getPersonalDocChunk(otherCtx, foreignChunk).PreviousBody)
	}
	return cases, assertUnchanged
}

func (s *IntegrationTestSuite) TestPersonalDocChunkSaveWeb() {
	ctx := s.Context()
	r := s.personalDocsRouter(ctx)

	docID := s.createPersonalDoc(ctx, "Plan")
	ids := s.createPersonalDocChunks(ctx, docID, "first", "second")

	s.Run("overwrites the body and redirects to the chunk's anchor", func() {
		w := personalDocsPost(r, fmt.Sprintf("/web/personal-docs/%d/chunks/%d", docID, ids[0]), url.Values{"body": {"first\r\nedited"}})

		require.Equal(s.T(), http.StatusSeeOther, w.Code)
		assert.Equal(s.T(), fmt.Sprintf("/web/personal-docs/%d#chunk-%d", docID, ids[0]), w.Header().Get("Location"))
		chunk := s.getPersonalDocChunk(ctx, ids[0])
		assert.Equal(s.T(), "first\nedited", chunk.Body)
		assert.Equal(s.T(), util.Ptr("first"), chunk.PreviousBody)
	})

	s.Run("body over a limit re-renders in edit mode", func() {
		body := "entered text\n" + strings.Repeat("a", personalDocMaxLineLength+1)

		w := personalDocsPost(r, fmt.Sprintf("/web/personal-docs/%d/chunks/%d", docID, ids[1]), url.Values{"body": {body}})

		assert.Equal(s.T(), http.StatusBadRequest, w.Code)
		html := w.Body.String()
		assert.Contains(s.T(), html, "line 2 of body has 201 characters")
		assert.Contains(s.T(), html, fmt.Sprintf(`<form method="POST" action="/web/personal-docs/%d/chunks/%d">`, docID, ids[1]))
		assert.Contains(s.T(), html, "entered text\naaa")
		chunk := s.getPersonalDocChunk(ctx, ids[1])
		assert.Equal(s.T(), "second", chunk.Body)
		assert.Nil(s.T(), chunk.PreviousBody)
	})

	cases, assertUnchanged := s.personalDocChunkNotFoundCases(ctx, docID, ids[1])
	for name, target := range cases {
		s.Run(name+" is 404", func() {
			w := personalDocsPost(r, target, url.Values{"body": {"hacked"}})

			assert.Equal(s.T(), http.StatusNotFound, w.Code)
			assertUnchanged()
			assert.Equal(s.T(), "second", s.getPersonalDocChunk(ctx, ids[1]).Body)
		})
	}
}

func (s *IntegrationTestSuite) TestPersonalDocChunkMoveWeb() {
	ctx := s.Context()
	r := s.personalDocsRouter(ctx)

	// Every case runs on a fresh document with the chunks a, b, c.
	tests := []struct {
		name      string
		index     int
		direction string
		want      []string
		wantCode  int
	}{
		{name: "up swaps with the previous chunk", index: 1, direction: "up", want: []string{"b", "a", "c"}, wantCode: http.StatusSeeOther},
		{name: "down swaps with the next chunk", index: 1, direction: "down", want: []string{"a", "c", "b"}, wantCode: http.StatusSeeOther},
		{name: "first chunk up is 400", index: 0, direction: "up", want: []string{"a", "b", "c"}, wantCode: http.StatusBadRequest},
		{name: "last chunk down is 400", index: 2, direction: "down", want: []string{"a", "b", "c"}, wantCode: http.StatusBadRequest},
		{name: "unknown direction is 400", index: 1, direction: "sideways", want: []string{"a", "b", "c"}, wantCode: http.StatusBadRequest},
		{name: "no direction is 400", index: 1, direction: "", want: []string{"a", "b", "c"}, wantCode: http.StatusBadRequest},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			docID := s.createPersonalDoc(ctx, "Plan")
			ids := s.createPersonalDocChunks(ctx, docID, "a", "b", "c")

			w := personalDocsPost(r, fmt.Sprintf("/web/personal-docs/%d/chunks/%d/move", docID, ids[tt.index]), url.Values{"direction": {tt.direction}})

			assert.Equal(s.T(), tt.wantCode, w.Code)
			if tt.wantCode == http.StatusSeeOther {
				assert.Equal(s.T(), fmt.Sprintf("/web/personal-docs/%d#chunk-%d", docID, ids[tt.index]), w.Header().Get("Location"))
			}
			s.assertPersonalDocChunks(ctx, docID, tt.want...)
		})
	}

	docID := s.createPersonalDoc(ctx, "Plan")
	ids := s.createPersonalDocChunks(ctx, docID, "a", "b", "c")
	cases, assertUnchanged := s.personalDocChunkNotFoundCases(ctx, docID, ids[1])
	for name, target := range cases {
		s.Run(name+" is 404", func() {
			w := personalDocsPost(r, target+"/move", url.Values{"direction": {"up"}})

			assert.Equal(s.T(), http.StatusNotFound, w.Code)
			assertUnchanged()
			s.assertPersonalDocChunks(ctx, docID, "a", "b", "c")
		})
	}
}

func (s *IntegrationTestSuite) TestPersonalDocChunkRevertWeb() {
	ctx := s.Context()
	r := s.personalDocsRouter(ctx)

	docID := s.createPersonalDoc(ctx, "Plan")
	ids := s.createPersonalDocChunks(ctx, docID, "never edited", "old")
	s.editPersonalDocChunkBody(ctx, ids[1], "new")

	s.Run("swaps the bodies and redirects", func() {
		w := personalDocsPost(r, fmt.Sprintf("/web/personal-docs/%d/chunks/%d/revert", docID, ids[1]), nil)

		require.Equal(s.T(), http.StatusSeeOther, w.Code)
		assert.Equal(s.T(), fmt.Sprintf("/web/personal-docs/%d#chunk-%d", docID, ids[1]), w.Header().Get("Location"))
		chunk := s.getPersonalDocChunk(ctx, ids[1])
		assert.Equal(s.T(), "old", chunk.Body)
		assert.Equal(s.T(), util.Ptr("new"), chunk.PreviousBody)
	})

	s.Run("chunk with no previous body is 400", func() {
		w := personalDocsPost(r, fmt.Sprintf("/web/personal-docs/%d/chunks/%d/revert", docID, ids[0]), nil)

		assert.Equal(s.T(), http.StatusBadRequest, w.Code)
		chunk := s.getPersonalDocChunk(ctx, ids[0])
		assert.Equal(s.T(), "never edited", chunk.Body)
		assert.Nil(s.T(), chunk.PreviousBody)
	})

	cases, assertUnchanged := s.personalDocChunkNotFoundCases(ctx, docID, ids[1])
	for name, target := range cases {
		s.Run(name+" is 404", func() {
			w := personalDocsPost(r, target+"/revert", nil)

			assert.Equal(s.T(), http.StatusNotFound, w.Code)
			assertUnchanged()
			assert.Equal(s.T(), "old", s.getPersonalDocChunk(ctx, ids[1]).Body)
		})
	}
}

func (s *IntegrationTestSuite) TestPersonalDocChunkDeleteWeb() {
	ctx := s.Context()
	r := s.personalDocsRouter(ctx)

	s.Run("deletes the chunk and redirects to the document", func() {
		docID := s.createPersonalDoc(ctx, "Plan")
		ids := s.createPersonalDocChunks(ctx, docID, "a", "b", "c")

		w := personalDocsPost(r, fmt.Sprintf("/web/personal-docs/%d/chunks/%d/delete", docID, ids[1]), nil)

		require.Equal(s.T(), http.StatusSeeOther, w.Code)
		assert.Equal(s.T(), fmt.Sprintf("/web/personal-docs/%d", docID), w.Header().Get("Location"))
		s.assertPersonalDocChunks(ctx, docID, "a", "c")
	})

	docID := s.createPersonalDoc(ctx, "Plan")
	ids := s.createPersonalDocChunks(ctx, docID, "a", "b")
	cases, assertUnchanged := s.personalDocChunkNotFoundCases(ctx, docID, ids[1])
	for name, target := range cases {
		s.Run(name+" is 404", func() {
			w := personalDocsPost(r, target+"/delete", nil)

			assert.Equal(s.T(), http.StatusNotFound, w.Code)
			assertUnchanged()
			s.assertPersonalDocChunks(ctx, docID, "a", "b")
		})
	}
}
