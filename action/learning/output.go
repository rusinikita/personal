package learning

import (
	"time"

	"personal/domain"
)

const dayLayout = "2006-01-02"

// PlanResult is one learning plan as returned by the MCP tools.
type PlanResult struct {
	ID          int64  `json:"id" jsonschema:"Learning plan ID"`
	Name        string `json:"name" jsonschema:"Plan name, e.g. Greek A1"`
	Description string `json:"description,omitempty" jsonschema:"Goal of the plan"`
	Status      string `json:"status" jsonschema:"active, paused or completed"`
	CreatedAt   string `json:"created_at" jsonschema:"ISO8601"`
	UpdatedAt   string `json:"updated_at" jsonschema:"ISO8601"`
}

// ExerciseStatsResult summarizes a plan's exercises.
type ExerciseStatsResult struct {
	Planned        int `json:"planned" jsonschema:"Exercises without an answer"`
	Due            int `json:"due" jsonschema:"Planned exercises whose day has come"`
	AwaitingReview int `json:"awaiting_review" jsonschema:"Answered exercises without a score"`
}

// VocabularyStatsResult summarizes a plan's words.
type VocabularyStatsResult struct {
	Total          int `json:"total" jsonschema:"Words in the plan"`
	Due            int `json:"due" jsonschema:"Pending reviews whose day has come"`
	Unscheduled    int `json:"unscheduled" jsonschema:"Words with no pending review"`
	Mastered       int `json:"mastered" jsonschema:"Words with effective mastery of 80 or more"`
	AverageMastery int `json:"average_mastery" jsonschema:"Average effective mastery"`
}

// SkillResult is one skill with its computed mastery and exercise counts.
type SkillResult struct {
	ID              int64  `json:"id" jsonschema:"Skill ID"`
	LearningPlanID  int64  `json:"learning_plan_id" jsonschema:"Plan the skill belongs to"`
	Name            string `json:"name" jsonschema:"Skill name"`
	AnsweredCount   int    `json:"answered_count" jsonschema:"Exercises with an answer"`
	PlannedCount    int    `json:"planned_count" jsonschema:"Exercises without an answer"`
	LastPracticedAt string `json:"last_practiced_at,omitempty" jsonschema:"Latest answer (ISO8601)"`
	LastScore       *int   `json:"last_score,omitempty" jsonschema:"Score of the latest reviewed exercise"`
	domain.MasteryState
}

// ExerciseResult is one learning exercise.
type ExerciseResult struct {
	ID         int64  `json:"id" jsonschema:"Exercise ID"`
	SkillID    int64  `json:"skill_id" jsonschema:"Skill ID"`
	SkillName  string `json:"skill_name" jsonschema:"Skill name"`
	Kind       string `json:"kind,omitempty" jsonschema:"Free-text kind, e.g. translate RU→EL"`
	Content    string `json:"content" jsonschema:"Full exercise text"`
	PlannedFor string `json:"planned_for" jsonschema:"Planned day, YYYY-MM-DD"`
	State      string `json:"state" jsonschema:"planned, answered or reviewed"`
	Answer     string `json:"answer,omitempty" jsonschema:"The user's answer as written"`
	AnsweredAt string `json:"answered_at,omitempty" jsonschema:"ISO8601"`
	Score      *int   `json:"score,omitempty" jsonschema:"1-5"`
	Note       string `json:"note,omitempty" jsonschema:"Mistakes and corrections"`
	ReviewedAt string `json:"reviewed_at,omitempty" jsonschema:"ISO8601"`
}

// WordResult is one vocabulary word with its computed mastery and review fields.
type WordResult struct {
	ID             int64  `json:"id" jsonschema:"Word ID"`
	LearningPlanID int64  `json:"learning_plan_id" jsonschema:"Plan ID"`
	SkillID        *int64 `json:"skill_id,omitempty" jsonschema:"Optional topic skill"`
	Term           string `json:"term" jsonschema:"Term"`
	Translation    string `json:"translation" jsonschema:"Translation"`
	Notes          string `json:"notes,omitempty" jsonschema:"Notes"`
	ReviewCount    int    `json:"review_count" jsonschema:"Scored reviews"`
	LastReviewedAt string `json:"last_reviewed_at,omitempty" jsonschema:"Latest scored review (ISO8601)"`
	LastScore      *int   `json:"last_score,omitempty" jsonschema:"Score of the latest review"`
	NextPlannedFor string `json:"next_planned_for,omitempty" jsonschema:"Earliest pending review day, YYYY-MM-DD"`
	domain.MasteryState
}

// ReviewResult is one vocabulary review.
type ReviewResult struct {
	ID           int64  `json:"id" jsonschema:"Review ID"`
	VocabularyID int64  `json:"vocabulary_id" jsonschema:"Word ID"`
	Term         string `json:"term" jsonschema:"Term"`
	Translation  string `json:"translation" jsonschema:"Translation"`
	PlannedFor   string `json:"planned_for" jsonschema:"Planned day, YYYY-MM-DD"`
	Score        *int   `json:"score,omitempty" jsonschema:"1-5; empty while pending"`
	ReviewedAt   string `json:"reviewed_at,omitempty" jsonschema:"ISO8601; empty while pending"`
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

func formatDay(t time.Time) string {
	return t.In(location()).Format(dayLayout)
}

func planToResult(p domain.LearningPlan) PlanResult {
	return PlanResult{
		ID:          p.ID,
		Name:        p.Name,
		Description: deref(p.Description),
		Status:      string(p.Status),
		CreatedAt:   p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   p.UpdatedAt.Format(time.RFC3339),
	}
}

func skillToResult(s domain.Skill) SkillResult {
	return SkillResult{
		ID:              s.ID,
		LearningPlanID:  s.LearningPlanID,
		Name:            s.Name,
		AnsweredCount:   s.AnsweredCount,
		PlannedCount:    s.PlannedCount,
		LastPracticedAt: formatTime(s.LastPracticedAt),
		LastScore:       s.LastScore,
		MasteryState:    s.MasteryState,
	}
}

func exerciseToResult(e domain.LearningExercise) ExerciseResult {
	return ExerciseResult{
		ID:         e.ID,
		SkillID:    e.SkillID,
		SkillName:  e.SkillName,
		Kind:       deref(e.Kind),
		Content:    e.Content,
		PlannedFor: formatDay(e.PlannedFor),
		State:      string(e.State()),
		Answer:     deref(e.Answer),
		AnsweredAt: formatTime(e.AnsweredAt),
		Score:      e.Score,
		Note:       deref(e.Note),
		ReviewedAt: formatTime(e.ReviewedAt),
	}
}

func exercisesToResults(exercises []domain.LearningExercise) []ExerciseResult {
	results := make([]ExerciseResult, len(exercises))
	for i, e := range exercises {
		results[i] = exerciseToResult(e)
	}
	return results
}

func wordToResult(w domain.VocabularyWord) WordResult {
	r := WordResult{
		ID:             w.ID,
		LearningPlanID: w.LearningPlanID,
		SkillID:        w.SkillID,
		Term:           w.Term,
		Translation:    w.Translation,
		Notes:          deref(w.Notes),
		ReviewCount:    w.ReviewCount,
		LastReviewedAt: formatTime(w.LastReviewedAt),
		LastScore:      w.LastScore,
		MasteryState:   w.MasteryState,
	}
	if w.NextPlannedFor != nil {
		r.NextPlannedFor = formatDay(*w.NextPlannedFor)
	}
	return r
}

func wordsToResults(words []domain.VocabularyWord) []WordResult {
	results := make([]WordResult, len(words))
	for i, w := range words {
		results[i] = wordToResult(w)
	}
	return results
}

func reviewToResult(r domain.VocabularyReview) ReviewResult {
	return ReviewResult{
		ID:           r.ID,
		VocabularyID: r.VocabularyID,
		Term:         r.Term,
		Translation:  r.Translation,
		PlannedFor:   formatDay(r.PlannedFor),
		Score:        r.Score,
		ReviewedAt:   formatTime(r.ReviewedAt),
	}
}

func reviewsToResults(reviews []domain.VocabularyReview) []ReviewResult {
	results := make([]ReviewResult, len(reviews))
	for i, r := range reviews {
		results[i] = reviewToResult(r)
	}
	return results
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
