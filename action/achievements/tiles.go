package achievements

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

// progressLabel formats an achievement's current/target progress per achievement_type, per
// achievements-spec.md's BuildAchievementTiles doc: "€420 / €1,000 (42%)", "82kg / 100kg",
// "3200kg / 5000kg", "12 / 30 sessions", "14 / 30 day streak".
func progressLabel(g domain.Achievement, rawPercent float64) string {
	switch g.AchievementType {
	case domain.AchievementTypeMoneySaving, domain.AchievementTypeMoneySpend:
		return fmt.Sprintf("€%.2f / €%.2f (%.0f%%)", g.CurrentValue, g.TargetValue, rawPercent)
	case domain.AchievementTypeExerciseMaxWeight, domain.AchievementTypeExerciseTotalVolume:
		return fmt.Sprintf("%skg / %skg", formatNumber(g.CurrentValue), formatNumber(g.TargetValue))
	case domain.AchievementTypeActivityStreakCount:
		return fmt.Sprintf("%s / %s day streak", formatNumber(g.CurrentValue), formatNumber(g.TargetValue))
	default: // activity_occurrence_count, manual
		return fmt.Sprintf("%s / %s%s", formatNumber(g.CurrentValue), formatNumber(g.TargetValue), unitSuffix(g.Unit))
	}
}

// achievementLinkURL derives an achievement card's drill-down link per achievement_type, per
// achievements-spec.md Best Practices. money_spend links to its category filter on
// the money transactions page (same URL shape as action/money's
// categoryLinkURL, reconstructed here rather than imported — action/money
// already imports action/achievements for its embedded tile grid, so importing
// back would cycle). exercise_max_weight/exercise_total_volume link to
// their exercise's page, activity_occurrence_count/activity_streak_count to
// their activity's page. money_saving (no Category of its own) and manual
// (no underlying page) get no link.
func achievementLinkURL(g domain.Achievement) string {
	switch g.AchievementType {
	case domain.AchievementTypeMoneySpend:
		if g.Category == nil {
			return ""
		}
		v := url.Values{}
		v.Set("category", *g.Category)
		return "/web/money/transactions?" + v.Encode()
	case domain.AchievementTypeExerciseMaxWeight, domain.AchievementTypeExerciseTotalVolume:
		if g.ExerciseID == nil {
			return ""
		}
		return fmt.Sprintf("/web/workouts/%d", *g.ExerciseID)
	case domain.AchievementTypeActivityOccurrenceCount, domain.AchievementTypeActivityStreakCount:
		if g.ActivityID == nil {
			return ""
		}
		return fmt.Sprintf("/web/progress/browse/%d", *g.ActivityID)
	default: // money_saving, manual
		return ""
	}
}

func toAchievementTileData(g domain.Achievement) webui.AchievementTileData {
	rawPercent := 0.0
	if g.TargetValue > 0 {
		rawPercent = g.CurrentValue / g.TargetValue * 100
	}

	deadline := ""
	if g.EndsAt != nil {
		deadline = "by " + g.EndsAt.Format("Jan 2, 2006")
	}

	return webui.AchievementTileData{
		Name:            g.Name,
		ProgressLabel:   progressLabel(g, rawPercent),
		PercentComplete: clampPercent(rawPercent),
		Deadline:        deadline,
		OverTarget:      g.AchievementType == domain.AchievementTypeMoneySpend && g.CurrentValue > g.TargetValue,
		LinkURL:         achievementLinkURL(g),
	}
}

// achievementCategoryRank buckets an AchievementType for the all-types dashboard sort order
// (see achievements-spec.md Best Practices): activities, then money, then gym,
// then everything else (manual).
func achievementCategoryRank(t domain.AchievementType) int {
	switch t {
	case domain.AchievementTypeActivityOccurrenceCount, domain.AchievementTypeActivityStreakCount:
		return 1
	case domain.AchievementTypeMoneySaving, domain.AchievementTypeMoneySpend:
		return 2
	case domain.AchievementTypeExerciseMaxWeight, domain.AchievementTypeExerciseTotalVolume:
		return 3
	default: // manual
		return 4
	}
}

// BuildAchievementTiles reads a user's active achievements — every type (types == nil,
// the dedicated Achievements page and the e-ink page) or a subset (an embedding
// page's own domain types) — and maps each to a webui.AchievementTileData, the
// single place that turns an Achievement into its tile's display formatting. A
// plain cached read via ListAchievements, no per-achievement_type computation (see
// achievements-spec.md Best Practices) — callers needing fresh numbers must call
// refresh_achievements first. When types is nil, the mixed-type result is
// additionally re-sorted into category buckets (activities, money, gym,
// others) before mapping — a non-nil types subset skips this and keeps
// ListAchievements' starts_at order, since a single-domain grid has nothing to
// group by category.
func BuildAchievementTiles(ctx context.Context, db gateways.DB, userID int64, at time.Time, types []domain.AchievementType) ([]webui.AchievementTileData, error) {
	achievementList, err := db.ListAchievements(ctx, domain.AchievementFilter{UserID: userID, ActiveOnly: true, At: at, Types: types})
	if err != nil {
		return nil, err
	}
	if types == nil {
		sort.SliceStable(achievementList, func(i, j int) bool {
			ri, rj := achievementCategoryRank(achievementList[i].AchievementType), achievementCategoryRank(achievementList[j].AchievementType)
			if ri != rj {
				return ri < rj
			}
			return achievementList[i].ID < achievementList[j].ID
		})
	}
	tiles := make([]webui.AchievementTileData, len(achievementList))
	for i, g := range achievementList {
		tiles[i] = toAchievementTileData(g)
	}
	return tiles, nil
}
