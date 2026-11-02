package tests

// Covers the /web/learning pages from docs/functions/learning-spec.md.

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

	"personal/action/learning"
	"personal/domain"
	"personal/gateways"
	"personal/util"
)

func (s *IntegrationTestSuite) learningRouter(ctx context.Context) *gin.Engine {
	userID := gateways.UserIDFromContext(ctx)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		reqCtx := gateways.WithDB(c.Request.Context(), s.Repo())
		reqCtx = gateways.WithUserID(reqCtx, userID)
		c.Request = c.Request.WithContext(reqCtx)
		c.Next()
	})
	r.GET("/web/learning", learning.PlansWebHandler)
	r.GET("/web/learning/:id", learning.PlanWebHandler)
	r.GET("/web/learning/:id/exercises", learning.ExercisesWebHandler)
	r.GET("/web/learning/:id/review", learning.ReviewWebHandler)
	r.POST("/web/learning/:id/review", learning.RecordReviewWebHandler)
	r.GET("/web/learning/:id/vocabulary", learning.VocabularyWebHandler)
	r.POST("/web/learning/:id/vocabulary", learning.AddVocabularyWebHandler)
	r.GET("/web/learning/:id/vocabulary/:word_id", learning.WordWebHandler)
	r.POST("/web/learning/:id/vocabulary/:word_id", learning.EditWordWebHandler)
	r.POST("/web/learning/:id/vocabulary/:word_id/delete", learning.DeleteWordWebHandler)
	return r
}

func learningGet(r *gin.Engine, target string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	return w
}

func learningPost(r *gin.Engine, target string, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func (s *IntegrationTestSuite) TestLearningWeb_PlanList() {
	ctx := s.Context()
	r := s.learningRouter(ctx)

	s.Run("empty state", func() {
		w := learningGet(r, "/web/learning")
		require.Equal(s.T(), http.StatusOK, w.Code)
		assert.Contains(s.T(), w.Body.String(), "No learning plans.")
	})

	greek := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	kube := s.seedLearningPlan(ctx, "Kubernetes", domain.LearningPlanPaused)
	s.seedLearningPlan(ctx, "Spanish", domain.LearningPlanCompleted)

	skill := s.seedSkill(ctx, greek, "present tense")
	s.seedExercise(ctx, skill, "a", learningDay(0))
	s.seedExercise(ctx, skill, "b", learningDay(-1))
	s.seedExercise(ctx, skill, "c", learningDay(1))
	for _, term := range []string{"νερό", "ψωμί", "γάλα"} {
		s.seedPendingReview(ctx, s.seedWord(ctx, greek, term, "x"), learningDay(0))
	}

	otherCtx := s.otherUserContext(ctx)
	s.seedLearningPlan(otherCtx, "Foreign plan", domain.LearningPlanActive)

	s.Run("default hides completed, rows link to plan pages with due counts", func() {
		html := learningGet(r, "/web/learning").Body.String()
		assert.Contains(s.T(), html, fmt.Sprintf(`<a href="/web/learning/%d">Greek A1</a>`, greek))
		assert.Contains(s.T(), html, fmt.Sprintf(`<a href="/web/learning/%d">Kubernetes</a>`, kube))
		assert.NotContains(s.T(), html, "Spanish")
		assert.NotContains(s.T(), html, "Foreign plan")
		assert.Contains(s.T(), html, `>2</td>`, "exercises due")
		assert.Contains(s.T(), html, `>3</td>`, "words due")
	})

	s.Run("status filter", func() {
		html := learningGet(r, "/web/learning?status=completed").Body.String()
		assert.Contains(s.T(), html, "Spanish")
		assert.NotContains(s.T(), html, "Greek A1")
	})
}

func (s *IntegrationTestSuite) TestLearningWeb_PlanPage() {
	ctx := s.Context()
	now := time.Now()
	r := s.learningRouter(ctx)

	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	skill := s.seedSkill(ctx, planID, "present tense")
	for i := range 4 {
		s.seedReviewedExercise(ctx, skill, 5, now.AddDate(0, 0, -51).Add(time.Duration(i-3)*time.Minute))
	}
	s.seedExercise(ctx, skill, "due", learningDay(0))
	awaiting := s.seedExercise(ctx, skill, "awaiting", learningDay(-60))
	s.seedAnswer(ctx, awaiting, "y", now.AddDate(0, 0, -51))

	mastered := s.seedWord(ctx, planID, "νερό", "water")
	for i := range 4 {
		s.seedScoredReview(ctx, mastered, 5, now.Add(time.Duration(i-4)*time.Minute))
	}
	s.seedPendingReview(ctx, s.seedWord(ctx, planID, "ψωμί", "bread"), learningDay(0))

	emptyPlan := s.seedLearningPlan(ctx, "Kubernetes", domain.LearningPlanActive)

	otherCtx := s.otherUserContext(ctx)
	foreign := s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive)

	s.Run("skills, exercise tiles and word tiles", func() {
		w := learningGet(r, fmt.Sprintf("/web/learning/%d", planID))
		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()

		assert.Contains(s.T(), html, fmt.Sprintf(`<a href="/web/learning/%d/exercises?skill_id=%d">present tense</a>`, planID, skill))
		assert.Contains(s.T(), html, "80% (−20%)")
		assert.Contains(s.T(), html, ">50%</td>")
		assert.Contains(s.T(), html, fmt.Sprintf(`href="/web/learning/%d/vocabulary"`, planID))
		assert.Contains(s.T(), html, fmt.Sprintf(`href="/web/learning/%d/exercises"`, planID))
		assert.Contains(s.T(), html, "Awaiting review")

		assert.Contains(s.T(), html, "Mastered")
		assert.Contains(s.T(), html, "Average mastery")
		assert.Contains(s.T(), html, `<div class="webui-stat-value">40%</div>`)
		assert.Contains(s.T(), html, fmt.Sprintf(`<a role="button" href="/web/learning/%d/review">Review 1 words</a>`, planID))
	})

	s.Run("plan without words shows no word tiles but the links", func() {
		html := learningGet(r, fmt.Sprintf("/web/learning/%d", emptyPlan)).Body.String()
		assert.NotContains(s.T(), html, "Average mastery")
		assert.NotContains(s.T(), html, "/review")
		assert.Contains(s.T(), html, fmt.Sprintf(`href="/web/learning/%d/vocabulary"`, emptyPlan))
		assert.Contains(s.T(), html, "No skills yet.")
	})

	for _, target := range []string{fmt.Sprintf("/web/learning/%d", foreign), "/web/learning/999999999", "/web/learning/abc"} {
		s.Run("404 for "+target, func() {
			assert.Equal(s.T(), http.StatusNotFound, learningGet(r, target).Code)
		})
	}
}

func (s *IntegrationTestSuite) TestLearningWeb_ExerciseHistory() {
	ctx := s.Context()
	now := time.Now()
	r := s.learningRouter(ctx)

	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	tense := s.seedSkill(ctx, planID, "present tense")
	articles := s.seedSkill(ctx, planID, "articles")

	reviewed, err := s.Repo().CreateLearningExercises(ctx, []domain.LearningExercise{{
		UserID: gateways.UserIDFromContext(ctx), SkillID: tense, Kind: util.Ptr("put into the correct form"),
		Content: "1. Εγώ ___\n2. Εσύ ___", PlannedFor: learningDay(-1),
	}})
	require.NoError(s.T(), err)
	s.seedAnswer(ctx, reviewed[0], "ειμαι\nεχεις", now)
	require.NoError(s.T(), s.Repo().ReviewLearningExercises(ctx, gateways.UserIDFromContext(ctx), []domain.LearningExerciseReview{
		{ExerciseID: reviewed[0], Score: 4, Note: util.Ptr("accents missing"), ReviewedAt: now},
	}))
	unreviewed := s.seedExercise(ctx, articles, "articles exercise", learningDay(-2))
	s.seedAnswer(ctx, unreviewed, "το νερό", now)
	_, err = s.Repo().CreateLearningExercises(ctx, []domain.LearningExercise{{
		UserID: gateways.UserIDFromContext(ctx), SkillID: tense, Kind: util.Ptr("choose the ending"),
		Content: "today line one\ntoday line two", PlannedFor: learningDay(0),
	}})
	require.NoError(s.T(), err)
	s.seedExercise(ctx, articles, "missed exercise", learningDay(-3))
	s.seedExercise(ctx, tense, "later exercise", learningDay(2))

	otherPlan := s.seedLearningPlan(ctx, "Greek A2", domain.LearningPlanActive)
	otherPlanSkill := s.seedSkill(ctx, otherPlan, "x")
	otherExercise := s.seedExercise(ctx, otherPlanSkill, "other plan exercise", learningDay(-1))
	s.seedAnswer(ctx, otherExercise, "z", now)
	s.seedExercise(ctx, otherPlanSkill, "other plan planned", learningDay(1))

	emptyPlan := s.seedLearningPlan(ctx, "Kubernetes", domain.LearningPlanActive)

	otherCtx := s.otherUserContext(ctx)
	foreign := s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive)

	s.Run("plan history newest first with full text", func() {
		w := learningGet(r, fmt.Sprintf("/web/learning/%d/exercises", planID))
		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()

		assert.Contains(s.T(), html, "<p>1. Εγώ ___<br>2. Εσύ ___</p>")
		assert.Contains(s.T(), html, "<p>ειμαι<br>εχεις</p>")
		assert.Contains(s.T(), html, "accents missing")
		assert.Contains(s.T(), html, "put into the correct form")
		assert.Contains(s.T(), html, "Score 4/5")
		assert.Contains(s.T(), html, "Not reviewed")
		assert.Less(s.T(), strings.Index(html, "Εγώ"), strings.Index(html, "articles exercise"))
		assert.NotContains(s.T(), html, "other plan exercise")
		assert.NotContains(s.T(), html, "other plan planned")
	})

	s.Run("planned section earliest first, above the history", func() {
		html := learningGet(r, fmt.Sprintf("/web/learning/%d/exercises", planID)).Body.String()

		plannedIdx := strings.Index(html, "<h3>Planned</h3>")
		missedIdx := strings.Index(html, "missed exercise")
		todayIdx := strings.Index(html, "<p>today line one<br>today line two</p>")
		laterIdx := strings.Index(html, "later exercise")
		historyIdx := strings.Index(html, "<h3>History</h3>")
		require.Greater(s.T(), plannedIdx, -1)
		require.Greater(s.T(), missedIdx, plannedIdx)
		require.Greater(s.T(), todayIdx, missedIdx)
		require.Greater(s.T(), laterIdx, todayIdx)
		require.Greater(s.T(), historyIdx, laterIdx, "planned exercises are not in the history")

		assert.Contains(s.T(), html, "Planned "+learningDate(-3)+" · <mark>Missed</mark>")
		assert.Contains(s.T(), html, "Planned "+learningDate(0)+" · <mark>Today</mark>")
		assert.Contains(s.T(), html, "Planned "+learningDate(2)+"</small>")
		assert.Contains(s.T(), html, "choose the ending")
	})

	s.Run("one skill", func() {
		html := learningGet(r, fmt.Sprintf("/web/learning/%d/exercises?skill_id=%d", planID, articles)).Body.String()
		assert.Contains(s.T(), html, "articles exercise")
		assert.Contains(s.T(), html, "missed exercise")
		assert.NotContains(s.T(), html, "Εγώ")
		assert.NotContains(s.T(), html, "later exercise")
		assert.NotContains(s.T(), html, "today line one")
	})

	s.Run("21 entries split into two pages", func() {
		for i := range 21 {
			id := s.seedExercise(ctx, otherPlanSkill, fmt.Sprintf("paged exercise %02d", i), learningDay(-10-i))
			s.seedAnswer(ctx, id, "a", now)
		}
		first := learningGet(r, fmt.Sprintf("/web/learning/%d/exercises", otherPlan)).Body.String()
		assert.Contains(s.T(), first, "Page 1 of 2")
		assert.Contains(s.T(), first, "paged exercise 00")
		assert.NotContains(s.T(), first, "paged exercise 20")

		second := learningGet(r, fmt.Sprintf("/web/learning/%d/exercises?page=2", otherPlan)).Body.String()
		assert.Contains(s.T(), second, "paged exercise 20")
		assert.NotContains(s.T(), second, "paged exercise 00")
		assert.Contains(s.T(), second, "other plan planned", "planned section stays on every page")
	})

	s.Run("empty state", func() {
		html := learningGet(r, fmt.Sprintf("/web/learning/%d/exercises", emptyPlan)).Body.String()
		assert.Contains(s.T(), html, "No planned exercises.")
		assert.Contains(s.T(), html, "No answered exercises yet.")
	})

	for name, target := range map[string]string{
		"skill outside the plan": fmt.Sprintf("/web/learning/%d/exercises?skill_id=%d", planID, otherPlanSkill),
		"foreign plan":           fmt.Sprintf("/web/learning/%d/exercises", foreign),
	} {
		s.Run(name+" 404s", func() {
			assert.Equal(s.T(), http.StatusNotFound, learningGet(r, target).Code)
		})
	}
}

func (s *IntegrationTestSuite) TestLearningWeb_Review() {
	ctx := s.Context()
	r := s.learningRouter(ctx)

	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	water := s.seedWord(ctx, planID, "νερό", "water")
	_, _, err := learning.EditVocabulary(ctx, nil, learning.EditVocabularyInput{WordID: water, Notes: util.Ptr("neuter")})
	require.NoError(s.T(), err)
	bread := s.seedWord(ctx, planID, "ψωμί", "bread")
	future := s.seedWord(ctx, planID, "μέλλον", "future")
	done := s.seedWord(ctx, planID, "τέλος", "end")

	s.seedPendingReview(ctx, water, learningDay(-2))
	s.seedPendingReview(ctx, water, learningDay(0))
	s.seedPendingReview(ctx, bread, learningDay(-1))
	s.seedPendingReview(ctx, future, learningDay(1))
	s.seedScoredReview(ctx, done, 5, time.Now())

	emptyPlan := s.seedLearningPlan(ctx, "Kubernetes", domain.LearningPlanActive)

	otherCtx := s.otherUserContext(ctx)
	foreign := s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive)

	s.Run("earliest due review with the number left", func() {
		w := learningGet(r, fmt.Sprintf("/web/learning/%d/review", planID))
		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()

		assert.Contains(s.T(), html, "<h2>νερό</h2>")
		assert.Contains(s.T(), html, "3 left", "a word with two due reviews counts twice")
		assert.Contains(s.T(), html, "<details>")
		assert.Contains(s.T(), html, "water")
		assert.Contains(s.T(), html, "neuter")
		for _, button := range []string{
			`value="5">Recalled</button>`,
			`value="3">With a hint</button>`,
			`value="2">Couldn&#39;t recall</button>`,
			`value="1">Wrong / guessed</button>`,
		} {
			assert.Contains(s.T(), html, button)
		}
		assert.Less(s.T(), strings.Index(html, "<details>"), strings.Index(html, "water"), "the answer is inside <details>")
		assert.NotContains(s.T(), html, "μέλλον")
		assert.NotContains(s.T(), html, "τέλος")
	})

	s.Run("empty state", func() {
		html := learningGet(r, fmt.Sprintf("/web/learning/%d/review", emptyPlan)).Body.String()
		assert.Contains(s.T(), html, "Nothing to review right now.")
	})

	s.Run("foreign plan 404s", func() {
		assert.Equal(s.T(), http.StatusNotFound, learningGet(r, fmt.Sprintf("/web/learning/%d/review", foreign)).Code)
	})
}

func (s *IntegrationTestSuite) TestLearningWeb_ReviewSubmit() {
	ctx := s.Context()
	now := time.Now()
	r := s.learningRouter(ctx)

	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	reviewURL := fmt.Sprintf("/web/learning/%d/review", planID)
	submit := func(reviewID int64, score string) *httptest.ResponseRecorder {
		return learningPost(r, reviewURL, url.Values{"review_id": {fmt.Sprint(reviewID)}, "score": {score}})
	}

	for _, tt := range []struct {
		button      string
		score       int
		wantMastery int
	}{
		{button: "Recalled", score: 5, wantMastery: 60},
		{button: "With a hint", score: 3, wantMastery: 40},
		{button: "Couldn't recall", score: 2, wantMastery: 30},
		{button: "Wrong / guessed", score: 1, wantMastery: 20},
	} {
		s.Run(tt.button, func() {
			word := s.seedWord(ctx, planID, tt.button, "x")
			s.seedScoredReview(ctx, word, 5, now.Add(-2*time.Hour))
			s.seedScoredReview(ctx, word, 5, now.Add(-time.Hour))
			review := s.seedPendingReview(ctx, word, learningDay(0))

			w := submit(review, fmt.Sprint(tt.score))
			require.Equal(s.T(), http.StatusSeeOther, w.Code)
			assert.Equal(s.T(), reviewURL, w.Header().Get("Location"))

			reviews := s.wordReviews(ctx, word)
			require.Len(s.T(), reviews, 3, "earlier reviews kept")
			assert.Equal(s.T(), review, reviews[0].ID)
			assert.Equal(s.T(), tt.score, *reviews[0].Score)
			assert.WithinDuration(s.T(), now, *reviews[0].ReviewedAt, time.Minute)
			assert.Equal(s.T(), tt.wantMastery, s.learningWord(ctx, word).Mastery)
		})
	}

	word := s.seedWord(ctx, planID, "νερό", "water")
	due := s.seedPendingReview(ctx, word, learningDay(0))
	notDue := s.seedPendingReview(ctx, word, learningDay(1))
	scored := s.seedScoredReview(ctx, word, 4, now)

	for _, tt := range []struct {
		name     string
		reviewID int64
		score    string
	}{
		{name: "score 4", reviewID: due, score: "4"},
		{name: "score outside 1-5", reviewID: due, score: "9"},
		{name: "review with a score", reviewID: scored, score: "5"},
		{name: "review not due yet", reviewID: notDue, score: "5"},
	} {
		s.Run(tt.name+" re-renders with an error", func() {
			w := submit(tt.reviewID, tt.score)
			assert.Equal(s.T(), http.StatusBadRequest, w.Code)
			assert.Contains(s.T(), w.Body.String(), `role="alert"`)

			pending := 0
			for _, r := range s.wordReviews(ctx, word) {
				if r.Score == nil {
					pending++
				}
			}
			assert.Equal(s.T(), 2, pending, "nothing changes")
		})
	}

	otherPlan := s.seedLearningPlan(ctx, "Greek A2", domain.LearningPlanActive)
	outside := s.seedPendingReview(ctx, s.seedWord(ctx, otherPlan, "γάλα", "milk"), learningDay(0))

	otherCtx := s.otherUserContext(ctx)
	foreignPlan := s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive)
	foreignReview := s.seedPendingReview(otherCtx, s.seedWord(otherCtx, foreignPlan, "x", "y"), learningDay(0))

	s.Run("review of a word outside the plan 404s", func() {
		assert.Equal(s.T(), http.StatusNotFound, submit(outside, "5").Code)
	})
	s.Run("foreign review 404s", func() {
		assert.Equal(s.T(), http.StatusNotFound, submit(foreignReview, "5").Code)
	})
	s.Run("foreign plan 404s", func() {
		w := learningPost(r, fmt.Sprintf("/web/learning/%d/review", foreignPlan), url.Values{"review_id": {fmt.Sprint(foreignReview)}, "score": {"5"}})
		assert.Equal(s.T(), http.StatusNotFound, w.Code)
	})
}

func (s *IntegrationTestSuite) TestLearningWeb_VocabularyList() {
	ctx := s.Context()
	now := time.Now()
	r := s.learningRouter(ctx)

	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	vocabURL := fmt.Sprintf("/web/learning/%d/vocabulary", planID)

	s.Run("empty state", func() {
		html := learningGet(r, vocabURL).Body.String()
		assert.Contains(s.T(), html, "No words yet.")
	})

	older := s.seedWordAt(ctx, planID, "νερό", "water", now.Add(-2*time.Hour))
	for i := range 4 {
		s.seedScoredReview(ctx, older, 5, now.AddDate(0, 0, -51).Add(time.Duration(i-3)*time.Minute))
	}
	s.seedPendingReview(ctx, older, learningDay(2))
	newer := s.seedWordAt(ctx, planID, "ψωμί", "bread", now.Add(-time.Hour))

	otherPlan := s.seedLearningPlan(ctx, "Greek A2", domain.LearningPlanActive)
	s.seedWord(ctx, otherPlan, "γάλα", "milk")

	otherCtx := s.otherUserContext(ctx)
	foreign := s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive)

	s.Run("newest first with mastery, uncertainty, last score and next review", func() {
		w := learningGet(r, vocabURL)
		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()

		newerIdx := strings.Index(html, fmt.Sprintf(`<a href="%s/%d">ψωμί</a>`, vocabURL, newer))
		olderIdx := strings.Index(html, fmt.Sprintf(`<a href="%s/%d">νερό</a>`, vocabURL, older))
		require.Greater(s.T(), newerIdx, -1)
		require.Greater(s.T(), olderIdx, newerIdx)

		assert.Contains(s.T(), html, "80% (−20%)")
		assert.Contains(s.T(), html, ">50%</td>")
		assert.Contains(s.T(), html, ">5</td>")
		assert.Contains(s.T(), html, learningDate(2))
		assert.NotContains(s.T(), html, "γάλα")
	})

	s.Run("51 words split into two pages", func() {
		for i := range 49 {
			s.seedWordAt(ctx, planID, fmt.Sprintf("word %02d", i), "x", now.Add(time.Duration(i)*time.Minute))
		}
		first := learningGet(r, vocabURL).Body.String()
		assert.Contains(s.T(), first, "word 48")
		assert.NotContains(s.T(), first, "νερό")

		second := learningGet(r, vocabURL+"?page=2").Body.String()
		assert.Contains(s.T(), second, "νερό")
		assert.NotContains(s.T(), second, "word 48")
	})

	s.Run("foreign plan 404s", func() {
		assert.Equal(s.T(), http.StatusNotFound, learningGet(r, fmt.Sprintf("/web/learning/%d/vocabulary", foreign)).Code)
	})
}

func (s *IntegrationTestSuite) TestLearningWeb_VocabularySearch() {
	ctx := s.Context()
	now := time.Now()
	r := s.learningRouter(ctx)

	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	vocabURL := fmt.Sprintf("/web/learning/%d/vocabulary", planID)
	water := s.seedWordAt(ctx, planID, "νερό", "Water", now.Add(-3*time.Hour))
	_, _, err := learning.EditVocabulary(ctx, nil, learning.EditVocabularyInput{WordID: water, Notes: util.Ptr("a drink")})
	require.NoError(s.T(), err)
	milk := s.seedWordAt(ctx, planID, "γάλα", "milk", now.Add(-time.Hour))
	_, _, err = learning.EditVocabulary(ctx, nil, learning.EditVocabularyInput{WordID: milk, Notes: util.Ptr("white DRINK")})
	require.NoError(s.T(), err)
	s.seedWord(ctx, planID, "ψωμί", "bread")

	otherPlan := s.seedLearningPlan(ctx, "Greek A2", domain.LearningPlanActive)
	s.seedWord(ctx, otherPlan, "καταρράκτης", "waterfall")

	s.Run("two phrases ranked by match count", func() {
		w := learningGet(r, vocabURL+"?q="+url.QueryEscape("water, drink"))
		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()

		waterIdx := strings.Index(html, ">νερό</a>")
		milkIdx := strings.Index(html, ">γάλα</a>")
		require.Greater(s.T(), waterIdx, -1)
		require.Greater(s.T(), milkIdx, waterIdx)
		assert.NotContains(s.T(), html, "ψωμί")
		assert.NotContains(s.T(), html, "waterfall")
	})

	s.Run("more than 5 phrases shows an error", func() {
		w := learningGet(r, vocabURL+"?q="+url.QueryEscape("a,b,c,d,e,f"))
		assert.Equal(s.T(), http.StatusBadRequest, w.Code)
		assert.Contains(s.T(), w.Body.String(), "maximum 5 comma-separated phrases")
	})
}

func (s *IntegrationTestSuite) TestLearningWeb_VocabularyAdd() {
	ctx := s.Context()
	r := s.learningRouter(ctx)

	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	vocabURL := fmt.Sprintf("/web/learning/%d/vocabulary", planID)
	skill := s.seedSkill(ctx, planID, "food")
	otherPlanSkill := s.seedSkill(ctx, s.seedLearningPlan(ctx, "Greek A2", domain.LearningPlanActive), "food")

	otherCtx := s.otherUserContext(ctx)
	foreign := s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive)

	s.Run("word saved with 0% mastery and no reviews", func() {
		w := learningPost(r, vocabURL, url.Values{"term": {"νερό"}, "translation": {"water"}, "notes": {"neuter"}, "skill_id": {""}})
		require.Equal(s.T(), http.StatusSeeOther, w.Code)
		assert.Equal(s.T(), vocabURL, w.Header().Get("Location"))

		words, err := s.Repo().SearchVocabulary(ctx, domain.VocabularySearchFilter{UserID: gateways.UserIDFromContext(ctx), Query: "νερό", LearningPlanID: &planID})
		require.NoError(s.T(), err)
		require.Len(s.T(), words, 1)
		w0 := s.learningWord(ctx, words[0].ID)
		assert.Equal(s.T(), "neuter", *w0.Notes)
		assert.Nil(s.T(), w0.SkillID)
		assert.Equal(s.T(), 0, w0.Mastery)
		assert.Empty(s.T(), s.wordReviews(ctx, w0.ID))
	})

	s.Run("with a skill", func() {
		w := learningPost(r, vocabURL, url.Values{"term": {"ψωμί"}, "translation": {"bread"}, "skill_id": {fmt.Sprint(skill)}})
		require.Equal(s.T(), http.StatusSeeOther, w.Code)

		words, err := s.Repo().SearchVocabulary(ctx, domain.VocabularySearchFilter{UserID: gateways.UserIDFromContext(ctx), Query: "ψωμί"})
		require.NoError(s.T(), err)
		require.Len(s.T(), words, 1)
		assert.Equal(s.T(), skill, *words[0].SkillID)
	})

	for _, tt := range []struct {
		name    string
		form    url.Values
		wantErr string
	}{
		{name: "empty term", form: url.Values{"term": {" "}, "translation": {"typed translation"}}, wantErr: "term is required"},
		{name: "empty translation", form: url.Values{"term": {"typed term"}, "translation": {""}}, wantErr: "translation is required"},
		{name: "duplicate term", form: url.Values{"term": {"ΝΕΡΌ"}, "translation": {"typed translation"}}, wantErr: "already in the plan"},
		{name: "skill of another plan", form: url.Values{"term": {"typed term"}, "translation": {"typed translation"}, "skill_id": {fmt.Sprint(otherPlanSkill)}}, wantErr: "not found in this plan"},
	} {
		s.Run(tt.name+" re-renders with an error", func() {
			before := s.countPlanWords(ctx, planID)
			w := learningPost(r, vocabURL, tt.form)
			assert.Equal(s.T(), http.StatusBadRequest, w.Code)
			html := w.Body.String()
			assert.Contains(s.T(), html, tt.wantErr)
			assert.Contains(s.T(), html, "typed ", "typed values kept")
			assert.Equal(s.T(), before, s.countPlanWords(ctx, planID))
		})
	}

	s.Run("foreign plan 404s", func() {
		w := learningPost(r, fmt.Sprintf("/web/learning/%d/vocabulary", foreign), url.Values{"term": {"x"}, "translation": {"y"}})
		assert.Equal(s.T(), http.StatusNotFound, w.Code)
	})
}

func (s *IntegrationTestSuite) TestLearningWeb_VocabularyEdit() {
	ctx := s.Context()
	now := time.Now()
	r := s.learningRouter(ctx)

	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	vocabURL := fmt.Sprintf("/web/learning/%d/vocabulary", planID)
	skill := s.seedSkill(ctx, planID, "food")
	word := s.seedWord(ctx, planID, "νερο", "water")
	_, _, err := learning.EditVocabulary(ctx, nil, learning.EditVocabularyInput{WordID: word, Notes: util.Ptr("neuter"), SkillID: &skill})
	require.NoError(s.T(), err)
	s.seedScoredReview(ctx, word, 5, now.AddDate(0, 0, -3))
	s.seedScoredReview(ctx, word, 2, now.AddDate(0, 0, -1))
	s.seedWord(ctx, planID, "ψωμί", "bread")
	wordURL := fmt.Sprintf("%s/%d", vocabURL, word)

	outside := s.seedWord(ctx, s.seedLearningPlan(ctx, "Greek A2", domain.LearningPlanActive), "γάλα", "milk")

	otherCtx := s.otherUserContext(ctx)
	foreignPlan := s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive)
	foreignWord := s.seedWord(otherCtx, foreignPlan, "x", "y")

	s.Run("page shows values, mastery and history newest first", func() {
		w := learningGet(r, wordURL)
		require.Equal(s.T(), http.StatusOK, w.Code)
		html := w.Body.String()

		assert.Contains(s.T(), html, `value="νερο"`)
		assert.Contains(s.T(), html, `value="water"`)
		assert.Contains(s.T(), html, "neuter</textarea>")
		assert.Contains(s.T(), html, fmt.Sprintf(`<option value="%d" selected>food</option>`, skill))
		assert.Contains(s.T(), html, "Mastery 10%")
		assert.Contains(s.T(), html, fmt.Sprintf(`action="%s/delete"`, wordURL))
		assert.Less(s.T(), strings.Index(html, now.AddDate(0, 0, -1).In(nicosia).Format("2006-01-02")), strings.Index(html, now.AddDate(0, 0, -3).In(nicosia).Format("2006-01-02")))
	})

	s.Run("saving changes fields and keeps reviews", func() {
		w := learningPost(r, wordURL, url.Values{"term": {"νερό"}, "translation": {"water (n.)"}, "notes": {"το νερό"}, "skill_id": {fmt.Sprint(skill)}})
		require.Equal(s.T(), http.StatusSeeOther, w.Code)
		assert.Equal(s.T(), vocabURL, w.Header().Get("Location"))

		stored := s.learningWord(ctx, word)
		assert.Equal(s.T(), "νερό", stored.Term)
		assert.Equal(s.T(), "water (n.)", stored.Translation)
		assert.Equal(s.T(), "το νερό", *stored.Notes)
		assert.Len(s.T(), s.wordReviews(ctx, word), 2)
	})

	s.Run("empty notes and No skill clear those fields", func() {
		w := learningPost(r, wordURL, url.Values{"term": {"νερό"}, "translation": {"water"}, "notes": {""}, "skill_id": {""}})
		require.Equal(s.T(), http.StatusSeeOther, w.Code)

		stored := s.learningWord(ctx, word)
		assert.Nil(s.T(), stored.Notes)
		assert.Nil(s.T(), stored.SkillID)
	})

	for _, tt := range []struct {
		name    string
		form    url.Values
		wantErr string
	}{
		{name: "empty term", form: url.Values{"term": {""}, "translation": {"water"}}, wantErr: "term cannot be empty"},
		{name: "duplicate term", form: url.Values{"term": {"ψωμί"}, "translation": {"water"}}, wantErr: "already in the plan"},
	} {
		s.Run(tt.name+" re-renders with an error", func() {
			w := learningPost(r, wordURL, tt.form)
			assert.Equal(s.T(), http.StatusBadRequest, w.Code)
			assert.Contains(s.T(), w.Body.String(), tt.wantErr)
			assert.Equal(s.T(), "νερό", s.learningWord(ctx, word).Term)
		})
	}

	for name, target := range map[string]string{
		"word outside the plan": fmt.Sprintf("%s/%d", vocabURL, outside),
		"foreign word":          fmt.Sprintf("%s/%d", vocabURL, foreignWord),
		"foreign plan":          fmt.Sprintf("/web/learning/%d/vocabulary/%d", foreignPlan, foreignWord),
	} {
		s.Run(name+" 404s", func() {
			assert.Equal(s.T(), http.StatusNotFound, learningGet(r, target).Code)
			assert.Equal(s.T(), http.StatusNotFound, learningPost(r, target, url.Values{"term": {"z"}, "translation": {"z"}}).Code)
		})
	}
}

func (s *IntegrationTestSuite) TestLearningWeb_VocabularyDelete() {
	ctx := s.Context()
	r := s.learningRouter(ctx)

	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	vocabURL := fmt.Sprintf("/web/learning/%d/vocabulary", planID)
	word := s.seedWord(ctx, planID, "νερό", "water")
	s.seedScoredReview(ctx, word, 5, time.Now())
	s.seedPendingReview(ctx, word, learningDay(1))

	otherPlan := s.seedLearningPlan(ctx, "Greek A2", domain.LearningPlanActive)
	outside := s.seedWord(ctx, otherPlan, "γάλα", "milk")

	otherCtx := s.otherUserContext(ctx)
	foreignWord := s.seedWord(otherCtx, s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive), "x", "y")

	s.Run("word and its reviews deleted", func() {
		w := learningPost(r, fmt.Sprintf("%s/%d/delete", vocabURL, word), nil)
		require.Equal(s.T(), http.StatusSeeOther, w.Code)
		assert.Equal(s.T(), vocabURL, w.Header().Get("Location"))
		assert.Equal(s.T(), 0, s.countPlanWords(ctx, planID))
		assert.Empty(s.T(), s.wordReviews(ctx, word))
	})

	s.Run("word outside the plan 404s, deleting nothing", func() {
		w := learningPost(r, fmt.Sprintf("%s/%d/delete", vocabURL, outside), nil)
		assert.Equal(s.T(), http.StatusNotFound, w.Code)
		assert.Equal(s.T(), 1, s.countPlanWords(ctx, otherPlan))
	})

	s.Run("foreign word 404s, deleting nothing", func() {
		w := learningPost(r, fmt.Sprintf("%s/%d/delete", vocabURL, foreignWord), nil)
		assert.Equal(s.T(), http.StatusNotFound, w.Code)
		words, err := s.Repo().ListVocabulary(otherCtx, domain.VocabularyFilter{UserID: gateways.UserIDFromContext(otherCtx), IDs: []int64{foreignWord}})
		require.NoError(s.T(), err)
		assert.Len(s.T(), words, 1)
	})
}
