package domain

import "time"

// AchievementType is the kind of achievement being tracked — seven flat, sibling values,
// each with its own required fields and progress calculation (see
// docs/functions/achievements-spec.md Best Practices). There is no shared
// "money"/"exercise"/"activity" grouping — an achievement type is never nested
// inside another via a second discriminator field.
type AchievementType string

const (
	AchievementTypeMoneySaving             AchievementType = "money_saving"              // reach TargetValue in accumulated balance
	AchievementTypeMoneySpend              AchievementType = "money_spend"               // don't exceed TargetValue spent on Category
	AchievementTypeExerciseMaxWeight       AchievementType = "exercise_max_weight"       // reach TargetValue kg all-time max for ExerciseID
	AchievementTypeExerciseTotalVolume     AchievementType = "exercise_total_volume"     // reach TargetValue kg total volume for ExerciseID since StartsAt
	AchievementTypeActivityOccurrenceCount AchievementType = "activity_occurrence_count" // reach TargetValue occurrences of ActivityID since StartsAt
	AchievementTypeActivityStreakCount     AchievementType = "activity_streak_count"     // reach TargetValue consecutive check-ins of ActivityID, evaluated as of now
	AchievementTypeManual                  AchievementType = "manual"                    // reach TargetValue; CurrentValue incremented directly via log_achievement_progress
)

// AllAchievementTypes is every known AchievementType, in the order create_achievement validates
// against — used to check an input string is one of the seven known values.
var AllAchievementTypes = []AchievementType{
	AchievementTypeMoneySaving, AchievementTypeMoneySpend,
	AchievementTypeExerciseMaxWeight, AchievementTypeExerciseTotalVolume,
	AchievementTypeActivityOccurrenceCount, AchievementTypeActivityStreakCount,
	AchievementTypeManual,
}

// IsValid reports whether t is one of the seven known AchievementType values.
func (t AchievementType) IsValid() bool {
	for _, known := range AllAchievementTypes {
		if t == known {
			return true
		}
	}
	return false
}

// Achievement represents any of the seven achievement types — see achievements-spec.md Best
// Practices for which fields apply to which AchievementType. Category/Unit/
// BaselineBalanceEUR are stored packed into the achievements.details JSONB column,
// not their own columns — the repository mapper marshals/unmarshals them, so
// this struct's shape is unaffected by that storage choice. CurrentValue is
// a real, always-populated column (not packed, not nullable) — cached, not
// computed live: create_achievement/refresh_achievements write it for six types,
// log_achievement_progress writes it for manual, and every read (get_achievement_progress,
// ListAchievements, BuildAchievementTiles) just returns whatever was last written.
type Achievement struct {
	ID                 int64           `json:"id" db:"id"`
	UserID             int64           `json:"user_id" db:"user_id"`
	Name               string          `json:"name" db:"name"`
	AchievementType    AchievementType `json:"achievement_type" db:"achievement_type"`
	ExerciseID         *int64          `json:"exercise_id,omitempty" db:"exercise_id"` // exercise_max_weight|exercise_total_volume only
	ActivityID         *int64          `json:"activity_id,omitempty" db:"activity_id"` // activity_occurrence_count|activity_streak_count only
	TargetValue        float64         `json:"target_value" db:"target_value"`
	CurrentValue       float64         `json:"current_value" db:"current_value"` // cached — not omitempty, every achievement_type populates it
	Category           *string         `json:"category,omitempty"`               // money_spend only — packed into details
	Unit               *string         `json:"unit,omitempty"`                   // exercise_max_weight|exercise_total_volume|activity_occurrence_count|activity_streak_count|manual only — packed into details
	BaselineBalanceEUR *float64        `json:"baseline_balance_eur,omitempty"`   // money_saving only — packed into details
	StartsAt           time.Time       `json:"starts_at" db:"starts_at"`
	EndsAt             *time.Time      `json:"ends_at,omitempty" db:"ends_at"` // nil means no deadline
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at" db:"updated_at"` // bumped by create_achievement, update_achievement, log_achievement_progress, refresh_achievements
}

// RemainingValue is a plain getter, not a stored/duplicated field — every
// caller reads it straight off an Achievement it already has, no repository
// round-trip, no separate AchievementProgress type (see achievements-spec.md Best
// Practices).
func (g Achievement) RemainingValue() float64 {
	return g.TargetValue - g.CurrentValue // negative means over/exceeded
}

// AchievementUpdate is a partial edit of an existing achievement — every field but ID is
// optional; nil/false means "leave unchanged" (same convention as
// money-spec.md's TransactionUpdate). It's the payload for DB.UpdateAchievement,
// the repository's only write method besides CreateAchievement — three different
// callers build one: the update_achievement MCP tool (user-facing correction),
// refresh_achievements (CurrentValue only, for the six derived types' recomputed
// snapshot), and log_achievement_progress (CurrentValue only, set to current+delta
// computed in Go). Structural/one-time fields — AchievementType, ExerciseID,
// ActivityID, BaselineBalanceEUR — aren't on this struct at all; they're
// never editable (see achievements-spec.md Best Practices).
type AchievementUpdate struct {
	ID           int64
	Name         *string
	TargetValue  *float64
	EndsAt       *time.Time // set a new deadline
	ClearEndsAt  bool       // explicitly remove the deadline; ignored if EndsAt is also set
	Category     *string    // money_spend only — DB CHECK rejects it otherwise
	Unit         *string    // applicable achievement types only — DB CHECK rejects it otherwise
	CurrentValue *float64   // which achievement_type may set it is a rule each caller enforces itself
}

// AchievementFilter defines query parameters for listing achievements — the one read path
// for more than one achievement at a time (see achievements-spec.md Best Practices: there
// is no separate GetAchievementProgress, ListAchievements covers it with these two extra
// fields).
type AchievementFilter struct {
	UserID     int64
	ActiveOnly bool              // starts_at <= At AND (ends_at IS NULL OR ends_at >= At)
	At         time.Time         // moment ActiveOnly's window is evaluated against; zero value means now
	Types      []AchievementType // nil/empty = every type; non-empty scopes to a subset (e.g. Money's embedded tile grid)
}
