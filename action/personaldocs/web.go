// Package personaldocs' web pages: GET /web/personal-docs lists the
// documents, GET /web/personal-docs/new and POST /web/personal-docs create
// one, and GET /web/personal-docs/:id is the document page, whose modes
// (edit a chunk, add a chunk, confirm a revert or a delete, edit the title
// and description) are query params served by the POST routes below it.
package personaldocs

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"personal/action/webui"
	"personal/domain"
	"personal/gateways"
)

var personalDocsNav = webui.BuildNav(webui.NavPersonalDocs)

// displayTimezone is the timezone of the dates shown on the pages.
const displayTimezone = "Asia/Nicosia"

const (
	// minBodyRows and maxBodyRows bound the height of a body textarea.
	minBodyRows = 6
	maxBodyRows = 30
)

// docFormData feeds the title and description form, both on the create page
// and in document edit mode.
type docFormData struct {
	Action      string
	CancelURL   string
	Submit      string
	Title       string
	Description string
	MaxLength   int
	Error       string
}

// bodyFormData feeds the body form of edit mode and new chunk mode.
type bodyFormData struct {
	Action        string
	CancelURL     string
	Body          string
	Rows          int
	MaxLines      int
	MaxLineLength int
	Error         string
}

// confirmData feeds the confirmation block of a revert or a delete.
type confirmData struct {
	Action    string
	CancelURL string
}

// chunkView is one chunk of the document page: a form in edit mode,
// otherwise its rendered body, an optional confirmation block and its
// action bar.
type chunkView struct {
	ID           int64
	HTML         template.HTML
	Actions      template.HTML
	Form         *bodyFormData
	Revert       *confirmData
	PreviousBody string
	Delete       *confirmData
}

// contentsItem is one heading in the table of contents, with the deeper
// headings that follow it nested under it.
type contentsItem struct {
	Text     string
	URL      string
	Children []*contentsItem
}

type docPageData struct {
	DocID       int64
	Title       string
	Description string
	DocActions  template.HTML
	DocForm     *docFormData
	Contents    []*contentsItem
	Chunks      []chunkView
	NewForm     *bodyFormData
}

// A newline right after <textarea> or <pre> is dropped by the browser, so
// the templates add one to keep a leading newline of the text itself.
const pagesSrc = `{{define "docForm"}}<form method="POST" action="{{.Action}}">
{{if .Error}}<p role="alert"><mark>{{.Error}}</mark></p>
{{end}}<label for="doc-title">Title</label>
<input id="doc-title" type="text" name="title" value="{{.Title}}" required>
<label for="doc-description">Description</label>
<input id="doc-description" type="text" name="description" value="{{.Description}}" maxlength="{{.MaxLength}}" required>
<small>One line saying what the document is for, up to {{.MaxLength}} characters.</small>
<div role="group"><button type="submit">{{.Submit}}</button><a role="button" class="secondary" href="{{.CancelURL}}">Cancel</a></div>
</form>{{end}}

{{define "bodyForm"}}<form method="POST" action="{{.Action}}">
{{if .Error}}<p role="alert"><mark>{{.Error}}</mark></p>
{{end}}<textarea name="body" rows="{{.Rows}}" aria-label="Chunk text" required autofocus>
{{.Body}}</textarea>
<small>Markdown, up to {{.MaxLines}} lines of up to {{.MaxLineLength}} characters.</small>
<div role="group"><button type="submit">Save</button><a role="button" class="secondary" href="{{.CancelURL}}">Cancel</a></div>
</form>{{end}}

{{define "docsList"}}<p><a role="button" href="/web/personal-docs/new">New document</a></p>
{{if .Empty}}<p>No documents yet.</p>{{else}}{{.Table}}{{end}}{{end}}

{{define "newDoc"}}<p><a href="/web/personal-docs">← Personal docs</a></p>
{{template "docForm" .}}{{end}}

{{define "contents"}}<ul>
{{range .}}<li><a href="{{.URL}}">{{.Text}}</a>{{if .Children}}{{template "contents" .Children}}{{end}}</li>
{{end}}</ul>{{end}}

{{define "docPage"}}<p><a href="/web/personal-docs">← Personal docs</a></p>
<article>
{{if .DocForm}}{{template "docForm" .DocForm}}
{{else}}<hgroup><h2>{{.Title}}</h2><p>{{.Description}}</p></hgroup>
<footer>{{.DocActions}}</footer>
{{end}}</article>
{{if .Contents}}<details open>
<summary>Contents</summary>
{{template "contents" .Contents}}
</details>
{{end}}{{range .Chunks}}<article id="chunk-{{.ID}}">
{{if .Form}}{{template "bodyForm" .Form}}
{{else}}{{.HTML}}
{{if .Revert}}<hr>
<p>Revert this chunk to its previous text?</p>
<pre>
{{.PreviousBody}}</pre>
<form method="POST" action="{{.Revert.Action}}">
<div role="group"><button type="submit">Revert</button><a role="button" class="secondary" href="{{.Revert.CancelURL}}">Cancel</a></div>
</form>
{{end}}{{if .Delete}}<hr>
<p>Delete this chunk? This cannot be undone.</p>
<form method="POST" action="{{.Delete.Action}}">
<div role="group"><button type="submit">Delete</button><a role="button" class="secondary" href="{{.Delete.CancelURL}}">Cancel</a></div>
</form>
{{end}}<footer>{{.Actions}}</footer>
{{end}}</article>
{{end}}{{if .NewForm}}<article id="chunk-new">
{{template "bodyForm" .NewForm}}
</article>
{{else}}<p><a role="button" href="/web/personal-docs/{{.DocID}}?new=1#chunk-new">Add chunk</a></p>
{{end}}{{end}}`

var pagesTemplate = template.Must(template.New("personalDocs").Parse(pagesSrc))

func execute(name string, data any) (template.HTML, error) {
	var b strings.Builder
	if err := pagesTemplate.ExecuteTemplate(&b, name, data); err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil
}

// docPageMode is what the document page shows besides the plain view; the
// zero value is view mode. The POST handlers pass the entered values and
// the error to re-render the mode their form was sent from.
type docPageMode struct {
	editDoc     bool
	title       string
	description string

	editChunkID   int64
	newChunk      bool
	body          string // text of the edit or the new chunk form
	revertChunkID int64
	deleteChunkID int64

	err string
}

// isView reports view mode: no form and no confirmation on the page.
func (m docPageMode) isView() bool {
	return !m.editDoc && m.editChunkID == 0 && !m.newChunk && m.revertChunkID == 0 && m.deleteChunkID == 0
}

func docURL(docID int64) string {
	return fmt.Sprintf("/web/personal-docs/%d", docID)
}

func chunkAnchorURL(docID, chunkID int64) string {
	return fmt.Sprintf("/web/personal-docs/%d#chunk-%d", docID, chunkID)
}

func chunkURL(docID, chunkID int64) string {
	return fmt.Sprintf("/web/personal-docs/%d/chunks/%d", docID, chunkID)
}

// PersonalDocsWebHandler renders GET /web/personal-docs: the user's
// documents as a table.
func PersonalDocsWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}

	docs, err := db.ListPersonalDocs(ctx, webui.CurrentUserID(c))
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load documents: %v", err)
		return
	}

	loc, err := time.LoadLocation(displayTimezone)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load timezone: %v", err)
		return
	}

	table := webui.TableData{
		Columns: []webui.TableColumn{
			{Label: "Title", Align: "left"},
			{Label: "Description", Align: "left"},
			{Label: "Updated", Align: "left"},
		},
	}
	for _, doc := range docs {
		table.Rows = append(table.Rows, webui.TableRow{
			Cells:   []string{doc.Title, doc.Description, doc.UpdatedAt.In(loc).Format("2006-01-02")},
			LinkURL: docURL(doc.ID),
		})
	}

	content, err := execute("docsList", struct {
		Empty bool
		Table template.HTML
	}{len(docs) == 0, webui.RenderTable(table)})
	if err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
		return
	}

	writePage(c, http.StatusOK, "Personal docs", content)
}

// NewPersonalDocWebHandler renders GET /web/personal-docs/new.
func NewPersonalDocWebHandler(c *gin.Context) {
	renderNewDocPage(c, "", "", "")
}

// CreatePersonalDocWebHandler handles POST /web/personal-docs: creates the
// document, then 303 to its page; an invalid value re-renders the form with
// an inline error.
func CreatePersonalDocWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}

	title := c.PostForm("title")
	description := c.PostForm("description")
	doc, err := createDoc(ctx, db, webui.CurrentUserID(c), title, description)
	if err != nil {
		renderNewDocPage(c, title, description, err.Error())
		return
	}

	c.Redirect(http.StatusSeeOther, docURL(doc.ID))
}

func renderNewDocPage(c *gin.Context, title, description, errText string) {
	content, err := execute("newDoc", docFormData{
		Action:      "/web/personal-docs",
		CancelURL:   "/web/personal-docs",
		Submit:      "Create",
		Title:       title,
		Description: description,
		MaxLength:   maxLineLength,
		Error:       errText,
	})
	if err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
		return
	}

	status := http.StatusOK
	if errText != "" {
		status = http.StatusBadRequest
	}
	writePage(c, status, "New document", content)
}

// PersonalDocWebHandler renders GET /web/personal-docs/:id in the mode
// chosen by the query params; a GET never changes data.
func PersonalDocWebHandler(c *gin.Context) {
	db, doc, ok := loadDoc(c)
	if !ok {
		return
	}

	mode := docPageMode{
		editDoc:     c.Query("edit_doc") != "",
		title:       doc.Title,
		description: doc.Description,
		newChunk:    c.Query("new") != "",
	}

	if raw := c.Query("edit"); raw != "" {
		chunk, ok := loadChunk(c, db, doc, raw)
		if !ok {
			return
		}
		mode.editChunkID = chunk.ID
		mode.body = chunk.Body
	}

	if raw := c.Query("revert"); raw != "" {
		chunk, ok := loadChunk(c, db, doc, raw)
		if !ok {
			return
		}
		if chunk.PreviousBody == nil {
			c.String(http.StatusBadRequest, "Chunk has no previous text to revert to")
			return
		}
		mode.revertChunkID = chunk.ID
	}

	if raw := c.Query("delete"); raw != "" {
		chunk, ok := loadChunk(c, db, doc, raw)
		if !ok {
			return
		}
		mode.deleteChunkID = chunk.ID
	}

	renderDocPage(c, db, doc, mode)
}

// EditPersonalDocWebHandler handles POST /web/personal-docs/:id: saves the
// title and description, then 303 back to the document.
func EditPersonalDocWebHandler(c *gin.Context) {
	db, doc, ok := loadDoc(c)
	if !ok {
		return
	}

	title := c.PostForm("title")
	description := c.PostForm("description")
	if _, err := editDoc(c.Request.Context(), db, doc.UserID, doc.ID, &title, &description); err != nil {
		renderDocPage(c, db, doc, docPageMode{editDoc: true, title: title, description: description, err: err.Error()})
		return
	}

	c.Redirect(http.StatusSeeOther, docURL(doc.ID))
}

// CreatePersonalDocChunkWebHandler handles POST
// /web/personal-docs/:id/chunks: adds a chunk at the end, then 303 to its
// anchor.
func CreatePersonalDocChunkWebHandler(c *gin.Context) {
	db, doc, ok := loadDoc(c)
	if !ok {
		return
	}

	body := c.PostForm("body")
	chunk, err := createChunk(c.Request.Context(), db, doc.UserID, doc.ID, body, nil)
	if err != nil {
		renderDocPage(c, db, doc, docPageMode{newChunk: true, body: body, err: err.Error()})
		return
	}

	c.Redirect(http.StatusSeeOther, chunkAnchorURL(doc.ID, chunk.ID))
}

// SavePersonalDocChunkWebHandler handles POST
// /web/personal-docs/:id/chunks/:chunk_id: replaces the chunk's body, then
// 303 to its anchor.
func SavePersonalDocChunkWebHandler(c *gin.Context) {
	db, doc, chunk, ok := loadDocChunk(c)
	if !ok {
		return
	}

	body := c.PostForm("body")
	if _, err := editChunk(c.Request.Context(), db, doc.UserID, chunkEdit{ChunkID: chunk.ID, Body: &body}); err != nil {
		renderDocPage(c, db, doc, docPageMode{editChunkID: chunk.ID, body: body, err: err.Error()})
		return
	}

	c.Redirect(http.StatusSeeOther, chunkAnchorURL(doc.ID, chunk.ID))
}

// MovePersonalDocChunkWebHandler handles POST
// /web/personal-docs/:id/chunks/:chunk_id/move: moves the chunk one
// position up or down, then 303 to its anchor.
func MovePersonalDocChunkWebHandler(c *gin.Context) {
	db, doc, chunk, ok := loadDocChunk(c)
	if !ok {
		return
	}

	position := chunk.Position
	switch c.PostForm("direction") {
	case "up":
		position--
	case "down":
		position++
	default:
		c.String(http.StatusBadRequest, "direction must be up or down")
		return
	}

	// editChunk rejects a position before the first or after the last chunk
	if _, err := editChunk(c.Request.Context(), db, doc.UserID, chunkEdit{ChunkID: chunk.ID, Position: &position}); err != nil {
		c.String(http.StatusBadRequest, "Failed to move chunk: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, chunkAnchorURL(doc.ID, chunk.ID))
}

// RevertPersonalDocChunkWebHandler handles POST
// /web/personal-docs/:id/chunks/:chunk_id/revert, sent from the revert
// confirmation: swaps the body with the previous body, then 303 to the
// chunk's anchor.
func RevertPersonalDocChunkWebHandler(c *gin.Context) {
	db, doc, chunk, ok := loadDocChunk(c)
	if !ok {
		return
	}

	if _, err := revertChunk(c.Request.Context(), db, doc.UserID, chunk.ID); err != nil {
		c.String(http.StatusBadRequest, "Failed to revert chunk: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, chunkAnchorURL(doc.ID, chunk.ID))
}

// DeletePersonalDocChunkWebHandler handles POST
// /web/personal-docs/:id/chunks/:chunk_id/delete, sent from the delete
// confirmation: deletes the chunk, then 303 to the document.
func DeletePersonalDocChunkWebHandler(c *gin.Context) {
	db, doc, chunk, ok := loadDocChunk(c)
	if !ok {
		return
	}

	if err := deleteChunk(c.Request.Context(), db, doc.UserID, chunk.ID); err != nil {
		c.String(http.StatusInternalServerError, "Failed to delete chunk: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, docURL(doc.ID))
}

// loadDoc loads the user's document from the :id route param, answering
// 404 itself when there is none.
func loadDoc(c *gin.Context) (gateways.DB, *domain.PersonalDoc, bool) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return nil, nil, false
	}

	docID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusNotFound, "Document not found")
		return nil, nil, false
	}

	doc, err := db.GetPersonalDoc(ctx, docID, webui.CurrentUserID(c))
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load document: %v", err)
		return nil, nil, false
	}
	if doc == nil {
		c.String(http.StatusNotFound, "Document not found")
		return nil, nil, false
	}

	return db, doc, true
}

// loadChunk loads the chunk with the id rawID, answering 404 itself when
// there is none in this document.
func loadChunk(c *gin.Context, db gateways.DB, doc *domain.PersonalDoc, rawID string) (*domain.PersonalDocChunk, bool) {
	chunkID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		c.String(http.StatusNotFound, "Chunk not found")
		return nil, false
	}

	chunk, err := db.GetPersonalDocChunk(c.Request.Context(), chunkID, doc.UserID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load chunk: %v", err)
		return nil, false
	}
	if chunk == nil || chunk.DocID != doc.ID {
		c.String(http.StatusNotFound, "Chunk not found")
		return nil, false
	}

	return chunk, true
}

// loadDocChunk loads the document and its chunk from the :id and :chunk_id
// route params.
func loadDocChunk(c *gin.Context) (gateways.DB, *domain.PersonalDoc, *domain.PersonalDocChunk, bool) {
	db, doc, ok := loadDoc(c)
	if !ok {
		return nil, nil, nil, false
	}

	chunk, ok := loadChunk(c, db, doc, c.Param("chunk_id"))
	if !ok {
		return nil, nil, nil, false
	}

	return db, doc, chunk, true
}

func renderDocPage(c *gin.Context, db gateways.DB, doc *domain.PersonalDoc, mode docPageMode) {
	chunks, err := db.ListPersonalDocChunks(c.Request.Context(), doc.ID, doc.UserID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load chunks: %v", err)
		return
	}

	page := docPageData{DocID: doc.ID, Title: doc.Title, Description: doc.Description}

	if mode.editDoc {
		page.DocForm = &docFormData{
			Action:      docURL(doc.ID),
			CancelURL:   docURL(doc.ID),
			Submit:      "Save",
			Title:       mode.title,
			Description: mode.description,
			MaxLength:   maxLineLength,
			Error:       mode.err,
		}
	} else {
		page.DocActions = webui.RenderActionBar(webui.ActionBarData{Buttons: []webui.ActionButton{
			{Label: "✎", Title: "Edit title and description", URL: docURL(doc.ID) + "?edit_doc=1"},
		}})
	}

	var headings []webui.MarkdownHeading
	for _, chunk := range chunks {
		view := chunkView{ID: chunk.ID}
		anchorURL := chunkAnchorURL(doc.ID, chunk.ID)

		if chunk.ID == mode.editChunkID {
			view.Form = newBodyForm(chunkURL(doc.ID, chunk.ID), anchorURL, mode.body, mode.err)
			page.Chunks = append(page.Chunks, view)
			continue
		}

		var chunkHeadings []webui.MarkdownHeading
		view.HTML, chunkHeadings = webui.RenderMarkdownWithHeadings(chunk.Body, true, fmt.Sprintf("chunk-%d-", chunk.ID))
		headings = append(headings, chunkHeadings...)
		view.Actions = webui.RenderActionBar(chunkActions(doc.ID, chunk, len(chunks)))
		if chunk.ID == mode.revertChunkID && chunk.PreviousBody != nil {
			view.Revert = &confirmData{Action: chunkURL(doc.ID, chunk.ID) + "/revert", CancelURL: anchorURL}
			view.PreviousBody = *chunk.PreviousBody
		}
		if chunk.ID == mode.deleteChunkID {
			view.Delete = &confirmData{Action: chunkURL(doc.ID, chunk.ID) + "/delete", CancelURL: anchorURL}
		}
		page.Chunks = append(page.Chunks, view)
	}

	if mode.isView() {
		page.Contents = buildContents(headings)
	}

	if mode.newChunk {
		page.NewForm = newBodyForm(docURL(doc.ID)+"/chunks", docURL(doc.ID), mode.body, mode.err)
	}

	content, err := execute("docPage", page)
	if err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
		return
	}

	status := http.StatusOK
	if mode.err != "" {
		status = http.StatusBadRequest
	}
	writePage(c, status, doc.Title, content)
}

// chunkActions are the icon buttons of one chunk in view mode: the moves
// act at once, the rest open a page mode.
func chunkActions(docID int64, chunk domain.PersonalDocChunk, chunkCount int) webui.ActionBarData {
	moveURL := chunkURL(docID, chunk.ID) + "/move"
	modeURL := func(mode string) string {
		return fmt.Sprintf("/web/personal-docs/%d?%s=%d#chunk-%d", docID, mode, chunk.ID, chunk.ID)
	}

	var buttons []webui.ActionButton
	if chunk.Position > 1 {
		buttons = append(buttons, webui.ActionButton{
			Label: "↑", Title: "Move up", URL: moveURL, Method: "POST",
			Fields: []webui.ActionField{{Name: "direction", Value: "up"}},
		})
	}
	if chunk.Position < chunkCount {
		buttons = append(buttons, webui.ActionButton{
			Label: "↓", Title: "Move down", URL: moveURL, Method: "POST",
			Fields: []webui.ActionField{{Name: "direction", Value: "down"}},
		})
	}
	buttons = append(buttons, webui.ActionButton{Label: "✎", Title: "Edit", URL: modeURL("edit")})
	if chunk.PreviousBody != nil {
		buttons = append(buttons, webui.ActionButton{Label: "↶", Title: "Revert the last edit", URL: modeURL("revert")})
	}
	buttons = append(buttons, webui.ActionButton{Label: "✕", Title: "Delete", URL: modeURL("delete")})

	return webui.ActionBarData{Buttons: buttons}
}

// buildContents nests the headings by level: a heading goes under the
// nearest heading above it that has a lower level.
func buildContents(headings []webui.MarkdownHeading) []*contentsItem {
	var roots []*contentsItem
	var parents []*contentsItem
	var levels []int
	for _, heading := range headings {
		for len(levels) > 0 && levels[len(levels)-1] >= heading.Level {
			parents = parents[:len(parents)-1]
			levels = levels[:len(levels)-1]
		}
		item := &contentsItem{Text: heading.Text, URL: "#" + heading.ID}
		if len(parents) == 0 {
			roots = append(roots, item)
		} else {
			parent := parents[len(parents)-1]
			parent.Children = append(parent.Children, item)
		}
		parents = append(parents, item)
		levels = append(levels, heading.Level)
	}
	return roots
}

func newBodyForm(action, cancelURL, body, errText string) *bodyFormData {
	rows := strings.Count(body, "\n") + 2
	return &bodyFormData{
		Action:        action,
		CancelURL:     cancelURL,
		Body:          body,
		Rows:          min(max(rows, minBodyRows), maxBodyRows),
		MaxLines:      maxChunkLines,
		MaxLineLength: maxLineLength,
		Error:         errText,
	}
}

func writePage(c *gin.Context, status int, title string, content template.HTML) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(status)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:    title,
		Nav:      personalDocsNav,
		UserName: c.GetString("user_name"),
		Content:  content,
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}
