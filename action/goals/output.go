package goals

import (
	"time"

	"personal/domain"
)

// GoalOutput mirrors domain.Goal for JSON serialization, adding
// RemainingValue (Goal.RemainingValue(), see goals-spec.md Best Practices —
// no separate GetGoalProgress/GoalProgress type, just a getter on whatever
// Goal a caller already has).
type GoalOutput struct {
	ID                 int64      `json:"id"`
	Name               string     `json:"name"`
	GoalType           string     `json:"goal_type"`
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

func toGoalOutput(g domain.Goal) GoalOutput {
	return GoalOutput{
		ID:                 g.ID,
		Name:               g.Name,
		GoalType:           string(g.GoalType),
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
