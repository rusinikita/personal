package tests

// Covers the /web/docs routes from
// docs/functions/docs-spec.md.

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

	"personal/action/docs"
	"personal/gateways"
	"personal/util"
)

func (s *IntegrationTestSuite) docsRouter(ctx context.Context) *gin.Engine {
	userID := gateways.UserIDFromContext(ctx)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		reqCtx := gateways.WithDB(c.Request.Context(), s.Repo())
		reqCtx = gateways.WithUserID(reqCtx, userID)
		c.Request = c.Request.WithContext(reqCtx)
		c.Next()
	})
	r.GET("/web/docs", docs.DocsWebHandler)
	r.GET("/web/docs/new", docs.NewDocWebHandler)
	r.POST("/web/docs", docs.CreateDocWebHandler)
	r.GET("/web/docs/:id", docs.DocWebHandler)
	r.POST("/web/docs/:id", docs.EditDocWebHandler)
	r.POST("/web/docs/:id/chunks", docs.CreateDocChunkWebHandler)
	r.POST("/web/docs/:id/chunks/:chunk_id", docs.SaveDocChunkWebHandler)
	r.POST("/web/docs/:id/chunks/:chunk_id/move", docs.MoveDocChunkWebHandler)
	r.POST("/web/docs/:id/chunks/:chunk_id/revert", docs.RevertDocChunkWebHandler)
	r.POST("/web/docs/:id/chunks/:chunk_id/delete", docs.DeleteDocChunkWebHandler)
	return r
}

func docsGet(r *gin.Engine, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func docsPost(r *gin.Engine, target string, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// editDocChunkBody gives a seeded chunk a previous body.
func (s *IntegrationTestSuite) editDocChunkBody(ctx context.Context, chunkID int64, body string) {
	chunk := s.getDocChunk(ctx, chunkID)
	chunk.PreviousBody = util.Ptr(chunk.Body)
	chunk.Body = body
	require.NoError(s.T(), s.Repo().UpdateDocChunk(ctx, chunk))
}

func (s *IntegrationTestSuite) TestDocsWeb() {
	ctx := s.Context()
	r := s.docsRouter(ctx)

	s.Run("no documents", func() {
		w := docsGet(r, "/web/docs")

		require.Equal(s.T(), http.StatusOK, w.Code)
		assert.Contains(s.T(), w.Body.String(), `href="/web/docs/new"`)
		assert.Contains(s.T(), w.Body.String(), "No documents yet.")
	})

	s.Run("lists the user's documents", func() {
		first := s.createDoc(ctx, "Development plan")
		second := s.createDoc(ctx, "Personal <instructions>")
		foreign := s.createDoc(s.otherUserContext(ctx), "Foreign doc")

		w := docsGet(r, "/web/docs")

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.Contains(s.T(), html, fmt.Sprintf(`<a href="/web/docs/%d">Development plan</a>`, first))
		assert.Contains(s.T(), html, "Development plan description")
		assert.Contains(s.T(), html, fmt.Sprintf(`<a href="/web/docs/%d">Personal &lt;instructions&gt;</a>`, second))
		assert.Contains(s.T(), html, `href="/web/docs/new"`)
		assert.Contains(s.T(), html, "New document")
		assert.Contains(s.T(), html, `<a href="/web/docs" aria-current="page">Docs</a>`)
		assert.NotContains(s.T(), html, "Foreign doc")
		assert.NotContains(s.T(), html, fmt.Sprintf(`/web/docs/%d"`, foreign))
	})
}

func (s *IntegrationTestSuite) TestDocsCreateWeb() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	r := s.docsRouter(ctx)

	s.Run("GET shows the form", func() {
		w := docsGet(r, "/web/docs/new")

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.Contains(s.T(), html, `<form method="POST" action="/web/docs">`)
		assert.Contains(s.T(), html, `name="title"`)
		assert.Contains(s.T(), html, `name="description"`)
	})

	s.Run("POST creates the document and redirects to it", func() {
		w := docsPost(r, "/web/docs", url.Values{"title": {"Plan"}, "description": {"Where I am going"}})

		require.Equal(s.T(), http.StatusSeeOther, w.Code)
		docs, err := s.Repo().ListDocs(ctx, userID)
		require.NoError(s.T(), err)
		require.Len(s.T(), docs, 1)
		assert.Equal(s.T(), "Plan", docs[0].Title)
		assert.Equal(s.T(), "Where I am going", docs[0].Description)
		assert.Equal(s.T(), fmt.Sprintf("/web/docs/%d", docs[0].ID), w.Header().Get("Location"))
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
			before, err := s.Repo().ListDocs(ctx, userID)
			require.NoError(s.T(), err)

			w := docsPost(r, "/web/docs", url.Values{"title": {tt.title}, "description": {tt.description}})

			assert.Equal(s.T(), http.StatusBadRequest, w.Code)
			html := w.Body.String()
			assert.Contains(s.T(), html, tt.wantErr)
			assert.Contains(s.T(), html, fmt.Sprintf(`name="title" value="%s"`, tt.title))
			assert.Contains(s.T(), html, fmt.Sprintf(`name="description" value="%s"`, tt.description))
			after, err := s.Repo().ListDocs(ctx, userID)
			require.NoError(s.T(), err)
			assert.Len(s.T(), after, len(before))
		})
	}
}

func (s *IntegrationTestSuite) TestDocViewWeb() {
	ctx := s.Context()
	r := s.docsRouter(ctx)

	s.Run("chunks rendered in order with their buttons", func() {
		docID := s.createDoc(ctx, "Plan")
		ids := s.createDocChunks(ctx, docID, "# First heading", "second **bold**\n\n<script>alert(1)</script>", "third text")
		s.editDocChunkBody(ctx, ids[2], "third edited")

		w := docsGet(r, fmt.Sprintf("/web/docs/%d", docID))

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.Contains(s.T(), html, "<h2>Plan</h2>")
		assert.Contains(s.T(), html, "Plan description")
		assert.Contains(s.T(), html, fmt.Sprintf(`<h1 id="chunk-%d-h1">First heading</h1>`, ids[0]))
		assert.Contains(s.T(), html, "<strong>bold</strong>")
		assert.NotContains(s.T(), html, "<script>alert(1)</script>")
		assert.Contains(s.T(), html, "third edited")
		assert.NotContains(s.T(), html, "<textarea")
		assert.NotContains(s.T(), html, "autofocus")

		var anchors []int
		for _, id := range ids {
			idx := strings.Index(html, fmt.Sprintf(`<article id="chunk-%d">`, id))
			require.Greater(s.T(), idx, -1)
			anchors = append(anchors, idx)
			assert.Contains(s.T(), html, fmt.Sprintf(`href="/web/docs/%d?edit=%d#chunk-%d"`, docID, id, id))
			assert.Contains(s.T(), html, fmt.Sprintf(`href="/web/docs/%d?delete=%d#chunk-%d"`, docID, id, id))
			assert.Contains(s.T(), html, fmt.Sprintf(`action="/web/docs/%d/chunks/%d/move"`, docID, id))
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

		assert.Contains(s.T(), html, fmt.Sprintf(`href="/web/docs/%d?new=1#chunk-new"`, docID))
		assert.Contains(s.T(), html, fmt.Sprintf(`href="/web/docs/%d?edit_doc=1"`, docID))
	})

	s.Run("mermaid code block is rendered as a diagram block", func() {
		docID := s.createDoc(ctx, "Diagram")
		s.createDocChunks(ctx, docID, "```mermaid\ngraph LR\n    A --> B\n```", "```go\nfmt.Println(1)\n```\n\n<script>alert(1)</script>")

		html := docsGet(r, fmt.Sprintf("/web/docs/%d", docID)).Body.String()

		assert.Contains(s.T(), html, "<pre class=\"mermaid\">graph LR\n    A --&gt; B\n</pre>")
		assert.Contains(s.T(), html, `<pre><code class="language-go">`)
		assert.NotContains(s.T(), html, "<script>alert(1)</script>")
		assert.Contains(s.T(), html, mermaidScriptSrc)
		assert.Contains(s.T(), html, mermaidInit)
	})

	s.Run("document with no diagram does not load mermaid JS", func() {
		docID := s.createDoc(ctx, "Plain")
		s.createDocChunks(ctx, docID, "```go\nfmt.Println(1)\n```")

		html := docsGet(r, fmt.Sprintf("/web/docs/%d", docID)).Body.String()

		assert.NotContains(s.T(), html, mermaidScriptSrc)
		assert.NotContains(s.T(), html, mermaidInit)
	})

	s.Run("single chunk has no move buttons", func() {
		docID := s.createDoc(ctx, "Single")
		s.createDocChunks(ctx, docID, "only")

		html := docsGet(r, fmt.Sprintf("/web/docs/%d", docID)).Body.String()

		assert.NotContains(s.T(), html, `title="Move up"`)
		assert.NotContains(s.T(), html, `title="Move down"`)
		assert.Contains(s.T(), html, `title="Edit"`)
		assert.Contains(s.T(), html, `title="Delete"`)
	})

	s.Run("empty document shows only Add chunk", func() {
		docID := s.createDoc(ctx, "Empty")

		w := docsGet(r, fmt.Sprintf("/web/docs/%d", docID))

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.NotContains(s.T(), html, `id="chunk-`)
		assert.Contains(s.T(), html, fmt.Sprintf(`href="/web/docs/%d?new=1#chunk-new"`, docID))
	})

	foreign := s.createDoc(s.otherUserContext(ctx), "Foreign")
	notFound := []struct {
		name   string
		target string
	}{
		{name: "unknown document", target: "/web/docs/999999999"},
		{name: "not a number", target: "/web/docs/abc"},
		{name: "other user's document", target: fmt.Sprintf("/web/docs/%d", foreign)},
	}
	for _, tt := range notFound {
		s.Run(tt.name+" is 404", func() {
			assert.Equal(s.T(), http.StatusNotFound, docsGet(r, tt.target).Code)
		})
	}
}

func (s *IntegrationTestSuite) TestDocContentsWeb() {
	ctx := s.Context()
	r := s.docsRouter(ctx)

	const contentsSummary = "<summary>Contents</summary>"

	docID := s.createDoc(ctx, "Plan")
	ids := s.createDocChunks(ctx, docID,
		"# Goals\n\ntext\n\n## This *year* `now` <b>!</b>\n\n```\n# not a heading\n```\n\n### Deep\n\n## Next year",
		"no headings here",
		"# Goals\n\nagain",
	)
	s.editDocChunkBody(ctx, ids[1], "still no headings")

	s.Run("lists the headings of all chunks nested by level", func() {
		w := docsGet(r, fmt.Sprintf("/web/docs/%d", docID))

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		start := strings.Index(html, "<details open>\n"+contentsSummary)
		require.Greater(s.T(), start, -1)
		end := start + strings.Index(html[start:], "</details>")
		contents := html[start:end]
		assert.Less(s.T(), end, strings.Index(html, fmt.Sprintf(`<article id="chunk-%d">`, ids[0])))

		// the second level is nested under "Goals", "Deep" under "This year now !"
		assert.Equal(s.T(), strings.Join([]string{
			"<details open>",
			contentsSummary,
			"<ul>",
			fmt.Sprintf(`<li><a href="#chunk-%d-h1">Goals</a><ul>`, ids[0]),
			fmt.Sprintf(`<li><a href="#chunk-%d-h2">This year now !</a><ul>`, ids[0]),
			fmt.Sprintf(`<li><a href="#chunk-%d-h3">Deep</a></li>`, ids[0]),
			"</ul></li>",
			fmt.Sprintf(`<li><a href="#chunk-%d-h4">Next year</a></li>`, ids[0]),
			"</ul></li>",
			fmt.Sprintf(`<li><a href="#chunk-%d-h1">Goals</a></li>`, ids[2]),
			"</ul>\n",
		}, "\n"), contents)

		// every link has its heading
		assert.Contains(s.T(), html, fmt.Sprintf(`<h1 id="chunk-%d-h1">Goals</h1>`, ids[0]))
		assert.Contains(s.T(), html, fmt.Sprintf(`<h2 id="chunk-%d-h2">This <em>year</em> <code>now</code> !</h2>`, ids[0]))
		assert.Contains(s.T(), html, fmt.Sprintf(`<h3 id="chunk-%d-h3">Deep</h3>`, ids[0]))
		assert.Contains(s.T(), html, fmt.Sprintf(`<h2 id="chunk-%d-h4">Next year</h2>`, ids[0]))
		assert.Contains(s.T(), html, fmt.Sprintf(`<h1 id="chunk-%d-h1">Goals</h1>`, ids[2]))
		assert.NotContains(s.T(), contents, "not a heading")
	})

	modes := []string{
		fmt.Sprintf("?edit=%d", ids[1]),
		"?new=1",
		fmt.Sprintf("?revert=%d", ids[1]),
		fmt.Sprintf("?delete=%d", ids[1]),
		"?edit_doc=1",
	}
	for _, mode := range modes {
		s.Run(mode+" shows no contents", func() {
			w := docsGet(r, fmt.Sprintf("/web/docs/%d%s", docID, mode))

			require.Equal(s.T(), http.StatusOK, w.Code)
			assert.NotContains(s.T(), w.Body.String(), contentsSummary)
		})
	}

	s.Run("document with no headings shows no contents", func() {
		plainDoc := s.createDoc(ctx, "Plain")
		s.createDocChunks(ctx, plainDoc, "just text", "```\n# in code\n```")

		assert.NotContains(s.T(), docsGet(r, fmt.Sprintf("/web/docs/%d", plainDoc)).Body.String(), contentsSummary)
	})

	s.Run("empty document shows no contents", func() {
		emptyDoc := s.createDoc(ctx, "Empty")

		assert.NotContains(s.T(), docsGet(r, fmt.Sprintf("/web/docs/%d", emptyDoc)).Body.String(), contentsSummary)
	})
}

func (s *IntegrationTestSuite) TestDocEditDocWeb() {
	ctx := s.Context()
	otherCtx := s.otherUserContext(ctx)
	r := s.docsRouter(ctx)

	s.Run("edit_doc mode shows a prefilled form", func() {
		docID := s.createDoc(ctx, "Plan")
		s.createDocChunks(ctx, docID, "chunk text")

		w := docsGet(r, fmt.Sprintf("/web/docs/%d?edit_doc=1", docID))

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.Contains(s.T(), html, fmt.Sprintf(`<form method="POST" action="/web/docs/%d">`, docID))
		assert.Contains(s.T(), html, `name="title" value="Plan"`)
		assert.Contains(s.T(), html, `name="description" value="Plan description"`)
		assert.Contains(s.T(), html, "<p>chunk text</p>")
		assert.Contains(s.T(), html, `title="Edit"`)
	})

	s.Run("POST saves both and redirects", func() {
		docID := s.createDoc(ctx, "Plan")

		w := docsPost(r, fmt.Sprintf("/web/docs/%d", docID), url.Values{"title": {"New plan"}, "description": {"New description"}})

		require.Equal(s.T(), http.StatusSeeOther, w.Code)
		assert.Equal(s.T(), fmt.Sprintf("/web/docs/%d", docID), w.Header().Get("Location"))
		doc := s.getDoc(ctx, docID)
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
			docID := s.createDoc(ctx, "Plan")
			s.createDocChunks(ctx, docID, "chunk text")

			w := docsPost(r, fmt.Sprintf("/web/docs/%d", docID), url.Values{"title": {tt.title}, "description": {tt.description}})

			assert.Equal(s.T(), http.StatusBadRequest, w.Code)
			html := w.Body.String()
			assert.Contains(s.T(), html, tt.wantErr)
			assert.Contains(s.T(), html, tt.wantValue)
			assert.Contains(s.T(), html, "<p>chunk text</p>")
			doc := s.getDoc(ctx, docID)
			assert.Equal(s.T(), "Plan", doc.Title)
			assert.Equal(s.T(), "Plan description", doc.Description)
		})
	}

	s.Run("unknown document is 404", func() {
		w := docsPost(r, "/web/docs/999999999", url.Values{"title": {"x"}, "description": {"y"}})

		assert.Equal(s.T(), http.StatusNotFound, w.Code)
	})

	s.Run("other user's document is 404 and unchanged", func() {
		foreign := s.createDoc(otherCtx, "Foreign")

		w := docsPost(r, fmt.Sprintf("/web/docs/%d", foreign), url.Values{"title": {"x"}, "description": {"y"}})

		assert.Equal(s.T(), http.StatusNotFound, w.Code)
		assert.Equal(s.T(), "Foreign", s.getDoc(otherCtx, foreign).Title)
	})
}

func (s *IntegrationTestSuite) TestDocEditModeWeb() {
	ctx := s.Context()
	otherCtx := s.otherUserContext(ctx)
	r := s.docsRouter(ctx)

	docID := s.createDoc(ctx, "Plan")
	ids := s.createDocChunks(ctx, docID, "first text", "# Raw <b>heading</b>\n\nsecond", "third text")

	s.Run("shows the chunk as a form", func() {
		w := docsGet(r, fmt.Sprintf("/web/docs/%d?edit=%d", docID, ids[1]))

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.Equal(s.T(), 1, strings.Count(html, "<textarea"))
		assert.Equal(s.T(), 1, strings.Count(html, "required autofocus>"))
		assert.Contains(s.T(), html, fmt.Sprintf(`<form method="POST" action="/web/docs/%d/chunks/%d">`, docID, ids[1]))
		assert.Contains(s.T(), html, "# Raw &lt;b&gt;heading&lt;/b&gt;\n\nsecond</textarea>")
		assert.NotContains(s.T(), html, "<h1>Raw")
		assert.Contains(s.T(), html, `<button type="submit">Save</button>`)
		assert.Contains(s.T(), html, fmt.Sprintf(`href="/web/docs/%d#chunk-%d">Cancel</a>`, docID, ids[1]))

		// the other chunks stay rendered with their buttons
		assert.Contains(s.T(), html, "<p>first text</p>")
		assert.Contains(s.T(), html, "<p>third text</p>")
		assert.Contains(s.T(), html, fmt.Sprintf(`?edit=%d#chunk-%d"`, ids[0], ids[0]))
		assert.Contains(s.T(), html, fmt.Sprintf(`?edit=%d#chunk-%d"`, ids[2], ids[2]))
		assert.NotContains(s.T(), html, fmt.Sprintf(`?edit=%d#chunk-%d"`, ids[1], ids[1]))
	})

	otherDoc := s.createDoc(ctx, "Other")
	otherDocChunk := s.createDocChunks(ctx, otherDoc, "other doc chunk")[0]
	foreignChunk := s.createDocChunks(otherCtx, s.createDoc(otherCtx, "Foreign"), "foreign")[0]

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
			w := docsGet(r, fmt.Sprintf("/web/docs/%d?edit=%s", docID, tt.chunk))

			assert.Equal(s.T(), http.StatusNotFound, w.Code)
		})
	}
}

func (s *IntegrationTestSuite) TestDocConfirmModesWeb() {
	ctx := s.Context()
	otherCtx := s.otherUserContext(ctx)
	r := s.docsRouter(ctx)

	docID := s.createDoc(ctx, "Plan")
	ids := s.createDocChunks(ctx, docID, "never edited", "old <text>")
	s.editDocChunkBody(ctx, ids[1], "new text")

	s.Run("delete confirmation changes nothing", func() {
		w := docsGet(r, fmt.Sprintf("/web/docs/%d?delete=%d", docID, ids[0]))

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.Contains(s.T(), html, "This cannot be undone.")
		assert.Contains(s.T(), html, fmt.Sprintf(`<form method="POST" action="/web/docs/%d/chunks/%d/delete">`, docID, ids[0]))
		assert.Contains(s.T(), html, "<p>never edited</p>")
		assert.NotContains(s.T(), html, fmt.Sprintf(`/chunks/%d/delete"`, ids[1]))
		s.assertDocChunks(ctx, docID, "never edited", "new text")
	})

	s.Run("revert confirmation changes nothing", func() {
		w := docsGet(r, fmt.Sprintf("/web/docs/%d?revert=%d", docID, ids[1]))

		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()
		assert.Contains(s.T(), html, "<pre>\nold &lt;text&gt;</pre>")
		assert.Contains(s.T(), html, fmt.Sprintf(`<form method="POST" action="/web/docs/%d/chunks/%d/revert">`, docID, ids[1]))
		assert.Contains(s.T(), html, "<p>new text</p>")
		chunk := s.getDocChunk(ctx, ids[1])
		assert.Equal(s.T(), "new text", chunk.Body)
		assert.Equal(s.T(), util.Ptr("old <text>"), chunk.PreviousBody)
	})

	s.Run("revert of a never edited chunk is 400", func() {
		w := docsGet(r, fmt.Sprintf("/web/docs/%d?revert=%d", docID, ids[0]))

		assert.Equal(s.T(), http.StatusBadRequest, w.Code)
	})

	otherDoc := s.createDoc(ctx, "Other")
	otherDocChunk := s.createDocChunks(ctx, otherDoc, "other doc chunk")[0]
	s.editDocChunkBody(ctx, otherDocChunk, "other doc chunk edited")
	foreignChunk := s.createDocChunks(otherCtx, s.createDoc(otherCtx, "Foreign"), "foreign")[0]
	s.editDocChunkBody(otherCtx, foreignChunk, "foreign edited")

	for _, mode := range []string{"delete", "revert"} {
		for name, chunk := range map[string]string{
			"unknown chunk":             "999999999",
			"chunk of another document": fmt.Sprint(otherDocChunk),
			"other user's chunk":        fmt.Sprint(foreignChunk),
		} {
			s.Run(mode+" with "+name+" is 404", func() {
				w := docsGet(r, fmt.Sprintf("/web/docs/%d?%s=%s", docID, mode, chunk))

				assert.Equal(s.T(), http.StatusNotFound, w.Code)
			})
		}
	}
}

func (s *IntegrationTestSuite) TestDocNewChunkModeWeb() {
	ctx := s.Context()
	r := s.docsRouter(ctx)

	docID := s.createDoc(ctx, "Plan")
	ids := s.createDocChunks(ctx, docID, "first text", "second text")

	w := docsGet(r, fmt.Sprintf("/web/docs/%d?new=1", docID))

	require.Equal(s.T(), http.StatusOK, w.Code)
	html := w.Body.String()
	formIdx := strings.Index(html, `<article id="chunk-new">`)
	require.Greater(s.T(), formIdx, -1)
	assert.Greater(s.T(), formIdx, strings.Index(html, fmt.Sprintf(`<article id="chunk-%d">`, ids[1])))
	assert.Contains(s.T(), html, fmt.Sprintf(`<form method="POST" action="/web/docs/%d/chunks">`, docID))
	assert.Contains(s.T(), html, "required autofocus>\n</textarea>")
	assert.Contains(s.T(), html, "<p>first text</p>")
	assert.Contains(s.T(), html, "<p>second text</p>")
	s.assertDocChunks(ctx, docID, "first text", "second text")
}

func (s *IntegrationTestSuite) TestDocChunkCreateWeb() {
	ctx := s.Context()
	r := s.docsRouter(ctx)

	s.Run("adds the chunk at the end and redirects to its anchor", func() {
		docID := s.createDoc(ctx, "Plan")
		s.createDocChunks(ctx, docID, "first")

		w := docsPost(r, fmt.Sprintf("/web/docs/%d/chunks", docID), url.Values{"body": {"line one\r\nline two\r\n\r\nparagraph"}})

		require.Equal(s.T(), http.StatusSeeOther, w.Code)
		s.assertDocChunks(ctx, docID, "first", "line one\nline two\n\nparagraph")
		chunks := s.listDocChunks(ctx, docID)
		assert.Equal(s.T(), fmt.Sprintf("/web/docs/%d#chunk-%d", docID, chunks[1].ID), w.Header().Get("Location"))
	})

	failures := []struct {
		name     string
		body     string
		wantErr  string
		wantText string
	}{
		{name: "too many lines", body: docLines(docMaxLines+1, "entered line"), wantErr: "body has 61 lines", wantText: "entered line\nentered line"},
		{name: "too long line", body: "entered " + strings.Repeat("a", docMaxLineLength), wantErr: "has 208 characters", wantText: "entered aaaa"},
		{name: "empty body", body: "  ", wantErr: "body is required", wantText: "</textarea>"},
	}
	for _, tt := range failures {
		s.Run(tt.name+" re-renders in new chunk mode", func() {
			docID := s.createDoc(ctx, "Plan")
			s.createDocChunks(ctx, docID, "first")

			w := docsPost(r, fmt.Sprintf("/web/docs/%d/chunks", docID), url.Values{"body": {tt.body}})

			assert.Equal(s.T(), http.StatusBadRequest, w.Code)
			html := w.Body.String()
			assert.Contains(s.T(), html, tt.wantErr)
			assert.Contains(s.T(), html, `<article id="chunk-new">`)
			assert.Contains(s.T(), html, tt.wantText)
			assert.Contains(s.T(), html, "required autofocus>")
			s.assertDocChunks(ctx, docID, "first")
		})
	}

	s.Run("unknown document is 404", func() {
		w := docsPost(r, "/web/docs/999999999/chunks", url.Values{"body": {"text"}})

		assert.Equal(s.T(), http.StatusNotFound, w.Code)
	})
}

// docChunkNotFoundCases seeds the chunks a POST to
// /web/docs/{docID}/chunks/{chunk} must answer 404 for, and returns
// their request paths by case name.
func (s *IntegrationTestSuite) docChunkNotFoundCases(ctx context.Context, docID int64, ownChunk int64) (map[string]string, func()) {
	otherCtx := s.otherUserContext(ctx)
	otherDoc := s.createDoc(ctx, "Other")
	otherDocChunk := s.createDocChunks(ctx, otherDoc, "other one", "other two")[1]
	s.editDocChunkBody(ctx, otherDocChunk, "other two edited")
	foreignDoc := s.createDoc(otherCtx, "Foreign")
	foreignChunk := s.createDocChunks(otherCtx, foreignDoc, "foreign one", "foreign two")[1]
	s.editDocChunkBody(otherCtx, foreignChunk, "foreign two edited")

	cases := map[string]string{
		"unknown document":          fmt.Sprintf("/web/docs/999999999/chunks/%d", ownChunk),
		"unknown chunk":             fmt.Sprintf("/web/docs/%d/chunks/999999999", docID),
		"chunk of another document": fmt.Sprintf("/web/docs/%d/chunks/%d", docID, otherDocChunk),
		"other user's chunk":        fmt.Sprintf("/web/docs/%d/chunks/%d", docID, foreignChunk),
		"other user's document":     fmt.Sprintf("/web/docs/%d/chunks/%d", foreignDoc, foreignChunk),
	}
	assertUnchanged := func() {
		s.assertDocChunks(ctx, otherDoc, "other one", "other two edited")
		s.assertDocChunks(otherCtx, foreignDoc, "foreign one", "foreign two edited")
		assert.Equal(s.T(), util.Ptr("other two"), s.getDocChunk(ctx, otherDocChunk).PreviousBody)
		assert.Equal(s.T(), util.Ptr("foreign two"), s.getDocChunk(otherCtx, foreignChunk).PreviousBody)
	}
	return cases, assertUnchanged
}

func (s *IntegrationTestSuite) TestDocChunkSaveWeb() {
	ctx := s.Context()
	r := s.docsRouter(ctx)

	docID := s.createDoc(ctx, "Plan")
	ids := s.createDocChunks(ctx, docID, "first", "second")

	s.Run("overwrites the body and redirects to the chunk's anchor", func() {
		w := docsPost(r, fmt.Sprintf("/web/docs/%d/chunks/%d", docID, ids[0]), url.Values{"body": {"first\r\nedited"}})

		require.Equal(s.T(), http.StatusSeeOther, w.Code)
		assert.Equal(s.T(), fmt.Sprintf("/web/docs/%d#chunk-%d", docID, ids[0]), w.Header().Get("Location"))
		chunk := s.getDocChunk(ctx, ids[0])
		assert.Equal(s.T(), "first\nedited", chunk.Body)
		assert.Equal(s.T(), util.Ptr("first"), chunk.PreviousBody)
	})

	s.Run("body over a limit re-renders in edit mode", func() {
		body := "entered text\n" + strings.Repeat("a", docMaxLineLength+1)

		w := docsPost(r, fmt.Sprintf("/web/docs/%d/chunks/%d", docID, ids[1]), url.Values{"body": {body}})

		assert.Equal(s.T(), http.StatusBadRequest, w.Code)
		html := w.Body.String()
		assert.Contains(s.T(), html, "line 2 of body has 201 characters")
		assert.Contains(s.T(), html, fmt.Sprintf(`<form method="POST" action="/web/docs/%d/chunks/%d">`, docID, ids[1]))
		assert.Contains(s.T(), html, "entered text\naaa")
		assert.Contains(s.T(), html, "required autofocus>")
		chunk := s.getDocChunk(ctx, ids[1])
		assert.Equal(s.T(), "second", chunk.Body)
		assert.Nil(s.T(), chunk.PreviousBody)
	})

	cases, assertUnchanged := s.docChunkNotFoundCases(ctx, docID, ids[1])
	for name, target := range cases {
		s.Run(name+" is 404", func() {
			w := docsPost(r, target, url.Values{"body": {"hacked"}})

			assert.Equal(s.T(), http.StatusNotFound, w.Code)
			assertUnchanged()
			assert.Equal(s.T(), "second", s.getDocChunk(ctx, ids[1]).Body)
		})
	}
}

func (s *IntegrationTestSuite) TestDocChunkMoveWeb() {
	ctx := s.Context()
	r := s.docsRouter(ctx)

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
			docID := s.createDoc(ctx, "Plan")
			ids := s.createDocChunks(ctx, docID, "a", "b", "c")

			w := docsPost(r, fmt.Sprintf("/web/docs/%d/chunks/%d/move", docID, ids[tt.index]), url.Values{"direction": {tt.direction}})

			assert.Equal(s.T(), tt.wantCode, w.Code)
			if tt.wantCode == http.StatusSeeOther {
				assert.Equal(s.T(), fmt.Sprintf("/web/docs/%d#chunk-%d", docID, ids[tt.index]), w.Header().Get("Location"))
			}
			s.assertDocChunks(ctx, docID, tt.want...)
		})
	}

	docID := s.createDoc(ctx, "Plan")
	ids := s.createDocChunks(ctx, docID, "a", "b", "c")
	cases, assertUnchanged := s.docChunkNotFoundCases(ctx, docID, ids[1])
	for name, target := range cases {
		s.Run(name+" is 404", func() {
			w := docsPost(r, target+"/move", url.Values{"direction": {"up"}})

			assert.Equal(s.T(), http.StatusNotFound, w.Code)
			assertUnchanged()
			s.assertDocChunks(ctx, docID, "a", "b", "c")
		})
	}
}

func (s *IntegrationTestSuite) TestDocChunkRevertWeb() {
	ctx := s.Context()
	r := s.docsRouter(ctx)

	docID := s.createDoc(ctx, "Plan")
	ids := s.createDocChunks(ctx, docID, "never edited", "old")
	s.editDocChunkBody(ctx, ids[1], "new")

	s.Run("swaps the bodies and redirects", func() {
		w := docsPost(r, fmt.Sprintf("/web/docs/%d/chunks/%d/revert", docID, ids[1]), nil)

		require.Equal(s.T(), http.StatusSeeOther, w.Code)
		assert.Equal(s.T(), fmt.Sprintf("/web/docs/%d#chunk-%d", docID, ids[1]), w.Header().Get("Location"))
		chunk := s.getDocChunk(ctx, ids[1])
		assert.Equal(s.T(), "old", chunk.Body)
		assert.Equal(s.T(), util.Ptr("new"), chunk.PreviousBody)
	})

	s.Run("chunk with no previous body is 400", func() {
		w := docsPost(r, fmt.Sprintf("/web/docs/%d/chunks/%d/revert", docID, ids[0]), nil)

		assert.Equal(s.T(), http.StatusBadRequest, w.Code)
		chunk := s.getDocChunk(ctx, ids[0])
		assert.Equal(s.T(), "never edited", chunk.Body)
		assert.Nil(s.T(), chunk.PreviousBody)
	})

	cases, assertUnchanged := s.docChunkNotFoundCases(ctx, docID, ids[1])
	for name, target := range cases {
		s.Run(name+" is 404", func() {
			w := docsPost(r, target+"/revert", nil)

			assert.Equal(s.T(), http.StatusNotFound, w.Code)
			assertUnchanged()
			assert.Equal(s.T(), "old", s.getDocChunk(ctx, ids[1]).Body)
		})
	}
}

func (s *IntegrationTestSuite) TestDocChunkDeleteWeb() {
	ctx := s.Context()
	r := s.docsRouter(ctx)

	s.Run("deletes the chunk and redirects to the document", func() {
		docID := s.createDoc(ctx, "Plan")
		ids := s.createDocChunks(ctx, docID, "a", "b", "c")

		w := docsPost(r, fmt.Sprintf("/web/docs/%d/chunks/%d/delete", docID, ids[1]), nil)

		require.Equal(s.T(), http.StatusSeeOther, w.Code)
		assert.Equal(s.T(), fmt.Sprintf("/web/docs/%d", docID), w.Header().Get("Location"))
		s.assertDocChunks(ctx, docID, "a", "c")
	})

	docID := s.createDoc(ctx, "Plan")
	ids := s.createDocChunks(ctx, docID, "a", "b")
	cases, assertUnchanged := s.docChunkNotFoundCases(ctx, docID, ids[1])
	for name, target := range cases {
		s.Run(name+" is 404", func() {
			w := docsPost(r, target+"/delete", nil)

			assert.Equal(s.T(), http.StatusNotFound, w.Code)
			assertUnchanged()
			s.assertDocChunks(ctx, docID, "a", "b")
		})
	}
}
