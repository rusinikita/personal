package gateways

import (
	"context"
	"time"

	"personal/domain"
)

type DB interface {
	// Existing methods
	CreateFood(ctx context.Context, food *domain.Food) (int64, error)
	GetFood(ctx context.Context, id int64) (*domain.Food, error)

	// New methods for consumption logging
	AddConsumptionLog(ctx context.Context, log *domain.ConsumptionLog) error
	SearchFood(ctx context.Context, filter domain.FoodFilter) ([]*domain.Food, error)

	// Methods for testing verification
	GetConsumptionLog(ctx context.Context, userID int64, consumedAt time.Time) (*domain.ConsumptionLog, error)
	GetConsumptionLogsByUser(ctx context.Context, userID int64) ([]*domain.ConsumptionLog, error)

	// Nutrition stats methods
	GetLastConsumptionTime(ctx context.Context, userID int64) (*time.Time, error)
	GetNutritionStats(ctx context.Context, filter domain.NutritionStatsFilter) ([]domain.NutritionStats, error)

	// Top products methods
	GetTopProducts(ctx context.Context, userID int64, from time.Time, to time.Time, limit int) ([]domain.FoodStats, error)

	// Exercise methods
	CreateExercise(ctx context.Context, exercise *domain.Exercise) (int64, error)
	ListWithLastUsed(ctx context.Context, userID int64) ([]domain.Exercise, error)
	ListExercises(ctx context.Context, userID int64, limit int64) ([]domain.Exercise, error)
	SearchExercises(ctx context.Context, userID int64, query string) ([]domain.Exercise, error)
	GetExercise(ctx context.Context, exerciseID int64, userID int64) (*domain.Exercise, error)
	UpdateExercise(ctx context.Context, exercise *domain.Exercise) error
	MoveSetsBetweenExercises(ctx context.Context, sourceID, targetID, userID int64) (int64, error)
	DeleteExercise(ctx context.Context, exerciseID int64, userID int64) error
	GetPersonalRecords(ctx context.Context, userID int64, exerciseID int64) (*domain.PersonalRecords, error)
	ListPersonalRecords(ctx context.Context, userID int64) ([]domain.ExercisePersonalRecords, error)
	ListExercisesByUsage(ctx context.Context, userID int64) ([]domain.Exercise, error)
	ListExerciseSets(ctx context.Context, userID int64, exerciseID int64, workoutLimit int) ([]domain.Set, error)

	// Workout methods
	CreateWorkout(ctx context.Context, workout *domain.Workout) (int64, error)
	CloseWorkout(ctx context.Context, workoutID int64, completedAt time.Time) error
	ListWorkouts(ctx context.Context, userID int64) ([]domain.Workout, error)
	GetWorkoutByDate(ctx context.Context, userID int64, date time.Time) (*domain.Workout, error)

	// Set methods
	CreateSet(ctx context.Context, set *domain.Set) (int64, error)
	GetLastSet(ctx context.Context, userID int64) (*domain.WorkoutSet, error)
	GetSetByID(ctx context.Context, setID int64, userID int64) (*domain.SetWithExercise, error)
	DeleteSet(ctx context.Context, setID int64, userID int64) error
	ListSets(ctx context.Context, userID int64, from time.Time, to time.Time) ([]domain.Set, error)
	GetExercisesByIDs(ctx context.Context, userID int64, exerciseIDs []int64) ([]domain.Exercise, error)
	GetWorkoutsByIDs(ctx context.Context, userID int64, workoutIDs []int64) ([]domain.Workout, error)
	GetExerciseHistory(ctx context.Context, userID int64, exerciseID int64, limit int, offset int) ([]domain.Workout, error)
	ListSetsByExerciseAndWorkouts(ctx context.Context, userID int64, exerciseID int64, workoutIDs []int64) ([]domain.Set, error)

	// Money tracking methods
	AddTransactions(ctx context.Context, txs []*domain.Transaction) ([]*domain.Transaction, error)
	EditTransactions(ctx context.Context, userID int64, updates []domain.TransactionUpdate) (int, error)
	DeleteTransaction(ctx context.Context, id int64, userID int64) error
	GetTransactions(ctx context.Context, filter domain.TransactionFilter) ([]*domain.Transaction, int, error)
	GetSpendingByCategory(ctx context.Context, userID int64, from, to time.Time, depth int) ([]domain.SpendingByCategory, error)
	GetTopMerchants(ctx context.Context, userID int64, from, to time.Time, limit int) ([]domain.MerchantSummary, error)
	GetSpendingForPeriod(ctx context.Context, userID int64, from, to time.Time) ([]domain.SpendingByCategory, error)
	GetBalance(ctx context.Context, userID int64, from, to time.Time) (domain.BalanceResult, error)

	// GetMoneySummary returns the user's first transaction date and last
	// sync/import timestamp — powers the web dashboard's sync-freshness
	// stat tile and the months-span used for every "average monthly" figure.
	GetMoneySummary(ctx context.Context, userID int64) (domain.MoneySummary, error)

	// GetDailyTransactionSummary returns one DailySummary per day in
	// [from, to] that has at least one transaction, day boundaries computed
	// in the display timezone — powers the transaction calendar.
	GetDailyTransactionSummary(ctx context.Context, userID int64, from, to time.Time) ([]domain.DailySummary, error)

	// Progress tracking methods
	ListLifeParts(ctx context.Context, userID int64) ([]domain.LifePart, error) // read-only; life_parts rows are inserted by hand, no write method yet
	CreateActivity(ctx context.Context, activity *domain.Activity) (int64, error)
	ListActivities(ctx context.Context, filter domain.ActivityFilter) ([]domain.Activity, error)
	CountActivities(ctx context.Context, filter domain.ActivityFilter) (int, error)
	GetActivity(ctx context.Context, activityID int64, userID int64) (*domain.Activity, error)
	UpdateActivity(ctx context.Context, activity *domain.Activity) error      // also writes progress_type, status, deferred_until
	DeleteActivity(ctx context.Context, activityID int64, userID int64) error // hard delete; blocked if an achievement still references the activity
	CreateProgress(ctx context.Context, progress *domain.ActivityPoint) (int64, error)
	GetProgress(ctx context.Context, progressID int64, userID int64) (*domain.ActivityPoint, error)
	ListProgress(ctx context.Context, filter domain.ProgressFilter) ([]domain.ActivityPoint, error)
	CountProgress(ctx context.Context, filter domain.ProgressFilter) (int, error)
	UpdateProgress(ctx context.Context, progress *domain.ActivityPoint) error
	DeleteProgress(ctx context.Context, progressID int64, userID int64) error
	GetTrendStats(ctx context.Context, activityID int64, userID int64, from time.Time, to time.Time) (domain.TrendStats, error)
	SearchProgressNotes(ctx context.Context, filter domain.ProgressNoteSearchFilter) ([]domain.ActivityPointWithActivity, error)

	// Step CRUD (next-action steps owned by an activity, see progress-spec.md)
	CreateStep(ctx context.Context, step *domain.Step) (int64, error)
	GetStep(ctx context.Context, stepID int64, userID int64) (*domain.Step, error)
	ListSteps(ctx context.Context, filter domain.StepFilter) ([]domain.Step, error)                         // used for progress-point form checkboxes and browse/drill-down display; caller is expected to only call this for an activity it already knows is status=active
	ListStepsWithActivity(ctx context.Context, filter domain.StepFilter) ([]domain.StepWithActivity, error) // used by get_step_list; joins to activities and always filters activities.status='active' server-side, regardless of filter.Statuses
	UpdateStep(ctx context.Context, step *domain.Step) error                                                // rename, status, closed_at, completed_by_progress_point_id — caller has already merged partial-update fields onto a fetched step
	DeleteStep(ctx context.Context, stepID int64, userID int64) error

	// Ideas (see docs/functions/ideas-spec.md)
	CreateIdea(ctx context.Context, idea *domain.Idea) (int64, error)
	GetIdea(ctx context.Context, ideaID int64, userID int64) (*domain.Idea, error)          // fills SurfaceCount/LastSurfacedAt
	ListIdeas(ctx context.Context, filter domain.IdeaFilter) ([]domain.Idea, error)         // fills SurfaceCount/LastSurfacedAt; ordered by created_at ASC, or the Limit newest DESC when Limit > 0
	SearchIdeas(ctx context.Context, filter domain.IdeaSearchFilter) ([]domain.Idea, error) // one variant; fills SurfaceCount/LastSurfacedAt; the action merges variants and counts match_count
	UpdateIdea(ctx context.Context, idea *domain.Idea) error                                // body, status, updated_at; caller has already merged the change onto a fetched idea
	ResolveIdea(ctx context.Context, resolve domain.IdeaResolve) error                      // sets status=resolved + resolution fields (overwrites them when re-resolving a blocked idea); for merged also re-points ideas merged into IdeaID to MergedIntoID in the same statement

	// Docs (see docs/functions/docs-spec.md)
	CreateDoc(ctx context.Context, doc *domain.Doc) (int64, error)
	GetDoc(ctx context.Context, docID int64, userID int64) (*domain.Doc, error) // nil when not found
	ListDocs(ctx context.Context, userID int64) ([]domain.Doc, error)           // ordered by id
	UpdateDoc(ctx context.Context, doc *domain.Doc) error                       // title, description, updated_at; caller has already merged the change onto a fetched document

	// Every chunk write below also sets docs.updated_at of the chunk's document, in the same transaction
	CreateDocChunk(ctx context.Context, chunk *domain.DocChunk) (int64, error)               // at chunk.Position, or at the end when it is 0; in one transaction first moves the chunks at that position and after it down by one, then inserts the new chunk, so two chunks never share a position
	GetDocChunk(ctx context.Context, chunkID int64, userID int64) (*domain.DocChunk, error)  // nil when not found
	ListDocChunks(ctx context.Context, docID int64, userID int64) ([]domain.DocChunk, error) // ordered by position
	UpdateDocChunk(ctx context.Context, chunk *domain.DocChunk) error                        // body, previous_body, updated_at, position; caller has already merged the change onto a fetched chunk; a new position first shifts the chunks between the old and the new position, then places the chunk, in one transaction
	DeleteDocChunk(ctx context.Context, chunkID int64, userID int64) error                   // hard delete; moves the following chunks of the document up, in one transaction

	// Learning plans (see docs/functions/learning-spec.md)
	CreateLearningPlan(ctx context.Context, plan *domain.LearningPlan) (int64, error)
	GetLearningPlan(ctx context.Context, planID int64, userID int64) (*domain.LearningPlan, error)          // nil when not found
	ListLearningPlans(ctx context.Context, filter domain.LearningPlanFilter) ([]domain.LearningPlan, error) // active first, then paused, then completed; newest first inside a status
	UpdateLearningPlan(ctx context.Context, plan *domain.LearningPlan) error

	// Skills
	CreateSkill(ctx context.Context, skill *domain.Skill) (int64, error)
	GetSkill(ctx context.Context, skillID int64, userID int64) (*domain.Skill, error)   // nil when not found; fills read-only fields
	ListSkills(ctx context.Context, planID int64, userID int64) ([]domain.Skill, error) // one query: fills read-only fields, Scores via ARRAY_AGG(score ORDER BY reviewed_at) grouped by skill; ordered by created_at ASC
	UpdateSkill(ctx context.Context, skill *domain.Skill) error

	// Learning exercises
	CreateLearningExercises(ctx context.Context, exercises []domain.LearningExercise) ([]int64, error)         // one transaction
	GetLearningExercise(ctx context.Context, exerciseID int64, userID int64) (*domain.LearningExercise, error) // nil when not found
	ListLearningExercises(ctx context.Context, filter domain.LearningExerciseFilter) ([]domain.LearningExercise, error)
	CountLearningExercises(ctx context.Context, filter domain.LearningExerciseFilter) (int, error) // ignores Limit and Offset
	GetLearningExerciseStats(ctx context.Context, userID int64, planIDs []int64, dueAt time.Time) ([]domain.LearningExerciseStats, error)
	UpdateLearningExercise(ctx context.Context, exercise *domain.LearningExercise) error // kind, content, planned_for
	SubmitLearningExerciseAnswer(ctx context.Context, exerciseID int64, userID int64, answer string, answeredAt time.Time) error
	ReviewLearningExercises(ctx context.Context, userID int64, reviews []domain.LearningExerciseReview) error // one transaction
	DeleteLearningExercise(ctx context.Context, exerciseID int64, userID int64) error

	// Vocabulary
	CreateVocabulary(ctx context.Context, words []domain.VocabularyWord) ([]int64, error)                                 // one transaction
	ListVocabulary(ctx context.Context, filter domain.VocabularyFilter) ([]domain.VocabularyWord, error)                  // one query: fills read-only fields, Scores via ARRAY_AGG(score ORDER BY reviewed_at) grouped by word
	CountVocabulary(ctx context.Context, filter domain.VocabularyFilter) (int, error)                                     // ignores Limit and Offset
	SearchVocabulary(ctx context.Context, filter domain.VocabularySearchFilter) ([]domain.VocabularyWord, error)          // one query, same read-only fields and Scores as ListVocabulary
	FindVocabularyTerms(ctx context.Context, planID int64, userID int64, terms []string) ([]string, error)                // case-insensitive; returns terms already in the plan
	CountDueVocabularyReviews(ctx context.Context, userID int64, planIDs []int64, dueAt time.Time) (map[int64]int, error) // per plan
	UpdateVocabulary(ctx context.Context, word *domain.VocabularyWord) error                                              // term, translation, notes, skill
	DeleteVocabulary(ctx context.Context, wordID int64, userID int64) error                                               // its reviews go with it

	// Vocabulary reviews
	CreateVocabularyReviews(ctx context.Context, reviews []domain.VocabularyReview) ([]int64, error) // one transaction; pending ones from plan_vocabulary, done ones from log_vocabulary_reviews
	ListVocabularyReviews(ctx context.Context, filter domain.VocabularyReviewFilter) ([]domain.VocabularyReview, error)
	CountVocabularyReviews(ctx context.Context, filter domain.VocabularyReviewFilter) (int, error)                   // ignores Limit and Offset
	RecordVocabularyReview(ctx context.Context, reviewID int64, userID int64, score int, reviewedAt time.Time) error // pending only
	DeleteVocabularyReviews(ctx context.Context, userID int64, reviewIDs []int64) error                              // one transaction; pending only

	// Achievements tracking methods (see docs/functions/achievements-spec.md)

	// CreateAchievement and UpdateAchievement are the only two write methods on the whole
	// repository for achievements — every write to an existing achievement, no matter the
	// caller (update_achievement, refresh_achievements, log_achievement_progress), goes through
	// UpdateAchievement's generic partial update (see achievements-spec.md Best Practices).
	CreateAchievement(ctx context.Context, g *domain.Achievement) (int64, error)
	UpdateAchievement(ctx context.Context, userID int64, update domain.AchievementUpdate) error

	GetAchievement(ctx context.Context, achievementID int64, userID int64) (*domain.Achievement, error)
	ListAchievements(ctx context.Context, filter domain.AchievementFilter) ([]domain.Achievement, error)

	// GetCategorySpend sums transactions.amount_eur where category starts
	// with the given prefix, within [from, to] — powers money_spend achievements.
	// Same query the old Budget/BudgetProgress used, now achievement-scoped.
	GetCategorySpend(ctx context.Context, userID int64, category string, from, to time.Time) (float64, error)

	// GetExerciseVolume sums sets.weight_kg * sets.reps for an exercise
	// since a given time — powers exercise_total_volume achievements.
	GetExerciseVolume(ctx context.Context, userID int64, exerciseID int64, since time.Time) (float64, error)
}

type DBMaintainer interface {
	ApplyMigrations(ctx context.Context) error
	TruncateUserData(ctx context.Context, userID int64) error
}

// Telegram wraps the Telegram Bot API for outbound notifications.
type Telegram interface {
	// SendMessage sends text to the configured chat. parseMode is "" (plain text),
	// "Markdown", "MarkdownV2", or "HTML" per the Telegram Bot API parse_mode param.
	SendMessage(ctx context.Context, text string, parseMode string) (messageID int64, err error)
}
