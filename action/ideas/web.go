// Package ideas' web pages: GET /web/ideas is a quick capture form plus the
// inbox ideas as cards, POST /web/ideas creates an idea, GET
// /web/ideas/spike shows the spike ideas, GET /web/ideas/search finds ideas
// in any status.
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

// searchLatestLimit is how many newest ideas the search page shows for an
// empty query.
const searchLatestLimit = 50

// cardsPerRow is the number of cards in one Pico .grid row.
const cardsPerRow = 3

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
</form>
<p><a href="/web/ideas/spike">Spike ideas</a> · <a href="/web/ideas/search">Search ideas</a></p>`

var captureFormTemplate = template.Must(template.New("ideaCaptureForm").Parse(captureFormSrc))

// searchFormData feeds searchFormTemplate.
type searchFormData struct {
	Query string
	Error string
}

// searchFormSrc is a GET form, so a search is a plain URL.
const searchFormSrc = `<p><a href="/web/ideas">← Ideas</a></p>
<form method="GET" action="/web/ideas/search">
<fieldset role="group">
<input type="search" name="q" value="{{.Query}}" placeholder="Words or phrases, comma-separated" aria-label="Search ideas">
<button type="submit">Search</button>
</fieldset>
</form>
{{if .Error}}<p style="color: var(--pico-del-color)">{{.Error}}</p>{{end}}`

var searchFormTemplate = template.Must(template.New("ideaSearchForm").Parse(searchFormSrc))

// ideaCard is one idea rendered as a Pico <article>; a nil card pads the
// last grid row. Each paragraph is a list of lines joined by <br>. The
// footer has Labels on the left and Date on the right.
type ideaCard struct {
	Paragraphs [][]string
	Labels     string
	Date       string
}

type cardGridData struct {
	Rows  [][]*ideaCard
	Empty string
}

// cardGridSrc lays cards out with Pico's .grid only: every child of one
// .grid sits in a single row, so each row of cards is its own .grid.
const cardGridSrc = `{{range .Rows}}<div class="grid">
{{range .}}{{if .}}<article>
{{range .Paragraphs}}<p>{{range $i, $line := .}}{{if $i}}<br>{{end}}{{$line}}{{end}}</p>
{{end}}<footer style="display: flex; justify-content: space-between"><span>{{.Labels}}</span><span>{{.Date}}</span></footer>
</article>
{{else}}<div></div>
{{end}}{{end}}</div>
{{else}}<p>{{.Empty}}</p>
{{end}}`

var cardGridTemplate = template.Must(template.New("ideaCardGrid").Parse(cardGridSrc))

func execute(t *template.Template, data any) (template.HTML, error) {
	var b strings.Builder
	if err := t.Execute(&b, data); err != nil {
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

	cards, err := newestCards(c, db, domain.IdeaStatusInbox)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load ideas: %v", err)
		return
	}

	formHTML, err := execute(captureFormTemplate, form)
	if err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
		return
	}
	gridHTML, err := renderCardGrid(cards, "Inbox is empty.")
	if err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
		return
	}

	status := http.StatusOK
	if form.Error != "" {
		status = http.StatusBadRequest
	}
	writePage(c, status, "Ideas", formHTML+gridHTML)
}

// SpikeIdeasWebHandler renders GET /web/ideas/spike: the ideas being
// spiked this week, newest first.
func SpikeIdeasWebHandler(c *gin.Context) {
	db := gateways.DBFromContext(c.Request.Context())
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}

	cards, err := newestCards(c, db, domain.IdeaStatusSpike)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load ideas: %v", err)
		return
	}
	gridHTML, err := renderCardGrid(cards, "No spike ideas.")
	if err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
		return
	}

	writePage(c, http.StatusOK, "Spike ideas", `<p><a href="/web/ideas">← Ideas</a></p>`+gridHTML)
}

// newestCards lists the user's ideas in one status as cards without the
// status label, newest first.
func newestCards(c *gin.Context, db gateways.DB, status domain.IdeaStatus) ([]*ideaCard, error) {
	ideas, err := db.ListIdeas(c.Request.Context(), domain.IdeaFilter{UserID: webui.CurrentUserID(c), Statuses: []domain.IdeaStatus{status}})
	if err != nil {
		return nil, err
	}

	cards := make([]*ideaCard, len(ideas))
	for i, idea := range ideas {
		// ListIdeas returns oldest first
		cards[len(ideas)-1-i] = newIdeaCard(idea, false)
	}
	return cards, nil
}

// SearchIdeasWebHandler renders GET /web/ideas/search: an empty q shows the
// newest ideas in any status; otherwise q is split by commas into phrases,
// each a variant of the same search as search_ideas over every status.
func SearchIdeasWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)

	query := c.Query("q")
	var phrases []string
	for _, p := range strings.Split(query, ",") {
		if p = strings.TrimSpace(p); p != "" {
			phrases = append(phrases, p)
		}
	}

	form := searchFormData{Query: query}
	var ideas []domain.Idea
	switch {
	case len(phrases) == 0:
		latest, err := db.ListIdeas(ctx, domain.IdeaFilter{UserID: userID, Statuses: domain.IdeaStatuses, Limit: searchLatestLimit})
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load ideas: %v", err)
			return
		}
		ideas = latest
	case len(phrases) > 5:
		form.Error = "maximum 5 comma-separated phrases allowed"
	default:
		matches, err := searchIdeas(ctx, db, userID, phrases, domain.IdeaStatuses)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to search ideas: %v", err)
			return
		}
		for _, m := range matches {
			ideas = append(ideas, m.Idea)
		}
	}

	cards := make([]*ideaCard, len(ideas))
	for i, idea := range ideas {
		cards[i] = newIdeaCard(idea, true)
	}

	formHTML, err := execute(searchFormTemplate, form)
	if err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
		return
	}
	var gridHTML template.HTML
	if form.Error == "" {
		if gridHTML, err = renderCardGrid(cards, "No ideas found."); err != nil {
			c.String(http.StatusInternalServerError, "render error: %v", err)
			return
		}
	}

	status := http.StatusOK
	if form.Error != "" {
		status = http.StatusBadRequest
	}
	writePage(c, status, "Search ideas", formHTML+gridHTML)
}

// newIdeaCard renders body paragraphs split into lines, the created date
// and footer labels: status and resolution (withStatus), then the surface
// count when > 1. Bodies captured via the web form have CRLF line endings.
func newIdeaCard(idea domain.Idea, withStatus bool) *ideaCard {
	var labels []string
	if withStatus {
		labels = append(labels, string(idea.Status))
		if idea.Resolution != nil {
			labels = append(labels, string(*idea.Resolution))
		}
	}
	if idea.SurfaceCount > 1 {
		labels = append(labels, fmt.Sprintf("×%d", idea.SurfaceCount))
	}
	card := &ideaCard{Labels: strings.Join(labels, " · "), Date: idea.CreatedAt.Format("2006-01-02")}
	body := strings.ReplaceAll(idea.Body, "\r\n", "\n")
	for _, p := range strings.Split(body, "\n\n") {
		if p = strings.TrimSpace(p); p != "" {
			card.Paragraphs = append(card.Paragraphs, strings.Split(p, "\n"))
		}
	}
	return card
}

// renderCardGrid splits cards into rows of cardsPerRow, padding the last
// row with nil cards so its cards keep the same width.
func renderCardGrid(cards []*ideaCard, empty string) (template.HTML, error) {
	var rows [][]*ideaCard
	for start := 0; start < len(cards); start += cardsPerRow {
		row := make([]*ideaCard, cardsPerRow)
		copy(row, cards[start:min(start+cardsPerRow, len(cards))])
		rows = append(rows, row)
	}
	return execute(cardGridTemplate, cardGridData{Rows: rows, Empty: empty})
}

func writePage(c *gin.Context, status int, title string, content template.HTML) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(status)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:    title,
		Nav:      ideasNav,
		UserName: c.GetString("user_name"),
		Content:  content,
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}
