package tests

// Covers docs/functions/docs-spec.md: the list_docs/get_doc MCP tools and
// the GET /web/docs, GET /web/docs/:topic pages.

import (
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/docs"
)

func docsRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/web/docs", docs.IndexWebHandler)
	r.GET("/web/docs/:topic", docs.DocWebHandler)
	return r
}

func (s *IntegrationTestSuite) TestListDocs_DiscoversEmbeddedContent() {
	_, out, err := docs.ListDocs(s.Context(), nil, docs.ListDocsInput{})
	require.NoError(s.T(), err)

	bySlug := map[string]docs.Topic{}
	for _, t := range out.Topics {
		bySlug[t.Slug] = t
	}
	require.Contains(s.T(), bySlug, "activity-mechanics")
	require.Contains(s.T(), bySlug, "activity-rituals")

	mechanics := bySlug["activity-mechanics"]
	assert.Equal(s.T(), "Механика системы", mechanics.Title, "title is the first # heading")
	assert.NotEmpty(s.T(), mechanics.Description, "description is the first blockquote line after the heading")
	assert.NotContains(s.T(), mechanics.Description, ">")
}

func (s *IntegrationTestSuite) TestGetDoc() {
	tests := []struct {
		name        string
		topic       string
		wantErr     bool
		wantContent string
	}{
		{name: "known topic returns raw markdown", topic: "activity-rituals", wantContent: "# Ритуалы и процесс"},
		{name: "unknown topic", topic: "nope", wantErr: true},
		{name: "empty topic", topic: "", wantErr: true},
		{name: "path traversal rejected", topic: "../content/activity-rituals", wantErr: true},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, out, err := docs.GetDoc(s.Context(), nil, docs.GetDocInput{Topic: tt.topic})
			if tt.wantErr {
				assert.Error(s.T(), err)
				return
			}
			require.NoError(s.T(), err)
			assert.Contains(s.T(), out.Content, tt.wantContent)
		})
	}
}

func (s *IntegrationTestSuite) TestDocsWeb() {
	tests := []struct {
		name         string
		path         string
		wantStatus   int
		wantContains []string
	}{
		{
			name:         "index lists docs with links",
			path:         "/web/docs",
			wantStatus:   http.StatusOK,
			wantContains: []string{`href="/web/docs/activity-mechanics"`, `href="/web/docs/activity-rituals"`, "Механика системы", `href="/web/docs"`},
		},
		{
			name:         "doc page renders markdown to HTML",
			path:         "/web/docs/activity-mechanics",
			wantStatus:   http.StatusOK,
			wantContains: []string{"<h1>Механика системы</h1>", "<table>", "<blockquote>", "docs-sidebar", `href="/web/docs/activity-rituals"`, `href="/web/docs/activity-mechanics" aria-current="page"`},
		},
		{
			name:       "unknown topic is 404",
			path:       "/web/docs/nope",
			wantStatus: http.StatusNotFound,
		},
	}
	r := docsRouter()
	for _, tt := range tests {
		s.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(s.T(), tt.wantStatus, w.Code)
			for _, want := range tt.wantContains {
				assert.Contains(s.T(), w.Body.String(), want)
			}
		})
	}
}
