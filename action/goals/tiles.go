package goals

import (
	"context"
	"fmt"
	"net/url"
	"sort"
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

// goalLinkURL derives a goal card's drill-down link per goal_type, per
// goals-spec.md Best Practices. money_spend links to its category filter on
// the money transactions page (same URL shape as action/money's
// categoryLinkURL, reconstructed here rather than imported — action/money
// already imports action/goals for its embedded tile grid, so importing
// back would cycle). exercise_max_weight/exercise_total_volume link to
// their exercise's page, activity_occurrence_count/activity_streak_count to
// their activity's page. money_saving (no Category of its own) and manual
// (no underlying page) get no link.
func goalLinkURL(g domain.Goal) string {
	switch g.GoalType {
	case domain.GoalTypeMoneySpend:
		if g.Category == nil {
			return ""
		}
		v := url.Values{}
		v.Set("category", *g.Category)
		return "/web/money/transactions?" + v.Encode()
	case domain.GoalTypeExerciseMaxWeight, domain.GoalTypeExerciseTotalVolume:
		if g.ExerciseID == nil {
			return ""
		}
		return fmt.Sprintf("/web/workouts/%d", *g.ExerciseID)
	case domain.GoalTypeActivityOccurrenceCount, domain.GoalTypeActivityStreakCount:
		if g.ActivityID == nil {
			return ""
		}
		return fmt.Sprintf("/web/progress/browse/%d", *g.ActivityID)
	default: // money_saving, manual
		return ""
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
		LinkURL:         goalLinkURL(g),
	}
}

// goalCategoryRank buckets a GoalType for the all-types dashboard sort order
// (see goals-spec.md Best Practices): activities, then money, then gym,
// then everything else (manual).
func goalCategoryRank(t domain.GoalType) int {
	switch t {
	case domain.GoalTypeActivityOccurrenceCount, domain.GoalTypeActivityStreakCount:
		return 1
	case domain.GoalTypeMoneySaving, domain.GoalTypeMoneySpend:
		return 2
	case domain.GoalTypeExerciseMaxWeight, domain.GoalTypeExerciseTotalVolume:
		return 3
	default: // manual
		return 4
	}
}

// BuildGoalTiles reads a user's active goals — every type (types == nil,
// the dedicated Goals page and the e-ink page) or a subset (an embedding
// page's own domain types) — and maps each to a webui.GoalTileData, the
// single place that turns a Goal into its tile's display formatting. A
// plain cached read via ListGoals, no per-goal_type computation (see
// goals-spec.md Best Practices) — callers needing fresh numbers must call
// refresh_goals first. When types is nil, the mixed-type result is
// additionally re-sorted into category buckets (activities, money, gym,
// others) before mapping — a non-nil types subset skips this and keeps
// ListGoals' starts_at order, since a single-domain grid has nothing to
// group by category.
func BuildGoalTiles(ctx context.Context, db gateways.DB, userID int64, at time.Time, types []domain.GoalType) ([]webui.GoalTileData, error) {
	goalList, err := db.ListGoals(ctx, domain.GoalFilter{UserID: userID, ActiveOnly: true, At: at, Types: types})
	if err != nil {
		return nil, err
	}
	if types == nil {
		sort.SliceStable(goalList, func(i, j int) bool {
			ri, rj := goalCategoryRank(goalList[i].GoalType), goalCategoryRank(goalList[j].GoalType)
			if ri != rj {
				return ri < rj
			}
			return goalList[i].ID < goalList[j].ID
		})
	}
	tiles := make([]webui.GoalTileData, len(goalList))
	for i, g := range goalList {
		tiles[i] = toGoalTileData(g)
	}
	return tiles, nil
}
