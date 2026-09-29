// Package ideas' web page: GET /web/ideas is a quick capture form plus the
// open ideas grouped by status, POST /web/ideas creates an idea.
package ideas

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"personal/action/webui"
	"personal/domain"
	"personal/gateways"
)

var ideasNav = webui.BuildNav(webui.NavIdeas)

// captureFormData feeds captureFormTemplate.
type captureFormData struct {
	Body  string
	Error string
}

// captureFormSrc is the quick capture form — a local html/template
// constant, same convention as pointFormContentSrc (action/progress).
const captureFormSrc = `{{if .Error}}<p style="color: var(--pico-del-color)">{{.Error}}</p>{{end}}
<form method="POST" action="/web/ideas">
    <label for="idea-body">New idea</label>
    <textarea id="idea-body" name="body" rows="3" required>{{.Body}}</textarea>
    <button type="submit">Capture</button>
</form>`

var captureFormTemplate = template.Must(template.New("ideaCaptureForm").Parse(captureFormSrc))

func renderCaptureForm(data captureFormData) (template.HTML, error) {
	var b strings.Builder
	if err := captureFormTemplate.Execute(&b, data); err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil
}

// IdeasWebHandler renders GET /web/ideas.
func IdeasWebHandler(c *gin.Context) {
	renderIdeasPage(c, captureFormData{})
}

// CreateIdeaWebHandler handles POST /web/ideas: creates an idea in inbox,
// then 303 back to GET /web/ideas; an empty body re-renders the page with
// an inline error.
func CreateIdeaWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}

	body := c.PostForm("body")
	if _, err := createIdea(ctx, db, webui.CurrentUserID(c), body); err != nil {
		renderIdeasPage(c, captureFormData{Body: body, Error: err.Error()})
		return
	}

	c.Redirect(http.StatusSeeOther, "/web/ideas")
}

func renderIdeasPage(c *gin.Context, form captureFormData) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)

	open, err := db.ListIdeas(ctx, domain.IdeaFilter{UserID: userID})
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load ideas: %v", err)
		return
	}
	blocked, err := db.ListIdeas(ctx, domain.IdeaFilter{UserID: userID, Resolutions: []domain.IdeaResolution{domain.IdeaResolutionBlocked}})
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load ideas: %v", err)
		return
	}

	byStatus := make(map[domain.IdeaStatus][]domain.Idea)
	for _, idea := range open {
		byStatus[idea.Status] = append(byStatus[idea.Status], idea)
	}

	var rows []webui.TableRow
	for _, status := range domain.IdeaOpenStatuses {
		rows = appendIdeaSection(rows, string(status), byStatus[status])
	}
	rows = appendIdeaSection(rows, string(domain.IdeaResolutionBlocked), blocked)

	formHTML, err := renderCaptureForm(form)
	if err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
		return
	}

	table := webui.TableData{
		Columns: []webui.TableColumn{
			{Label: "Idea"},
			{Label: "Created"},
			{Label: "Surfaced", Align: "right"},
		},
		Rows: rows,
	}

	status := http.StatusOK
	if form.Error != "" {
		status = http.StatusBadRequest
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(status)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:    "Ideas",
		Nav:      ideasNav,
		UserName: c.GetString("user_name"),
		Content:  formHTML + webui.RenderTable(table),
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}

// appendIdeaSection adds a heading row plus one row per idea; empty
// sections are skipped.
func appendIdeaSection(rows []webui.TableRow, heading string, ideas []domain.Idea) []webui.TableRow {
	if len(ideas) == 0 {
		return rows
	}
	rows = append(rows, webui.TableRow{Heading: heading})
	for _, idea := range ideas {
		surfaced := ""
		if idea.SurfaceCount > 1 {
			surfaced = fmt.Sprintf("×%d", idea.SurfaceCount)
		}
		rows = append(rows, webui.TableRow{Cells: []string{idea.Body, idea.CreatedAt.Format("2006-01-02"), surfaced}})
	}
	return rows
}
