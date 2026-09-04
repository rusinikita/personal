package goals

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"personal/action/webui"
	"personal/domain"
	"personal/gateways"
)

// formatNumber trims a float to its minimal decimal representation — 82 for
// 82.0, 82.5 for 82.5 — so tile labels read like "82kg / 100kg" instead of
// "82.0kg / 100.0kg".
func formatNumber(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func clampPercent(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 100:
		return 100
	default:
		return v
	}
}

func unitSuffix(unit *string) string {
	if unit == nil || *unit == "" {
		return ""
	}
	return " " + *unit
}

// progressLabel formats a goal's current/target progress per goal_type, per
// goals-spec.md's BuildGoalTiles doc: "€420 / €1,000 (42%)", "82kg / 100kg",
// "3200kg / 5000kg", "12 / 30 sessions", "14 / 30 day streak".
func progressLabel(g domain.Goal, rawPercent float64) string {
	switch g.GoalType {
	case domain.GoalTypeMoneySaving, domain.GoalTypeMoneySpend:
		return fmt.Sprintf("€%.2f / €%.2f (%.0f%%)", g.CurrentValue, g.TargetValue, rawPercent)
	case domain.GoalTypeExerciseMaxWeight, domain.GoalTypeExerciseTotalVolume:
		return fmt.Sprintf("%skg / %skg", formatNumber(g.CurrentValue), formatNumber(g.TargetValue))
	case domain.GoalTypeActivityStreakCount:
		return fmt.Sprintf("%s / %s day streak", formatNumber(g.CurrentValue), formatNumber(g.TargetValue))
	default: // activity_occurrence_count, manual
		return fmt.Sprintf("%s / %s%s", formatNumber(g.CurrentValue), formatNumber(g.TargetValue), unitSuffix(g.Unit))
	}
}

func toGoalTileData(g domain.Goal) webui.GoalTileData {
	rawPercent := 0.0
	if g.TargetValue > 0 {
		rawPercent = g.CurrentValue / g.TargetValue * 100
	}

	deadline := ""
	if g.EndsAt != nil {
		deadline = "by " + g.EndsAt.Format("Jan 2, 2006")
	}

	return webui.GoalTileData{
		Name:            g.Name,
		ProgressLabel:   progressLabel(g, rawPercent),
		PercentComplete: clampPercent(rawPercent),
		Deadline:        deadline,
		OverTarget:      g.GoalType == domain.GoalTypeMoneySpend && g.CurrentValue > g.TargetValue,
	}
}

// BuildGoalTiles reads a user's active goals — every type (types == nil,
// the dedicated Goals page) or a subset (an embedding page's own domain
// types) — and maps each to a webui.GoalTileData, the single place that
// turns a Goal into its tile's display formatting. A plain cached read via
// ListGoals, no per-goal_type computation (see goals-spec.md Best
// Practices) — callers needing fresh numbers must call refresh_goals first.
func BuildGoalTiles(ctx context.Context, db gateways.DB, userID int64, at time.Time, types []domain.GoalType) ([]webui.GoalTileData, error) {
	goalList, err := db.ListGoals(ctx, domain.GoalFilter{UserID: userID, ActiveOnly: true, At: at, Types: types})
	if err != nil {
		return nil, err
	}
	tiles := make([]webui.GoalTileData, len(goalList))
	for i, g := range goalList {
		tiles[i] = toGoalTileData(g)
	}
	return tiles, nil
}
