// Package goals implements the cross-domain Goals feature (see
// docs/functions/goals-spec.md): seven flat, sibling goal types, one shared
// "goals" table, a cached current_value refreshed on demand rather than
// recomputed on every read, and a shared webui tile grid embedded on the
// dedicated Goals page plus Money/Progress-browse/Workouts.
package goals

import "personal/domain"

// categoryApplicable reports whether goalType uses Category — money_spend
// only (see goals-spec.md Best Practices).
func categoryApplicable(t domain.GoalType) bool {
	return t == domain.GoalTypeMoneySpend
}

// unitApplicable reports whether goalType accepts an optional display Unit
// label — every type except money_saving/money_spend (see goals-spec.md
// Best Practices).
func unitApplicable(t domain.GoalType) bool {
	switch t {
	case domain.GoalTypeExerciseMaxWeight, domain.GoalTypeExerciseTotalVolume,
		domain.GoalTypeActivityOccurrenceCount, domain.GoalTypeActivityStreakCount,
		domain.GoalTypeManual:
		return true
	default:
		return false
	}
}

// exerciseApplicable reports whether goalType requires ExerciseID.
func exerciseApplicable(t domain.GoalType) bool {
	return t == domain.GoalTypeExerciseMaxWeight || t == domain.GoalTypeExerciseTotalVolume
}

// activityApplicable reports whether goalType requires ActivityID.
func activityApplicable(t domain.GoalType) bool {
	return t == domain.GoalTypeActivityOccurrenceCount || t == domain.GoalTypeActivityStreakCount
}
