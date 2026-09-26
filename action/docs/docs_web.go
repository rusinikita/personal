package docs

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/russross/blackfriday/v2"

	"personal/action/webui"
)

var docsNav = webui.BuildNav(webui.NavDocs)

const docsIndexSrc = `<h1>Docs</h1>
{{range .}}<article><h3><a href="/web/docs/{{.Slug}}">{{.Title}}</a></h3><p>{{.Description}}</p></article>
{{else}}<p>No docs yet.</p>{{end}}`

var docsIndexTemplate = template.Must(template.New("docsIndex").Parse(docsIndexSrc))

const docLayoutSrc = `<div class="docs-layout">
<aside class="docs-sidebar"><nav><ul>
{{range .Topics}}<li><a href="/web/docs/{{.Slug}}"{{if eq .Slug $.Current}} aria-current="page"{{end}}>{{.Title}}</a></li>
{{end}}</ul></nav></aside>
<div>{{.Doc}}</div>
</div>`

var docLayoutTemplate = template.Must(template.New("docLayout").Parse(docLayoutSrc))

// IndexWebHandler handles GET /web/docs: lists every discovered doc with its
// title and one-line description, linking to each doc's page.
func IndexWebHandler(c *gin.Context) {
	topics, err := Topics()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load docs: %v", err)
		return
	}
	var b strings.Builder
	if err := docsIndexTemplate.Execute(&b, topics); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
		return
	}
	renderDocsPage(c, "Docs", template.HTML(b.String()))
}

// DocWebHandler handles GET /web/docs/:topic: renders one doc's markdown to
// HTML under the shared webui layout, next to a sidebar linking every doc.
// 404 for an unknown topic.
func DocWebHandler(c *gin.Context) {
	slug := c.Param("topic")
	content, err := Content(slug)
	if err != nil {
		c.String(http.StatusNotFound, "Doc not found")
		return
	}
	title, _ := parseHeader(content)
	if title == "" {
		title = slug
	}
	topics, err := Topics()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load docs: %v", err)
		return
	}
	html := blackfriday.Run(content, blackfriday.WithExtensions(blackfriday.CommonExtensions))
	var b strings.Builder
	if err := docLayoutTemplate.Execute(&b, struct {
		Topics  []Topic
		Current string
		Doc     template.HTML
	}{topics, slug, template.HTML(html)}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
		return
	}
	renderDocsPage(c, title, template.HTML(b.String()))
}

func renderDocsPage(c *gin.Context, title string, content template.HTML) {
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:    title,
		Nav:      docsNav,
		UserName: c.GetString("user_name"),
		Content:  content,
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}
