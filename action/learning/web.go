// Package learning's web pages: GET /web/learning lists the plans, GET
// /web/learning/:id is a plan page with its skills and stats, GET
// /web/learning/:id/exercises is the read-only exercise history, GET/POST
// /web/learning/:id/review are the flashcards, and the vocabulary routes
// list, search, add, edit and delete the plan's words.
package learning

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"personal/action/webui"
	"personal/domain"
	"personal/gateways"
	"personal/util"
)

var learningNav = webui.BuildNav(webui.NavLearning)

const (
	historyPageSize    = 20
	vocabularyPageSize = 50
	webDayLayout       = "2006-01-02"
)

// flashcardButtons are the review buttons, in display order.
var flashcardButtons = []flashcardButton{
	{Label: "Recalled", Score: 5},
	{Label: "With a hint", Score: 3},
	{Label: "Couldn't recall", Score: 2},
	{Label: "Wrong / guessed", Score: 1},
}

type flashcardButton struct {
	Label string
	Score int
}

type planListData struct {
	Filters []filterLink
	Table   template.HTML
	Empty   bool
}

type filterLink struct {
	Label  string
	URL    string
	Active bool
}

type planPageData struct {
	PlanID        int64
	Description   string
	ExerciseTiles template.HTML
	WordTiles     template.HTML
	DueWords      int
	HasWords      bool
	Skills        template.HTML
	NoSkills      bool
}

type historyData struct {
	BackURL    string
	Heading    string
	Planned    []plannedCard
	Cards      []historyCard
	Pagination *webui.PaginationData
}

// plannedCard is one exercise without an answer; Status is «Missed», «Today» or empty for a later day.
type plannedCard struct {
	Title     string
	Meta      string
	PlannedOn string
	Status    string
	Content   []string
}

type historyCard struct {
	Title      string
	Meta       string
	Score      string
	Content    []string
	Answer     []string
	Note       []string
	HasNote    bool
	AnsweredOn string
}

type reviewPageData struct {
	PlanID     int64
	Error      string
	Card       *reviewCard
	Left       int
	Buttons    []flashcardButton
	BackToPlan string
	ActionURL  string
}

type reviewCard struct {
	ReviewID    int64
	Term        string
	Translation string
	Notes       []string
}

type wordFormData struct {
	Action      string
	Submit      string
	Term        string
	Translation string
	Notes       string
	SkillID     string
	Skills      []skillOption
	Error       string
}

type skillOption struct {
	ID       string
	Name     string
	Selected bool
}

type vocabularyPageData struct {
	PlanID      int64
	Form        wordFormData
	Query       string
	SearchError string
	Table       template.HTML
	Empty       string
}

type wordEditData struct {
	PlanID    int64
	BackURL   string
	Form      wordFormData
	Mastery   string
	Uncertain string
	History   template.HTML
	NoHistory bool
	DeleteURL string
}

// A newline right after <textarea> is dropped by the browser, so the
// templates add one to keep a leading newline of the text itself.
const pagesSrc = `{{define "lines"}}{{range $i, $line := .}}{{if $i}}<br>{{end}}{{$line}}{{end}}{{end}}

{{define "planList"}}<p>{{range $i, $f := .Filters}}{{if $i}} · {{end}}{{if $f.Active}}<strong>{{$f.Label}}</strong>{{else}}<a href="{{$f.URL}}">{{$f.Label}}</a>{{end}}{{end}}</p>
{{if .Empty}}<p>No learning plans.</p>{{else}}{{.Table}}{{end}}{{end}}

{{define "planPage"}}<p><a href="/web/learning">← Learning</a></p>
{{if .Description}}<p>{{.Description}}</p>
{{end}}<p><a href="/web/learning/{{.PlanID}}/vocabulary">Vocabulary</a> · <a href="/web/learning/{{.PlanID}}/exercises">Exercises</a></p>
<h3>Exercises</h3>
{{.ExerciseTiles}}
{{if .HasWords}}<h3>Vocabulary</h3>
{{.WordTiles}}
<p><a role="button" href="/web/learning/{{.PlanID}}/review">Review {{.DueWords}} words</a></p>
{{end}}<h3>Skills</h3>
{{if .NoSkills}}<p>No skills yet.</p>{{else}}{{.Skills}}{{end}}{{end}}

{{define "history"}}<p><a href="{{.BackURL}}">← Plan</a></p>
{{if .Heading}}<h2>{{.Heading}}</h2>
{{end}}<h3>Planned</h3>
{{range .Planned}}<article>
<header><strong>{{.Title}}</strong>{{if .Meta}} · {{.Meta}}{{end}}<br><small>Planned {{.PlannedOn}}{{if .Status}} · <mark>{{.Status}}</mark>{{end}}</small></header>
<p>{{template "lines" .Content}}</p>
</article>
{{else}}<p>No planned exercises.</p>
{{end}}<h3>History</h3>
{{range .Cards}}<article>
<header><strong>{{.Title}}</strong>{{if .Meta}} · {{.Meta}}{{end}}<br><small>Answered {{.AnsweredOn}} · {{.Score}}</small></header>
<h6>Exercise</h6>
<p>{{template "lines" .Content}}</p>
<h6>Answer</h6>
<p>{{template "lines" .Answer}}</p>
{{if .HasNote}}<h6>Note</h6>
<p>{{template "lines" .Note}}</p>
{{end}}</article>
{{else}}<p>No answered exercises yet.</p>
{{end}}{{with .Pagination}}<p>{{if .PrevURL}}<a href="{{.PrevURL}}">← Newer</a> · {{end}}Page {{.Page}} of {{.TotalPages}}{{if .NextURL}} · <a href="{{.NextURL}}">Older →</a>{{end}}</p>
{{end}}{{end}}

{{define "review"}}<p><a href="{{.BackToPlan}}">← Plan</a></p>
{{if .Error}}<p role="alert"><mark>{{.Error}}</mark></p>
{{end}}{{with .Card}}<article>
<header><small>{{$.Left}} left</small></header>
<h2>{{.Term}}</h2>
<details>
<summary>Show answer</summary>
<p><strong>{{.Translation}}</strong></p>
{{if .Notes}}<p>{{template "lines" .Notes}}</p>
{{end}}<form method="POST" action="{{$.ActionURL}}">
<input type="hidden" name="review_id" value="{{.ReviewID}}">
<div role="group">{{range $.Buttons}}<button type="submit" name="score" value="{{.Score}}">{{.Label}}</button>{{end}}</div>
</form>
</details>
</article>
{{else}}<p>Nothing to review right now.</p>
{{end}}{{end}}

{{define "wordForm"}}<form method="POST" action="{{.Action}}">
{{if .Error}}<p role="alert"><mark>{{.Error}}</mark></p>
{{end}}<div class="grid">
<label>Term<input type="text" name="term" value="{{.Term}}" required></label>
<label>Translation<input type="text" name="translation" value="{{.Translation}}" required></label>
</div>
<label>Notes<textarea name="notes" rows="2">
{{.Notes}}</textarea></label>
<label>Skill<select name="skill_id">
<option value="">No skill</option>
{{range .Skills}}<option value="{{.ID}}"{{if .Selected}} selected{{end}}>{{.Name}}</option>
{{end}}</select></label>
<button type="submit">{{.Submit}}</button>
</form>{{end}}

{{define "vocabulary"}}<p><a href="/web/learning/{{.PlanID}}">← Plan</a></p>
<details{{if .Form.Error}} open{{end}}>
<summary>Add a word</summary>
{{template "wordForm" .Form}}
</details>
<form method="GET" action="/web/learning/{{.PlanID}}/vocabulary">
<fieldset role="group">
<input type="search" name="q" value="{{.Query}}" placeholder="Words or phrases, comma-separated" aria-label="Search words">
<button type="submit">Search</button>
</fieldset>
</form>
{{if .SearchError}}<p role="alert"><mark>{{.SearchError}}</mark></p>
{{else if .Empty}}<p>{{.Empty}}</p>
{{else}}{{.Table}}{{end}}{{end}}

{{define "wordEdit"}}<p><a href="{{.BackURL}}">← Vocabulary</a></p>
{{template "wordForm" .Form}}
<p>Mastery {{.Mastery}} · uncertainty {{.Uncertain}}</p>
<h3>Reviews</h3>
{{if .NoHistory}}<p>No reviews yet.</p>{{else}}{{.History}}{{end}}
<details>
<summary>Delete this word</summary>
<form method="POST" action="{{.DeleteURL}}">
<p>The word and its whole review history are deleted.</p>
<button type="submit" class="secondary">Delete</button>
</form>
</details>{{end}}`

var pages = template.Must(template.New("learningPages").Parse(pagesSrc))

func execute(name string, data any) (template.HTML, error) {
	var b strings.Builder
	if err := pages.ExecuteTemplate(&b, name, data); err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil
}

func writePage(c *gin.Context, status int, title, name string, data any) {
	content, err := execute(name, data)
	if err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(status)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:    title,
		Nav:      learningNav,
		UserName: c.GetString("user_name"),
		Content:  content,
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}

// formatMastery shows mastery with its debuff, e.g. "80% (−20%)".
func formatMastery(m domain.MasteryState) string {
	if m.Debuff == 0 {
		return fmt.Sprintf("%d%%", m.Mastery)
	}
	return fmt.Sprintf("%d%% (−%d%%)", m.Mastery, m.Debuff)
}

func formatWebDay(t *time.Time) string {
	if t == nil {
		return "—"
	}
	return t.In(location()).Format(webDayLayout)
}

func formatScore(score *int) string {
	if score == nil {
		return "—"
	}
	return strconv.Itoa(*score)
}

// splitLines splits text into lines for the "lines" template; form input has CRLF line endings.
func splitLines(s string) []string {
	return strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
}

func planURL(planID int64) string {
	return fmt.Sprintf("/web/learning/%d", planID)
}

// loadPlanParam loads the user's plan from the :id route param, answering 404 itself when there is none.
func loadPlanParam(c *gin.Context) (gateways.DB, *domain.LearningPlan, bool) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return nil, nil, false
	}

	planID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusNotFound, "Learning plan not found")
		return nil, nil, false
	}

	plan, err := db.GetLearningPlan(ctx, planID, webui.CurrentUserID(c))
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load learning plan: %v", err)
		return nil, nil, false
	}
	if plan == nil {
		c.String(http.StatusNotFound, "Learning plan not found")
		return nil, nil, false
	}

	return db, plan, true
}

// loadWordParam loads the word from the :word_id route param, answering 404 itself when it is not in the plan.
func loadWordParam(c *gin.Context, db gateways.DB, plan *domain.LearningPlan) (*domain.VocabularyWord, bool) {
	wordID, err := strconv.ParseInt(c.Param("word_id"), 10, 64)
	if err != nil {
		c.String(http.StatusNotFound, "Word not found")
		return nil, false
	}

	words, err := db.ListVocabulary(c.Request.Context(), domain.VocabularyFilter{UserID: plan.UserID, LearningPlanID: &plan.ID, IDs: []int64{wordID}})
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load word: %v", err)
		return nil, false
	}
	if len(words) == 0 {
		c.String(http.StatusNotFound, "Word not found")
		return nil, false
	}

	return &withWordMastery(words, time.Now())[0], true
}

// --- GET /web/learning -----------------------------------------------------

// PlansWebHandler renders GET /web/learning: one row per plan, filtered by ?status=.
func PlansWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}

	status := c.Query("status")
	var statuses []domain.LearningPlanStatus
	if status != "" {
		if !validPlanStatuses[domain.LearningPlanStatus(status)] {
			status = ""
		} else {
			statuses = []domain.LearningPlanStatus{domain.LearningPlanStatus(status)}
		}
	}

	plans, err := listPlanSummaries(ctx, db, webui.CurrentUserID(c), statuses, time.Now())
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load learning plans: %v", err)
		return
	}

	filters := []filterLink{{Label: "Active & paused", URL: "/web/learning", Active: status == ""}}
	for _, s := range domain.LearningPlanStatuses {
		filters = append(filters, filterLink{
			Label:  strings.ToUpper(string(s)[:1]) + string(s)[1:],
			URL:    "/web/learning?status=" + string(s),
			Active: status == string(s),
		})
	}

	rows := make([]webui.TableRow, len(plans))
	for i, p := range plans {
		rows[i] = webui.TableRow{
			Cells: []string{
				p.Name,
				p.Status,
				strconv.Itoa(p.SkillCount),
				strconv.Itoa(p.ExerciseStats.Due),
				strconv.Itoa(p.VocabularyStats.Due),
			},
			LinkURL: planURL(p.ID),
		}
	}
	table := webui.RenderTable(webui.TableData{
		Columns: []webui.TableColumn{
			{Label: "Plan"},
			{Label: "Status"},
			{Label: "Skills", Align: "right"},
			{Label: "Exercises due", Align: "right"},
			{Label: "Words due", Align: "right"},
		},
		Rows: rows,
	})

	writePage(c, http.StatusOK, "Learning", "planList", planListData{Filters: filters, Table: table, Empty: len(plans) == 0})
}

// --- GET /web/learning/:id -------------------------------------------------

// PlanWebHandler renders GET /web/learning/:id: stats and the skills table.
func PlanWebHandler(c *gin.Context) {
	db, plan, ok := loadPlanParam(c)
	if !ok {
		return
	}

	detail, err := planDetail(c.Request.Context(), db, plan.UserID, plan.ID, time.Now())
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load learning plan: %v", err)
		return
	}

	rows := make([]webui.TableRow, len(detail.Skills))
	for i, s := range detail.Skills {
		var lastPracticed *time.Time
		if s.LastPracticedAt != "" {
			t, _ := time.Parse(time.RFC3339, s.LastPracticedAt)
			lastPracticed = &t
		}
		rows[i] = webui.TableRow{
			Cells: []string{
				s.Name,
				formatMastery(s.MasteryState),
				fmt.Sprintf("%d%%", s.Uncertainty),
				strconv.Itoa(s.AnsweredCount),
				strconv.Itoa(s.PlannedCount),
				formatWebDay(lastPracticed),
				formatScore(s.LastScore),
			},
			LinkURL: fmt.Sprintf("/web/learning/%d/exercises?skill_id=%d", plan.ID, s.ID),
		}
	}

	ex, voc := detail.ExerciseStats, detail.VocabularyStats
	data := planPageData{
		PlanID:      plan.ID,
		Description: deref(plan.Description),
		ExerciseTiles: webui.RenderStatTiles([]webui.StatTileData{
			{Label: "Due", Value: strconv.Itoa(ex.Due), SubLabel: fmt.Sprintf("%d planned", ex.Planned), Emphasis: ex.Due > 0},
			{Label: "Awaiting review", Value: strconv.Itoa(ex.AwaitingReview)},
		}),
		HasWords: voc.Total > 0,
		DueWords: voc.Due,
		Skills: webui.RenderTable(webui.TableData{
			Columns: []webui.TableColumn{
				{Label: "Skill"},
				{Label: "Mastery", Align: "right"},
				{Label: "Uncertainty", Align: "right"},
				{Label: "Answered", Align: "right"},
				{Label: "Planned", Align: "right"},
				{Label: "Last practiced"},
				{Label: "Last score", Align: "right"},
			},
			Rows: rows,
		}),
		NoSkills: len(rows) == 0,
	}
	if data.HasWords {
		data.WordTiles = webui.RenderStatTiles([]webui.StatTileData{
			{Label: "Words", Value: strconv.Itoa(voc.Total)},
			{Label: "Due", Value: strconv.Itoa(voc.Due), Emphasis: voc.Due > 0},
			{Label: "Unscheduled", Value: strconv.Itoa(voc.Unscheduled)},
			{Label: "Mastered", Value: strconv.Itoa(voc.Mastered)},
			{Label: "Average mastery", Value: fmt.Sprintf("%d%%", voc.AverageMastery)},
		})
	}

	writePage(c, http.StatusOK, plan.Name, "planPage", data)
}

// --- GET /web/learning/:id/exercises ---------------------------------------

// ExercisesWebHandler renders GET /web/learning/:id/exercises: exercises of
// the plan, or of one skill with ?skill_id=N — planned ones earliest first,
// then the answered history, newest first and paginated.
func ExercisesWebHandler(c *gin.Context) {
	db, plan, ok := loadPlanParam(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()

	filter := domain.LearningExerciseFilter{
		UserID:         plan.UserID,
		LearningPlanID: &plan.ID,
		States:         []domain.LearningExerciseState{domain.LearningExerciseAnswered, domain.LearningExerciseReviewed},
	}
	data := historyData{BackURL: planURL(plan.ID)}
	baseURL := planURL(plan.ID) + "/exercises?"

	if raw := c.Query("skill_id"); raw != "" {
		skillID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			c.String(http.StatusNotFound, "Skill not found")
			return
		}
		skill, err := db.GetSkill(ctx, skillID, plan.UserID)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load skill: %v", err)
			return
		}
		if skill == nil || skill.LearningPlanID != plan.ID {
			c.String(http.StatusNotFound, "Skill not found")
			return
		}
		filter.SkillID = &skill.ID
		data.Heading = skill.Name
		baseURL += "skill_id=" + raw + "&"
	}

	planned, err := db.ListLearningExercises(ctx, domain.LearningExerciseFilter{
		UserID:         plan.UserID,
		LearningPlanID: &plan.ID,
		SkillID:        filter.SkillID,
		States:         []domain.LearningExerciseState{domain.LearningExercisePlanned},
		OldestFirst:    true,
	})
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load planned exercises: %v", err)
		return
	}
	todayStart := today(time.Now())
	for _, e := range planned {
		card := plannedCard{
			Title:     e.SkillName,
			Meta:      deref(e.Kind),
			PlannedOn: formatWebDay(&e.PlannedFor),
			Content:   splitLines(e.Content),
		}
		switch {
		case e.PlannedFor.Before(todayStart):
			card.Status = "Missed"
		case e.PlannedFor.Equal(todayStart):
			card.Status = "Today"
		}
		data.Planned = append(data.Planned, card)
	}

	total, err := db.CountLearningExercises(ctx, filter)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to count exercises: %v", err)
		return
	}
	page := webui.ParsePage(c)
	filter.Limit = historyPageSize
	filter.Offset = (page - 1) * historyPageSize
	exercises, err := db.ListLearningExercises(ctx, filter)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load exercises: %v", err)
		return
	}

	for _, e := range exercises {
		card := historyCard{
			Title:      e.SkillName,
			Meta:       deref(e.Kind),
			Score:      "Not reviewed",
			AnsweredOn: formatWebDay(e.AnsweredAt),
			Content:    splitLines(e.Content),
			Answer:     splitLines(deref(e.Answer)),
		}
		if e.Score != nil {
			card.Score = fmt.Sprintf("Score %d/5", *e.Score)
		}
		if e.Note != nil {
			card.Note = splitLines(*e.Note)
			card.HasNote = true
		}
		data.Cards = append(data.Cards, card)
	}
	data.Pagination = webui.BuildPagination(page, total, historyPageSize, func(p int) string {
		return fmt.Sprintf("%spage=%d", baseURL, p)
	})

	writePage(c, http.StatusOK, plan.Name+" — exercises", "history", data)
}

// --- GET, POST /web/learning/:id/review ------------------------------------

// ReviewWebHandler renders GET /web/learning/:id/review: the earliest due pending review as a flashcard.
func ReviewWebHandler(c *gin.Context) {
	db, plan, ok := loadPlanParam(c)
	if !ok {
		return
	}
	renderReviewPage(c, db, plan, "")
}

// RecordReviewWebHandler handles POST /web/learning/:id/review: records the
// posted flashcard score on the posted review, then 303 back to the next card.
func RecordReviewWebHandler(c *gin.Context) {
	db, plan, ok := loadPlanParam(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()

	reviewID, err := strconv.ParseInt(c.PostForm("review_id"), 10, 64)
	if err != nil {
		c.String(http.StatusNotFound, "Review not found")
		return
	}
	reviews, err := db.ListVocabularyReviews(ctx, domain.VocabularyReviewFilter{UserID: plan.UserID, LearningPlanID: &plan.ID, IDs: []int64{reviewID}})
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load review: %v", err)
		return
	}
	if len(reviews) == 0 {
		c.String(http.StatusNotFound, "Review not found")
		return
	}
	review := reviews[0]

	now := time.Now()
	score, err := strconv.Atoi(c.PostForm("score"))
	switch {
	case err != nil || !slices.Contains(domain.FlashcardScores, score):
		renderReviewPage(c, db, plan, "Unknown answer button.")
		return
	case review.Score != nil:
		renderReviewPage(c, db, plan, fmt.Sprintf("%q is already reviewed.", review.Term))
		return
	case review.PlannedFor.After(now):
		renderReviewPage(c, db, plan, fmt.Sprintf("%q is not due yet.", review.Term))
		return
	}

	if err := db.RecordVocabularyReview(ctx, review.ID, plan.UserID, score, now.UTC()); err != nil {
		renderReviewPage(c, db, plan, err.Error())
		return
	}

	c.Redirect(http.StatusSeeOther, planURL(plan.ID)+"/review")
}

func renderReviewPage(c *gin.Context, db gateways.DB, plan *domain.LearningPlan, errMsg string) {
	now := time.Now()
	due, err := db.ListVocabularyReviews(c.Request.Context(), domain.VocabularyReviewFilter{
		UserID:         plan.UserID,
		LearningPlanID: &plan.ID,
		Pending:        util.Ptr(true),
		DueAt:          &now,
	})
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load reviews: %v", err)
		return
	}

	data := reviewPageData{
		PlanID:     plan.ID,
		Error:      errMsg,
		Left:       len(due),
		Buttons:    flashcardButtons,
		BackToPlan: planURL(plan.ID),
		ActionURL:  planURL(plan.ID) + "/review",
	}
	if len(due) > 0 {
		r := due[0]
		data.Card = &reviewCard{ReviewID: r.ID, Term: r.Term, Translation: r.Translation}
		if r.Notes != nil {
			data.Card.Notes = splitLines(*r.Notes)
		}
	}

	status := http.StatusOK
	if errMsg != "" {
		status = http.StatusBadRequest
	}
	writePage(c, status, plan.Name+" — review", "review", data)
}

// --- GET, POST /web/learning/:id/vocabulary --------------------------------

// VocabularyWebHandler renders GET /web/learning/:id/vocabulary.
func VocabularyWebHandler(c *gin.Context) {
	db, plan, ok := loadPlanParam(c)
	if !ok {
		return
	}
	renderVocabularyPage(c, db, plan, wordFormData{})
}

// AddVocabularyWebHandler handles POST /web/learning/:id/vocabulary: adds
// one word through the same validation as add_vocabulary, then 303 back.
func AddVocabularyWebHandler(c *gin.Context) {
	db, plan, ok := loadPlanParam(c)
	if !ok {
		return
	}

	form := wordFormData{
		Term:        c.PostForm("term"),
		Translation: c.PostForm("translation"),
		Notes:       c.PostForm("notes"),
		SkillID:     c.PostForm("skill_id"),
	}
	in := wordInput{Term: form.Term, Translation: form.Translation, Notes: &form.Notes}
	if form.SkillID != "" {
		skillID, err := strconv.ParseInt(form.SkillID, 10, 64)
		if err != nil {
			form.Error = "invalid skill"
			renderVocabularyPage(c, db, plan, form)
			return
		}
		in.SkillID = &skillID
	}

	if _, err := addVocabulary(c.Request.Context(), db, plan.UserID, plan.ID, []wordInput{in}, time.Now()); err != nil {
		form.Error = err.Error()
		renderVocabularyPage(c, db, plan, form)
		return
	}

	c.Redirect(http.StatusSeeOther, planURL(plan.ID)+"/vocabulary")
}

func renderVocabularyPage(c *gin.Context, db gateways.DB, plan *domain.LearningPlan, form wordFormData) {
	ctx := c.Request.Context()
	now := time.Now()

	skills, err := db.ListSkills(ctx, plan.ID, plan.UserID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load skills: %v", err)
		return
	}
	skillNames := make(map[int64]string, len(skills))
	for _, s := range skills {
		skillNames[s.ID] = s.Name
	}

	form.Action = planURL(plan.ID) + "/vocabulary"
	form.Submit = "Add"
	form.Skills = skillOptions(skills, form.SkillID)

	query := c.Query("q")
	var phrases []string
	for _, p := range strings.Split(query, ",") {
		if p = strings.TrimSpace(p); p != "" {
			phrases = append(phrases, p)
		}
	}

	data := vocabularyPageData{PlanID: plan.ID, Form: form, Query: query}
	page := webui.ParsePage(c)
	var words []domain.VocabularyWord
	var total int

	switch {
	case len(phrases) > 5:
		data.SearchError = "maximum 5 comma-separated phrases allowed"
	case len(phrases) > 0:
		matches, err := searchVocabulary(ctx, db, plan.UserID, phrases, &plan.ID, now)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to search words: %v", err)
			return
		}
		total = len(matches)
		start := min((page-1)*vocabularyPageSize, total)
		for _, m := range matches[start:min(start+vocabularyPageSize, total)] {
			words = append(words, m.Word)
		}
		data.Empty = "No words found."
	default:
		filter := domain.VocabularyFilter{UserID: plan.UserID, LearningPlanID: &plan.ID}
		if total, err = db.CountVocabulary(ctx, filter); err != nil {
			c.String(http.StatusInternalServerError, "Failed to count words: %v", err)
			return
		}
		filter.Limit = vocabularyPageSize
		filter.Offset = (page - 1) * vocabularyPageSize
		if words, err = db.ListVocabulary(ctx, filter); err != nil {
			c.String(http.StatusInternalServerError, "Failed to load words: %v", err)
			return
		}
		words = withWordMastery(words, now)
		data.Empty = "No words yet."
	}

	if len(words) > 0 {
		data.Empty = ""
	}
	rows := make([]webui.TableRow, len(words))
	for i, w := range words {
		skill := ""
		if w.SkillID != nil {
			skill = skillNames[*w.SkillID]
		}
		rows[i] = webui.TableRow{
			Cells: []string{
				w.Term,
				w.Translation,
				deref(w.Notes),
				skill,
				formatMastery(w.MasteryState),
				fmt.Sprintf("%d%%", w.Uncertainty),
				formatScore(w.LastScore),
				formatWebDay(w.NextPlannedFor),
			},
			LinkURL: fmt.Sprintf("%s/vocabulary/%d", planURL(plan.ID), w.ID),
		}
	}
	data.Table = webui.RenderTable(webui.TableData{
		Columns: []webui.TableColumn{
			{Label: "Term"},
			{Label: "Translation"},
			{Label: "Notes"},
			{Label: "Skill"},
			{Label: "Mastery", Align: "right"},
			{Label: "Uncertainty", Align: "right"},
			{Label: "Last score", Align: "right"},
			{Label: "Next review"},
		},
		Rows: rows,
		Pagination: webui.BuildPagination(page, total, vocabularyPageSize, func(p int) string {
			params := url.Values{"page": {strconv.Itoa(p)}}
			if query != "" {
				params.Set("q", query)
			}
			return planURL(plan.ID) + "/vocabulary?" + params.Encode()
		}),
	})

	status := http.StatusOK
	if form.Error != "" || data.SearchError != "" {
		status = http.StatusBadRequest
	}
	writePage(c, status, plan.Name+" — vocabulary", "vocabulary", data)
}

func skillOptions(skills []domain.Skill, selected string) []skillOption {
	options := make([]skillOption, len(skills))
	for i, s := range skills {
		id := strconv.FormatInt(s.ID, 10)
		options[i] = skillOption{ID: id, Name: s.Name, Selected: id == selected}
	}
	return options
}

// --- GET, POST /web/learning/:id/vocabulary/:word_id -----------------------

// WordWebHandler renders GET /web/learning/:id/vocabulary/:word_id: the edit form, mastery and review history.
func WordWebHandler(c *gin.Context) {
	db, plan, ok := loadPlanParam(c)
	if !ok {
		return
	}
	word, ok := loadWordParam(c, db, plan)
	if !ok {
		return
	}

	form := wordFormData{Term: word.Term, Translation: word.Translation, Notes: deref(word.Notes)}
	if word.SkillID != nil {
		form.SkillID = strconv.FormatInt(*word.SkillID, 10)
	}
	renderWordPage(c, db, plan, word, form)
}

// EditWordWebHandler handles POST /web/learning/:id/vocabulary/:word_id
// through the same validation as edit_vocabulary, then 303 to the vocabulary list.
func EditWordWebHandler(c *gin.Context) {
	db, plan, ok := loadPlanParam(c)
	if !ok {
		return
	}
	word, ok := loadWordParam(c, db, plan)
	if !ok {
		return
	}
	original := *word

	form := wordFormData{
		Term:        c.PostForm("term"),
		Translation: c.PostForm("translation"),
		Notes:       c.PostForm("notes"),
		SkillID:     c.PostForm("skill_id"),
	}
	skillID := int64(0)
	if form.SkillID != "" {
		id, err := strconv.ParseInt(form.SkillID, 10, 64)
		if err != nil {
			form.Error = "invalid skill"
			renderWordPage(c, db, plan, &original, form)
			return
		}
		skillID = id
	}

	_, err := editVocabulary(c.Request.Context(), db, plan.UserID, word, wordChanges{
		Term:        &form.Term,
		Translation: &form.Translation,
		Notes:       &form.Notes,
		SkillID:     &skillID,
	}, time.Now())
	if err != nil {
		form.Error = err.Error()
		renderWordPage(c, db, plan, &original, form)
		return
	}

	c.Redirect(http.StatusSeeOther, planURL(plan.ID)+"/vocabulary")
}

func renderWordPage(c *gin.Context, db gateways.DB, plan *domain.LearningPlan, word *domain.VocabularyWord, form wordFormData) {
	ctx := c.Request.Context()

	skills, err := db.ListSkills(ctx, plan.ID, plan.UserID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load skills: %v", err)
		return
	}
	reviews, err := db.ListVocabularyReviews(ctx, domain.VocabularyReviewFilter{UserID: plan.UserID, VocabularyID: &word.ID})
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load reviews: %v", err)
		return
	}

	wordURL := fmt.Sprintf("%s/vocabulary/%d", planURL(plan.ID), word.ID)
	form.Action = wordURL
	form.Submit = "Save"
	form.Skills = skillOptions(skills, form.SkillID)

	rows := make([]webui.TableRow, len(reviews))
	for i, r := range reviews {
		planned := r.PlannedFor
		rows[i] = webui.TableRow{Cells: []string{formatWebDay(&planned), formatScore(r.Score), formatWebDay(r.ReviewedAt)}}
	}

	data := wordEditData{
		PlanID:    plan.ID,
		BackURL:   planURL(plan.ID) + "/vocabulary",
		Form:      form,
		Mastery:   formatMastery(word.MasteryState),
		Uncertain: fmt.Sprintf("%d%%", word.Uncertainty),
		History: webui.RenderTable(webui.TableData{
			Columns: []webui.TableColumn{{Label: "Planned"}, {Label: "Score", Align: "right"}, {Label: "Reviewed"}},
			Rows:    rows,
		}),
		NoHistory: len(rows) == 0,
		DeleteURL: wordURL + "/delete",
	}

	status := http.StatusOK
	if form.Error != "" {
		status = http.StatusBadRequest
	}
	writePage(c, status, word.Term, "wordEdit", data)
}

// DeleteWordWebHandler handles POST /web/learning/:id/vocabulary/:word_id/delete.
func DeleteWordWebHandler(c *gin.Context) {
	db, plan, ok := loadPlanParam(c)
	if !ok {
		return
	}
	word, ok := loadWordParam(c, db, plan)
	if !ok {
		return
	}

	if err := db.DeleteVocabulary(c.Request.Context(), word.ID, plan.UserID); err != nil {
		c.String(http.StatusInternalServerError, "Failed to delete word: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, planURL(plan.ID)+"/vocabulary")
}
