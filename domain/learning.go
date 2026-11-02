package domain

import (
	"math"
	"time"
)

// LearningPlanStatus is the lifecycle state of a learning plan (see
// docs/functions/learning-spec.md).
type LearningPlanStatus string

const (
	LearningPlanActive    LearningPlanStatus = "active"
	LearningPlanPaused    LearningPlanStatus = "paused"
	LearningPlanCompleted LearningPlanStatus = "completed"
)

// LearningPlanStatuses are every plan status.
var LearningPlanStatuses = []LearningPlanStatus{LearningPlanActive, LearningPlanPaused, LearningPlanCompleted}

// LearningExerciseState is computed from an exercise's fields
type LearningExerciseState string

const (
	LearningExercisePlanned  LearningExerciseState = "planned"
	LearningExerciseAnswered LearningExerciseState = "answered"
	LearningExerciseReviewed LearningExerciseState = "reviewed"
)

// MasterySteps is the mastery change of each score 1-5
var MasterySteps = map[int]int{5: 20, 4: 10, 3: 0, 2: -10, 1: -20}

// FlashcardScores are the scores the web flashcard buttons post
var FlashcardScores = []int{5, 3, 2, 1}

const (
	// MasteredThreshold is the effective mastery from which an item counts as mastered.
	MasteredThreshold = 80
	// stabilityBaseDays and stabilityPerMastery give S = 3 + 0.6 × mastery days.
	stabilityBaseDays   = 3.0
	stabilityPerMastery = 0.6
	// debuffFactor is the share of mastery lost at 100% uncertainty.
	debuffFactor = 0.5
)

type LearningPlan struct {
	ID          int64              `json:"id" db:"id"`
	UserID      int64              `json:"-" db:"user_id"`
	Name        string             `json:"name" db:"name"`
	Description *string            `json:"description,omitempty" db:"description" jsonschema:"Goal of the plan"`
	Status      LearningPlanStatus `json:"status" db:"status" jsonschema:"active, paused or completed"`
	CreatedAt   time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" db:"updated_at"`
}

type Skill struct {
	ID             int64     `json:"id" db:"id"`
	UserID         int64     `json:"-" db:"user_id"`
	LearningPlanID int64     `json:"learning_plan_id" db:"learning_plan_id"`
	Name           string    `json:"name" db:"name"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
	// Read-only, computed by ListSkills from learning_exercises
	AnsweredCount   int        `json:"answered_count" db:"answered_count"`
	PlannedCount    int        `json:"planned_count" db:"planned_count"`
	LastPracticedAt *time.Time `json:"last_practiced_at,omitempty" db:"last_practiced_at"`
	LastScore       *int       `json:"last_score,omitempty" db:"last_score"`
	Scores          []int      `json:"-" db:"scores"` // reviewed scores, oldest first
	// Computed in the action layer by ComputeMastery(Scores, LastPracticedAt, now)
	MasteryState
}

type LearningExercise struct {
	ID         int64      `json:"id" db:"id"`
	UserID     int64      `json:"-" db:"user_id"`
	SkillID    int64      `json:"skill_id" db:"skill_id"`
	Kind       *string    `json:"kind,omitempty" db:"kind" jsonschema:"Free-text kind of exercise, e.g. translate RU→EL"`
	Content    string     `json:"content" db:"content" jsonschema:"Full exercise text"`
	PlannedFor time.Time  `json:"planned_for" db:"planned_for"`
	Answer     *string    `json:"answer,omitempty" db:"answer" jsonschema:"The user's answer as written, recognized word for word from handwriting when sent as a photo, not corrected"`
	AnsweredAt *time.Time `json:"answered_at,omitempty" db:"answered_at"`
	Score      *int       `json:"score,omitempty" db:"score" jsonschema:"How it went, 1-5"`
	Note       *string    `json:"note,omitempty" db:"note" jsonschema:"Mistakes and corrections"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty" db:"reviewed_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	// Read-only, joined from skills
	SkillName string `json:"skill_name" db:"skill_name"`
}

// State is computed from the answer and score.
func (e LearningExercise) State() LearningExerciseState {
	switch {
	case e.Score != nil:
		return LearningExerciseReviewed
	case e.Answer != nil:
		return LearningExerciseAnswered
	default:
		return LearningExercisePlanned
	}
}

type VocabularyWord struct {
	ID             int64     `json:"id" db:"id"`
	UserID         int64     `json:"-" db:"user_id"`
	LearningPlanID int64     `json:"learning_plan_id" db:"learning_plan_id"`
	SkillID        *int64    `json:"skill_id,omitempty" db:"skill_id"`
	Term           string    `json:"term" db:"term"`
	Translation    string    `json:"translation" db:"translation"`
	Notes          *string   `json:"notes,omitempty" db:"notes"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
	// Read-only, computed from vocabulary_reviews
	ReviewCount    int        `json:"review_count" db:"review_count" jsonschema:"Scored reviews"`
	LastReviewedAt *time.Time `json:"last_reviewed_at,omitempty" db:"last_reviewed_at"`
	LastScore      *int       `json:"last_score,omitempty" db:"last_score"`
	NextPlannedFor *time.Time `json:"next_planned_for,omitempty" db:"next_planned_for" jsonschema:"Earliest pending review"`
	Scores         []int      `json:"-" db:"scores"` // review scores, oldest by reviewed_at first
	// Computed in the action layer by ComputeMastery(Scores, LastReviewedAt, now)
	MasteryState
}

type VocabularyReview struct {
	ID           int64      `json:"id" db:"id"`
	UserID       int64      `json:"-" db:"user_id"`
	VocabularyID int64      `json:"vocabulary_id" db:"vocabulary_id"`
	PlannedFor   time.Time  `json:"planned_for" db:"planned_for"`
	Score        *int       `json:"score,omitempty" db:"score" jsonschema:"1-5; empty while pending"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty" db:"reviewed_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	// Read-only, joined from vocabulary
	Term        string  `json:"term" db:"term"`
	Translation string  `json:"translation" db:"translation"`
	Notes       *string `json:"notes,omitempty" db:"notes"`
}

// MasteryState is computed at read time, never stored; shared by skills and words
type MasteryState struct {
	Mastery          int `json:"mastery" db:"-" jsonschema:"Mastery percentage from the score history, 0-100"`
	Uncertainty      int `json:"uncertainty" db:"-" jsonschema:"Percentage, 0-100, growing with days since the last practice"`
	Debuff           int `json:"debuff" db:"-" jsonschema:"Points taken off mastery because of uncertainty"`
	EffectiveMastery int `json:"effective_mastery" db:"-" jsonschema:"mastery - debuff; 80+ is confidently mastered"`
}

// ComputeMastery folds scores, oldest first, with MasterySteps into mastery, then applies
// uncertainty from the days between lastPracticedAt and now; nil lastPracticedAt = 100% uncertainty
func ComputeMastery(scores []int, lastPracticedAt *time.Time, now time.Time) MasteryState {
	mastery := 0
	for _, score := range scores {
		mastery = min(max(mastery+MasterySteps[score], 0), 100)
	}

	uncertainty := 1.0
	if lastPracticedAt != nil {
		days := max(now.Sub(*lastPracticedAt).Hours()/24, 0)
		stability := stabilityBaseDays + stabilityPerMastery*float64(mastery)
		uncertainty = days / (days + stability)
	}

	debuff := 0
	if lastPracticedAt != nil {
		debuff = int(math.Round(float64(mastery) * uncertainty * debuffFactor))
	}

	return MasteryState{
		Mastery:          mastery,
		Uncertainty:      int(math.Round(uncertainty * 100)),
		Debuff:           debuff,
		EffectiveMastery: mastery - debuff,
	}
}

// VocabularyStats is a per-plan summary, computed by the action from the plan's words
type VocabularyStats struct {
	LearningPlanID int64
	Total          int
	Due            int // pending reviews whose day has come
	Unscheduled    int // words with no pending review
	Mastered       int // words with effective mastery >= 80
	AverageMastery int // average effective mastery
}

// LearningExerciseStats is a per-plan summary, computed at read time
type LearningExerciseStats struct {
	LearningPlanID int64
	Planned        int // all without an answer
	Due            int // planned, day has come
	AwaitingReview int // answered, no score yet
}

type LearningPlanFilter struct {
	UserID   int64
	Statuses []LearningPlanStatus // empty = active and paused
}

type LearningExerciseFilter struct {
	UserID         int64
	LearningPlanID *int64
	SkillID        *int64
	States         []LearningExerciseState // empty = every state
	DueAt          *time.Time              // planned_for <= DueAt
	OldestFirst    bool                    // planned_for ASC, id ASC; default planned_for DESC, id DESC
	Limit          int
	Offset         int
}

type VocabularyFilter struct {
	UserID             int64
	LearningPlanID     *int64
	IDs                []int64
	Unscheduled        bool // no pending review
	LeastRecentlyFirst bool // last reviewed ASC NULLS FIRST, then id ASC; default created_at DESC, then id DESC
	Limit              int  // 0 = no limit
	Offset             int
}

// VocabularySearchFilter is a single-variant search; the action merges variants and counts match_count
type VocabularySearchFilter struct {
	UserID         int64
	Query          string // ILIKE substring on term, translation and notes
	LearningPlanID *int64
}

type VocabularyReviewFilter struct {
	UserID         int64
	LearningPlanID *int64
	VocabularyID   *int64
	IDs            []int64
	Pending        *bool      // true = no score yet, false = scored
	DueAt          *time.Time // planned_for <= DueAt; ordered by planned_for ASC, then id ASC
	ReviewedFrom   *time.Time // reviewed_at >= ReviewedFrom
	ReviewedTo     *time.Time // reviewed_at < ReviewedTo
	Limit          int        // 0 = no limit; default order reviewed_at DESC NULLS LAST, planned_for ASC
	Offset         int
}

// LearningExerciseReview is the agent's result for one answered exercise
type LearningExerciseReview struct {
	ExerciseID int64
	Score      int
	Note       *string
	ReviewedAt time.Time
}
