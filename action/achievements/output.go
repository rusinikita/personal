package achievements

import (
	"time"

	"personal/domain"
)

// AchievementOutput mirrors domain.Achievement for JSON serialization, adding
// RemainingValue (Achievement.RemainingValue(), see achievements-spec.md Best Practices —
// no separate GetAchievementProgress/AchievementProgress type, just a getter on whatever
// Achievement a caller already has).
type AchievementOutput struct {
	ID                 int64      `json:"id"`
	Name               string     `json:"name"`
	AchievementType    string     `json:"achievement_type"`
	ExerciseID         *int64     `json:"exercise_id,omitempty"`
	ActivityID         *int64     `json:"activity_id,omitempty"`
	TargetValue        float64    `json:"target_value"`
	CurrentValue       float64    `json:"current_value"`
	RemainingValue     float64    `json:"remaining_value"`
	Category           *string    `json:"category,omitempty"`
	Unit               *string    `json:"unit,omitempty"`
	BaselineBalanceEUR *float64   `json:"baseline_balance_eur,omitempty"`
	StartsAt           time.Time  `json:"starts_at"`
	EndsAt             *time.Time `json:"ends_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func toAchievementOutput(g domain.Achievement) AchievementOutput {
	return AchievementOutput{
		ID:                 g.ID,
		Name:               g.Name,
		AchievementType:    string(g.AchievementType),
		ExerciseID:         g.ExerciseID,
		ActivityID:         g.ActivityID,
		TargetValue:        g.TargetValue,
		CurrentValue:       g.CurrentValue,
		RemainingValue:     g.RemainingValue(),
		Category:           g.Category,
		Unit:               g.Unit,
		BaselineBalanceEUR: g.BaselineBalanceEUR,
		StartsAt:           g.StartsAt,
		EndsAt:             g.EndsAt,
		CreatedAt:          g.CreatedAt,
		UpdatedAt:          g.UpdatedAt,
	}
}
