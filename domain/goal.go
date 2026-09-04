package domain

import "time"

// GoalType is the kind of goal being tracked — seven flat, sibling values,
// each with its own required fields and progress calculation (see
// docs/functions/goals-spec.md Best Practices). There is no shared
// "money"/"exercise"/"activity" grouping — a goal type is never nested
// inside another via a second discriminator field.
type GoalType string

const (
	GoalTypeMoneySaving             GoalType = "money_saving"              // reach TargetValue in accumulated balance
	GoalTypeMoneySpend              GoalType = "money_spend"               // don't exceed TargetValue spent on Category
	GoalTypeExerciseMaxWeight       GoalType = "exercise_max_weight"       // reach TargetValue kg all-time max for ExerciseID
	GoalTypeExerciseTotalVolume     GoalType = "exercise_total_volume"     // reach TargetValue kg total volume for ExerciseID since StartsAt
	GoalTypeActivityOccurrenceCount GoalType = "activity_occurrence_count" // reach TargetValue occurrences of ActivityID since StartsAt
	GoalTypeActivityStreakCount     GoalType = "activity_streak_count"     // reach TargetValue consecutive check-ins of ActivityID, evaluated as of now
	GoalTypeManual                  GoalType = "manual"                    // reach TargetValue; CurrentValue incremented directly via log_goal_progress
)

// AllGoalTypes is every known GoalType, in the order create_goal validates
// against — used to check an input string is one of the seven known values.
var AllGoalTypes = []GoalType{
	GoalTypeMoneySaving, GoalTypeMoneySpend,
	GoalTypeExerciseMaxWeight, GoalTypeExerciseTotalVolume,
	GoalTypeActivityOccurrenceCount, GoalTypeActivityStreakCount,
	GoalTypeManual,
}

// IsValid reports whether t is one of the seven known GoalType values.
func (t GoalType) IsValid() bool {
	for _, known := range AllGoalTypes {
		if t == known {
			return true
		}
	}
	return false
}

// Goal represents any of the seven goal types — see goals-spec.md Best
// Practices for which fields apply to which GoalType. Category/Unit/
// BaselineBalanceEUR are stored packed into the goals.details JSONB column,
// not their own columns — the repository mapper marshals/unmarshals them, so
// this struct's shape is unaffected by that storage choice. CurrentValue is
// a real, always-populated column (not packed, not nullable) — cached, not
// computed live: create_goal/refresh_goals write it for six types,
// log_goal_progress writes it for manual, and every read (get_goal_progress,
// ListGoals, BuildGoalTiles) just returns whatever was last written.
type Goal struct {
	ID                 int64      `json:"id" db:"id"`
	UserID             int64      `json:"user_id" db:"user_id"`
	Name               string     `json:"name" db:"name"`
	GoalType           GoalType   `json:"goal_type" db:"goal_type"`
	ExerciseID         *int64     `json:"exercise_id,omitempty" db:"exercise_id"` // exercise_max_weight|exercise_total_volume only
	ActivityID         *int64     `json:"activity_id,omitempty" db:"activity_id"` // activity_occurrence_count|activity_streak_count only
	TargetValue        float64    `json:"target_value" db:"target_value"`
	CurrentValue       float64    `json:"current_value" db:"current_value"` // cached — not omitempty, every goal_type populates it
	Category           *string    `json:"category,omitempty"`               // money_spend only — packed into details
	Unit               *string    `json:"unit,omitempty"`                   // exercise_max_weight|exercise_total_volume|activity_occurrence_count|activity_streak_count|manual only — packed into details
	BaselineBalanceEUR *float64   `json:"baseline_balance_eur,omitempty"`   // money_saving only — packed into details
	StartsAt           time.Time  `json:"starts_at" db:"starts_at"`
	EndsAt             *time.Time `json:"ends_at,omitempty" db:"ends_at"` // nil means no deadline
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"` // bumped by create_goal, update_goal, log_goal_progress, refresh_goals
}

// RemainingValue is a plain getter, not a stored/duplicated field — every
// caller reads it straight off a Goal it already has, no repository
// round-trip, no separate GoalProgress type (see goals-spec.md Best
// Practices).
func (g Goal) RemainingValue() float64 {
	return g.TargetValue - g.CurrentValue // negative means over/exceeded
}

// GoalUpdate is a partial edit of an existing goal — every field but ID is
// optional; nil/false means "leave unchanged" (same convention as
// money-spec.md's TransactionUpdate). It's the payload for DB.UpdateGoal,
// the repository's only write method besides CreateGoal — three different
// callers build one: the update_goal MCP tool (user-facing correction),
// refresh_goals (CurrentValue only, for the six derived types' recomputed
// snapshot), and log_goal_progress (CurrentValue only, set to current+delta
// computed in Go). Structural/one-time fields — GoalType, ExerciseID,
// ActivityID, BaselineBalanceEUR — aren't on this struct at all; they're
// never editable (see goals-spec.md Best Practices).
type GoalUpdate struct {
	ID           int64
	Name         *string
	TargetValue  *float64
	EndsAt       *time.Time // set a new deadline
	ClearEndsAt  bool       // explicitly remove the deadline; ignored if EndsAt is also set
	Category     *string    // money_spend only — DB CHECK rejects it otherwise
	Unit         *string    // applicable goal types only — DB CHECK rejects it otherwise
	CurrentValue *float64   // which goal_type may set it is a rule each caller enforces itself
}

// GoalFilter defines query parameters for listing goals — the one read path
// for more than one goal at a time (see goals-spec.md Best Practices: there
// is no separate GetGoalProgress, ListGoals covers it with these two extra
// fields).
type GoalFilter struct {
	UserID     int64
	ActiveOnly bool       // starts_at <= At AND (ends_at IS NULL OR ends_at >= At)
	At         time.Time  // moment ActiveOnly's window is evaluated against; zero value means now
	Types      []GoalType // nil/empty = every type; non-empty scopes to a subset (e.g. Money's embedded tile grid)
}
