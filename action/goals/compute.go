package goals

import (
	"context"
	"fmt"
	"time"

	"personal/domain"
	"personal/gateways"
)

// moneyBalanceSince returns the account's cumulative balance from the
// user's very first transaction through to — the same "all time" balance
// money-spec.md's dashboard uses, needed by both money_saving's baseline
// snapshot (create_goal) and its recompute (refresh_goals). A user with no
// transactions yet has no meaningful "since" bound, so from == to (an empty
// window, balance 0).
func moneyBalanceSince(ctx context.Context, db gateways.DB, userID int64, to time.Time) (float64, error) {
	summary, err := db.GetMoneySummary(ctx, userID)
	if err != nil {
		return 0, err
	}
	from := to
	if summary.FirstTransactionAt != nil {
		from = *summary.FirstTransactionAt
	}
	balance, err := db.GetBalance(ctx, userID, from, to)
	if err != nil {
		return 0, err
	}
	return balance.BalanceEUR, nil
}

// activityStreak walks activityID's progress points (newest-first, per
// ListProgress's default order) counting consecutive check-ins whose gap to
// the next-older point is within the activity's own FrequencyDays. If the
// very first gap — from at back to the newest point itself — already
// exceeds FrequencyDays, the streak has already lapsed (current_value = 0),
// matching ordinary streak semantics: a missed check-in resets it (see
// goals-spec.md Best Practices).
func activityStreak(ctx context.Context, db gateways.DB, userID int64, activityID int64, at time.Time) (float64, error) {
	activity, err := db.GetActivity(ctx, activityID, userID)
	if err != nil {
		return 0, err
	}
	if activity == nil {
		return 0, fmt.Errorf("activity not found")
	}

	points, err := db.ListProgress(ctx, domain.ProgressFilter{UserID: userID, ActivityID: activityID, To: at})
	if err != nil {
		return 0, err
	}
	if len(points) == 0 {
		return 0, nil
	}

	maxGap := time.Duration(activity.FrequencyDays) * 24 * time.Hour
	if at.Sub(points[0].ProgressAt) > maxGap {
		return 0, nil
	}

	streak := 1
	for i := 0; i < len(points)-1; i++ {
		gap := points[i].ProgressAt.Sub(points[i+1].ProgressAt)
		if gap > maxGap {
			break
		}
		streak++
	}
	return float64(streak), nil
}

// computeCurrentValue recomputes current_value for one of the six derived
// goal types (everything but manual) — the shared logic behind create_goal's
// initial snapshot and refresh_goals' recompute (see goals-spec.md's
// "Refresh Goals" sequence diagram). g must already have its per-type
// fields (Category/BaselineBalanceEUR/ExerciseID/ActivityID/StartsAt/EndsAt)
// populated; g.CurrentValue itself is not read.
func computeCurrentValue(ctx context.Context, db gateways.DB, userID int64, g domain.Goal, now time.Time) (float64, error) {
	switch g.GoalType {
	case domain.GoalTypeMoneySpend:
		category := ""
		if g.Category != nil {
			category = *g.Category
		}
		to := now
		if g.EndsAt != nil {
			to = *g.EndsAt
		}
		return db.GetCategorySpend(ctx, userID, category, g.StartsAt, to)

	case domain.GoalTypeMoneySaving:
		balance, err := moneyBalanceSince(ctx, db, userID, now)
		if err != nil {
			return 0, err
		}
		baseline := 0.0
		if g.BaselineBalanceEUR != nil {
			baseline = *g.BaselineBalanceEUR
		}
		return balance - baseline, nil

	case domain.GoalTypeExerciseMaxWeight:
		records, err := db.GetPersonalRecords(ctx, userID, *g.ExerciseID)
		if err != nil {
			return 0, err
		}
		if records.MaxWeight == nil {
			return 0, nil
		}
		return records.MaxWeight.WeightKg, nil

	case domain.GoalTypeExerciseTotalVolume:
		return db.GetExerciseVolume(ctx, userID, *g.ExerciseID, g.StartsAt)

	case domain.GoalTypeActivityOccurrenceCount:
		count, err := db.CountProgress(ctx, domain.ProgressFilter{UserID: userID, ActivityID: *g.ActivityID, From: g.StartsAt})
		if err != nil {
			return 0, err
		}
		return float64(count), nil

	case domain.GoalTypeActivityStreakCount:
		return activityStreak(ctx, db, userID, *g.ActivityID, now)

	default:
		return 0, fmt.Errorf("goal_type %q has no derived progress to compute", g.GoalType)
	}
}
