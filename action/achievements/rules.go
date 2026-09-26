// Package achievements implements the cross-domain Achievements feature (see
// docs/functions/achievements-spec.md): seven flat, sibling achievement types, one shared
// "achievements" table, a cached current_value refreshed on demand rather than
// recomputed on every read, and a shared webui tile grid embedded on the
// dedicated Achievements page plus Money/Progress-browse/Workouts.
package achievements

import "personal/domain"

// categoryApplicable reports whether achievementType uses Category — money_spend
// only (see achievements-spec.md Best Practices).
func categoryApplicable(t domain.AchievementType) bool {
	return t == domain.AchievementTypeMoneySpend
}

// unitApplicable reports whether achievementType accepts an optional display Unit
// label — every type except money_saving/money_spend (see achievements-spec.md
// Best Practices).
func unitApplicable(t domain.AchievementType) bool {
	switch t {
	case domain.AchievementTypeExerciseMaxWeight, domain.AchievementTypeExerciseTotalVolume,
		domain.AchievementTypeActivityOccurrenceCount, domain.AchievementTypeActivityStreakCount,
		domain.AchievementTypeManual:
		return true
	default:
		return false
	}
}

// exerciseApplicable reports whether achievementType requires ExerciseID.
func exerciseApplicable(t domain.AchievementType) bool {
	return t == domain.AchievementTypeExerciseMaxWeight || t == domain.AchievementTypeExerciseTotalVolume
}

// activityApplicable reports whether achievementType requires ActivityID.
func activityApplicable(t domain.AchievementType) bool {
	return t == domain.AchievementTypeActivityOccurrenceCount || t == domain.AchievementTypeActivityStreakCount
}
