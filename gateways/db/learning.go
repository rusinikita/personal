package db

// Learning (see docs/functions/learning-spec.md)

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"

	"personal/domain"
)

func psql() squirrel.StatementBuilderType {
	return squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
}

// countRows wraps a select into SELECT COUNT(*).
func (r *repository) countRows(ctx context.Context, query squirrel.SelectBuilder) (int, error) {
	sql, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build query: %w", err)
	}

	var count int
	err = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM ("+sql+") q", args...).Scan(&count)
	return count, err
}

// ---------------------------------------------------------------------------
// Learning plans
// ---------------------------------------------------------------------------

const learningPlanColumns = "id, user_id, name, description, status, created_at, updated_at"

func scanLearningPlan(row pgx.Row) (*domain.LearningPlan, error) {
	var plan domain.LearningPlan
	err := row.Scan(&plan.ID, &plan.UserID, &plan.Name, &plan.Description, &plan.Status, &plan.CreatedAt, &plan.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func (r *repository) CreateLearningPlan(ctx context.Context, plan *domain.LearningPlan) (int64, error) {
	if plan.Status == "" {
		plan.Status = domain.LearningPlanActive
	}
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = time.Now().UTC()
	}
	plan.UpdatedAt = plan.CreatedAt

	var id int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO learning_plans (user_id, name, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id`,
		plan.UserID, plan.Name, plan.Description, plan.Status, plan.CreatedAt,
	).Scan(&id)

	return id, err
}

func (r *repository) GetLearningPlan(ctx context.Context, planID int64, userID int64) (*domain.LearningPlan, error) {
	plan, err := scanLearningPlan(r.db.QueryRow(ctx,
		`SELECT `+learningPlanColumns+` FROM learning_plans WHERE id = $1 AND user_id = $2`,
		planID, userID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return plan, err
}

func (r *repository) ListLearningPlans(ctx context.Context, filter domain.LearningPlanFilter) ([]domain.LearningPlan, error) {
	statuses := filter.Statuses
	if len(statuses) == 0 {
		statuses = []domain.LearningPlanStatus{domain.LearningPlanActive, domain.LearningPlanPaused}
	}

	sql, args, err := psql().
		Select(learningPlanColumns).
		From("learning_plans").
		Where(squirrel.Eq{"user_id": filter.UserID, "status": statuses}).
		OrderBy("CASE status WHEN 'active' THEN 0 WHEN 'paused' THEN 1 ELSE 2 END", "created_at DESC", "id DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query learning plans: %w", err)
	}
	defer rows.Close()

	var plans []domain.LearningPlan
	for rows.Next() {
		plan, err := scanLearningPlan(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan learning plan: %w", err)
		}
		plans = append(plans, *plan)
	}

	return plans, rows.Err()
}

func (r *repository) UpdateLearningPlan(ctx context.Context, plan *domain.LearningPlan) error {
	plan.UpdatedAt = time.Now().UTC()

	result, err := r.db.Exec(ctx, `
		UPDATE learning_plans
		SET name = $1, description = $2, status = $3, updated_at = $4
		WHERE id = $5 AND user_id = $6`,
		plan.Name, plan.Description, plan.Status, plan.UpdatedAt, plan.ID, plan.UserID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("learning plan not found")
	}

	return nil
}

// ---------------------------------------------------------------------------
// Skills
// ---------------------------------------------------------------------------

// skillSelect selects skills with the read-only fields computed from their
// exercises; callers add WHERE clauses on "s".
func skillSelect() squirrel.SelectBuilder {
	return psql().
		Select(
			"s.id", "s.user_id", "s.learning_plan_id", "s.name", "s.created_at", "s.updated_at",
			"COUNT(e.id) FILTER (WHERE e.answer IS NOT NULL) AS answered_count",
			"COUNT(e.id) FILTER (WHERE e.answer IS NULL) AS planned_count",
			"MAX(e.answered_at) AS last_practiced_at",
			"(ARRAY_AGG(e.score::int ORDER BY e.reviewed_at DESC, e.id DESC) FILTER (WHERE e.score IS NOT NULL))[1] AS last_score",
			"ARRAY_AGG(e.score::int ORDER BY e.reviewed_at, e.id) FILTER (WHERE e.score IS NOT NULL) AS scores",
		).
		From("skills s").
		LeftJoin("learning_exercises e ON e.skill_id = s.id").
		GroupBy("s.id")
}

func (r *repository) querySkills(ctx context.Context, query squirrel.SelectBuilder) ([]domain.Skill, error) {
	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query skills: %w", err)
	}
	defer rows.Close()

	var skills []domain.Skill
	for rows.Next() {
		var s domain.Skill
		err := rows.Scan(
			&s.ID, &s.UserID, &s.LearningPlanID, &s.Name, &s.CreatedAt, &s.UpdatedAt,
			&s.AnsweredCount, &s.PlannedCount, &s.LastPracticedAt, &s.LastScore, &s.Scores,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan skill: %w", err)
		}
		skills = append(skills, s)
	}

	return skills, rows.Err()
}

func (r *repository) CreateSkill(ctx context.Context, skill *domain.Skill) (int64, error) {
	if skill.CreatedAt.IsZero() {
		skill.CreatedAt = time.Now().UTC()
	}
	skill.UpdatedAt = skill.CreatedAt

	var id int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO skills (user_id, learning_plan_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		RETURNING id`,
		skill.UserID, skill.LearningPlanID, skill.Name, skill.CreatedAt,
	).Scan(&id)

	return id, err
}

func (r *repository) GetSkill(ctx context.Context, skillID int64, userID int64) (*domain.Skill, error) {
	skills, err := r.querySkills(ctx, skillSelect().Where(squirrel.Eq{"s.id": skillID, "s.user_id": userID}))
	if err != nil || len(skills) == 0 {
		return nil, err
	}
	return &skills[0], nil
}

func (r *repository) ListSkills(ctx context.Context, planID int64, userID int64) ([]domain.Skill, error) {
	return r.querySkills(ctx, skillSelect().
		Where(squirrel.Eq{"s.learning_plan_id": planID, "s.user_id": userID}).
		OrderBy("s.created_at ASC", "s.id ASC"))
}

func (r *repository) UpdateSkill(ctx context.Context, skill *domain.Skill) error {
	skill.UpdatedAt = time.Now().UTC()

	result, err := r.db.Exec(ctx,
		`UPDATE skills SET name = $1, updated_at = $2 WHERE id = $3 AND user_id = $4`,
		skill.Name, skill.UpdatedAt, skill.ID, skill.UserID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("skill not found")
	}

	return nil
}

// ---------------------------------------------------------------------------
// Learning exercises
// ---------------------------------------------------------------------------

func learningExerciseSelect() squirrel.SelectBuilder {
	return psql().
		Select(
			"e.id", "e.user_id", "e.skill_id", "e.kind", "e.content", "e.planned_for", "e.answer", "e.answered_at",
			"e.score::int", "e.note", "e.reviewed_at", "e.created_at", "s.name",
		).
		From("learning_exercises e").
		Join("skills s ON s.id = e.skill_id")
}

var exerciseStateConditions = map[domain.LearningExerciseState]squirrel.Sqlizer{
	domain.LearningExercisePlanned:  squirrel.Expr("e.answer IS NULL"),
	domain.LearningExerciseAnswered: squirrel.Expr("(e.answer IS NOT NULL AND e.score IS NULL)"),
	domain.LearningExerciseReviewed: squirrel.Expr("e.score IS NOT NULL"),
}

func applyLearningExerciseFilter(query squirrel.SelectBuilder, filter domain.LearningExerciseFilter) squirrel.SelectBuilder {
	query = query.Where(squirrel.Eq{"e.user_id": filter.UserID})

	if filter.LearningPlanID != nil {
		query = query.Where(squirrel.Eq{"s.learning_plan_id": *filter.LearningPlanID})
	}
	if filter.SkillID != nil {
		query = query.Where(squirrel.Eq{"e.skill_id": *filter.SkillID})
	}
	if len(filter.States) > 0 {
		var or squirrel.Or
		for _, state := range filter.States {
			if cond, ok := exerciseStateConditions[state]; ok {
				or = append(or, cond)
			}
		}
		query = query.Where(or)
	}
	if filter.DueAt != nil {
		query = query.Where(squirrel.LtOrEq{"e.planned_for": *filter.DueAt})
	}

	return query
}

func (r *repository) queryLearningExercises(ctx context.Context, query squirrel.SelectBuilder) ([]domain.LearningExercise, error) {
	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query learning exercises: %w", err)
	}
	defer rows.Close()

	var exercises []domain.LearningExercise
	for rows.Next() {
		var e domain.LearningExercise
		err := rows.Scan(
			&e.ID, &e.UserID, &e.SkillID, &e.Kind, &e.Content, &e.PlannedFor, &e.Answer, &e.AnsweredAt,
			&e.Score, &e.Note, &e.ReviewedAt, &e.CreatedAt, &e.SkillName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan learning exercise: %w", err)
		}
		exercises = append(exercises, e)
	}

	return exercises, rows.Err()
}

func (r *repository) CreateLearningExercises(ctx context.Context, exercises []domain.LearningExercise) ([]int64, error) {
	now := time.Now().UTC()
	ids := make([]int64, 0, len(exercises))

	err := r.inTx(ctx, func(tx pgx.Tx) error {
		for _, e := range exercises {
			var id int64
			err := tx.QueryRow(ctx, `
				INSERT INTO learning_exercises (user_id, skill_id, kind, content, planned_for, created_at)
				VALUES ($1, $2, $3, $4, $5, $6)
				RETURNING id`,
				e.UserID, e.SkillID, e.Kind, e.Content, e.PlannedFor, now,
			).Scan(&id)
			if err != nil {
				return err
			}
			ids = append(ids, id)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func (r *repository) GetLearningExercise(ctx context.Context, exerciseID int64, userID int64) (*domain.LearningExercise, error) {
	exercises, err := r.queryLearningExercises(ctx, learningExerciseSelect().Where(squirrel.Eq{"e.id": exerciseID, "e.user_id": userID}))
	if err != nil || len(exercises) == 0 {
		return nil, err
	}
	return &exercises[0], nil
}

func (r *repository) ListLearningExercises(ctx context.Context, filter domain.LearningExerciseFilter) ([]domain.LearningExercise, error) {
	query := applyLearningExerciseFilter(learningExerciseSelect(), filter)

	if filter.OldestFirst {
		query = query.OrderBy("e.planned_for ASC", "e.id ASC")
	} else {
		query = query.OrderBy("e.planned_for DESC", "e.id DESC")
	}
	if filter.Limit > 0 {
		query = query.Limit(uint64(filter.Limit))
	}
	if filter.Offset > 0 {
		query = query.Offset(uint64(filter.Offset))
	}

	return r.queryLearningExercises(ctx, query)
}

func (r *repository) CountLearningExercises(ctx context.Context, filter domain.LearningExerciseFilter) (int, error) {
	query := applyLearningExerciseFilter(
		psql().Select("e.id").From("learning_exercises e").Join("skills s ON s.id = e.skill_id"),
		filter,
	)
	return r.countRows(ctx, query)
}

func (r *repository) GetLearningExerciseStats(ctx context.Context, userID int64, planIDs []int64, dueAt time.Time) ([]domain.LearningExerciseStats, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.learning_plan_id,
		       COUNT(*) FILTER (WHERE e.answer IS NULL),
		       COUNT(*) FILTER (WHERE e.answer IS NULL AND e.planned_for <= $3),
		       COUNT(*) FILTER (WHERE e.answer IS NOT NULL AND e.score IS NULL)
		FROM learning_exercises e
		JOIN skills s ON s.id = e.skill_id
		WHERE e.user_id = $1 AND s.learning_plan_id = ANY($2)
		GROUP BY s.learning_plan_id`,
		userID, planIDs, dueAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query exercise stats: %w", err)
	}
	defer rows.Close()

	var stats []domain.LearningExerciseStats
	for rows.Next() {
		var st domain.LearningExerciseStats
		if err := rows.Scan(&st.LearningPlanID, &st.Planned, &st.Due, &st.AwaitingReview); err != nil {
			return nil, fmt.Errorf("failed to scan exercise stats: %w", err)
		}
		stats = append(stats, st)
	}

	return stats, rows.Err()
}

func (r *repository) UpdateLearningExercise(ctx context.Context, exercise *domain.LearningExercise) error {
	result, err := r.db.Exec(ctx, `
		UPDATE learning_exercises
		SET kind = $1, content = $2, planned_for = $3
		WHERE id = $4 AND user_id = $5`,
		exercise.Kind, exercise.Content, exercise.PlannedFor, exercise.ID, exercise.UserID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("learning exercise not found")
	}

	return nil
}

func (r *repository) SubmitLearningExerciseAnswer(ctx context.Context, exerciseID int64, userID int64, answer string, answeredAt time.Time) error {
	result, err := r.db.Exec(ctx, `
		UPDATE learning_exercises
		SET answer = $1, answered_at = $2
		WHERE id = $3 AND user_id = $4 AND answer IS NULL`,
		answer, answeredAt, exerciseID, userID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("learning exercise not found or already answered")
	}

	return nil
}

func (r *repository) ReviewLearningExercises(ctx context.Context, userID int64, reviews []domain.LearningExerciseReview) error {
	return r.inTx(ctx, func(tx pgx.Tx) error {
		for _, review := range reviews {
			result, err := tx.Exec(ctx, `
				UPDATE learning_exercises
				SET score = $1, note = $2, reviewed_at = $3
				WHERE id = $4 AND user_id = $5 AND answer IS NOT NULL`,
				review.Score, review.Note, review.ReviewedAt, review.ExerciseID, userID,
			)
			if err != nil {
				return err
			}
			if result.RowsAffected() == 0 {
				return fmt.Errorf("learning exercise %d not found or not answered", review.ExerciseID)
			}
		}
		return nil
	})
}

func (r *repository) DeleteLearningExercise(ctx context.Context, exerciseID int64, userID int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM learning_exercises WHERE id = $1 AND user_id = $2`, exerciseID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("learning exercise not found")
	}

	return nil
}

// ---------------------------------------------------------------------------
// Vocabulary
// ---------------------------------------------------------------------------

// vocabularySelect selects words with the read-only fields computed from
// their reviews; callers add WHERE clauses on "v".
func vocabularySelect() squirrel.SelectBuilder {
	return psql().
		Select(
			"v.id", "v.user_id", "v.learning_plan_id", "v.skill_id", "v.term", "v.translation", "v.notes",
			"v.created_at", "v.updated_at",
			"COUNT(r.id) FILTER (WHERE r.score IS NOT NULL) AS review_count",
			"MAX(r.reviewed_at) AS last_reviewed_at",
			"(ARRAY_AGG(r.score::int ORDER BY r.reviewed_at DESC, r.id DESC) FILTER (WHERE r.score IS NOT NULL))[1] AS last_score",
			"MIN(r.planned_for) FILTER (WHERE r.score IS NULL) AS next_planned_for",
			"ARRAY_AGG(r.score::int ORDER BY r.reviewed_at, r.id) FILTER (WHERE r.score IS NOT NULL) AS scores",
		).
		From("vocabulary v").
		LeftJoin("vocabulary_reviews r ON r.vocabulary_id = v.id").
		GroupBy("v.id")
}

func applyVocabularyFilter(query squirrel.SelectBuilder, filter domain.VocabularyFilter) squirrel.SelectBuilder {
	query = query.Where(squirrel.Eq{"v.user_id": filter.UserID})

	if filter.LearningPlanID != nil {
		query = query.Where(squirrel.Eq{"v.learning_plan_id": *filter.LearningPlanID})
	}
	if len(filter.IDs) > 0 {
		query = query.Where(squirrel.Eq{"v.id": filter.IDs})
	}
	if filter.Unscheduled {
		query = query.Having("COUNT(r.id) FILTER (WHERE r.score IS NULL) = 0")
	}

	return query
}

func (r *repository) queryVocabulary(ctx context.Context, query squirrel.SelectBuilder) ([]domain.VocabularyWord, error) {
	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query vocabulary: %w", err)
	}
	defer rows.Close()

	var words []domain.VocabularyWord
	for rows.Next() {
		var w domain.VocabularyWord
		err := rows.Scan(
			&w.ID, &w.UserID, &w.LearningPlanID, &w.SkillID, &w.Term, &w.Translation, &w.Notes,
			&w.CreatedAt, &w.UpdatedAt,
			&w.ReviewCount, &w.LastReviewedAt, &w.LastScore, &w.NextPlannedFor, &w.Scores,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan vocabulary word: %w", err)
		}
		words = append(words, w)
	}

	return words, rows.Err()
}

func (r *repository) CreateVocabulary(ctx context.Context, words []domain.VocabularyWord) ([]int64, error) {
	now := time.Now().UTC()
	ids := make([]int64, 0, len(words))

	err := r.inTx(ctx, func(tx pgx.Tx) error {
		for _, w := range words {
			createdAt := w.CreatedAt
			if createdAt.IsZero() {
				createdAt = now
			}
			var id int64
			err := tx.QueryRow(ctx, `
				INSERT INTO vocabulary (user_id, learning_plan_id, skill_id, term, translation, notes, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
				RETURNING id`,
				w.UserID, w.LearningPlanID, w.SkillID, w.Term, w.Translation, w.Notes, createdAt,
			).Scan(&id)
			if err != nil {
				return err
			}
			ids = append(ids, id)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func (r *repository) ListVocabulary(ctx context.Context, filter domain.VocabularyFilter) ([]domain.VocabularyWord, error) {
	query := applyVocabularyFilter(vocabularySelect(), filter)

	if filter.LeastRecentlyFirst {
		query = query.OrderBy("MAX(r.reviewed_at) ASC NULLS FIRST", "v.id ASC")
	} else {
		query = query.OrderBy("v.created_at DESC", "v.id DESC")
	}
	if filter.Limit > 0 {
		query = query.Limit(uint64(filter.Limit))
	}
	if filter.Offset > 0 {
		query = query.Offset(uint64(filter.Offset))
	}

	return r.queryVocabulary(ctx, query)
}

func (r *repository) CountVocabulary(ctx context.Context, filter domain.VocabularyFilter) (int, error) {
	query := applyVocabularyFilter(
		psql().Select("v.id").From("vocabulary v").LeftJoin("vocabulary_reviews r ON r.vocabulary_id = v.id").GroupBy("v.id"),
		filter,
	)
	return r.countRows(ctx, query)
}

func (r *repository) SearchVocabulary(ctx context.Context, filter domain.VocabularySearchFilter) ([]domain.VocabularyWord, error) {
	pattern := "%" + filter.Query + "%"
	query := vocabularySelect().
		Where(squirrel.Eq{"v.user_id": filter.UserID}).
		Where(squirrel.Or{
			squirrel.ILike{"v.term": pattern},
			squirrel.ILike{"v.translation": pattern},
			squirrel.ILike{"v.notes": pattern},
		}).
		OrderBy("v.created_at DESC", "v.id DESC")

	if filter.LearningPlanID != nil {
		query = query.Where(squirrel.Eq{"v.learning_plan_id": *filter.LearningPlanID})
	}

	return r.queryVocabulary(ctx, query)
}

func (r *repository) FindVocabularyTerms(ctx context.Context, planID int64, userID int64, terms []string) ([]string, error) {
	lowered := make([]string, len(terms))
	for i, t := range terms {
		lowered[i] = strings.ToLower(t)
	}

	rows, err := r.db.Query(ctx, `
		SELECT term FROM vocabulary
		WHERE learning_plan_id = $1 AND user_id = $2 AND LOWER(term) = ANY($3)
		ORDER BY id`,
		planID, userID, lowered,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query vocabulary terms: %w", err)
	}
	defer rows.Close()

	var found []string
	for rows.Next() {
		var term string
		if err := rows.Scan(&term); err != nil {
			return nil, err
		}
		found = append(found, term)
	}

	return found, rows.Err()
}

func (r *repository) CountDueVocabularyReviews(ctx context.Context, userID int64, planIDs []int64, dueAt time.Time) (map[int64]int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT v.learning_plan_id, COUNT(*)
		FROM vocabulary_reviews r
		JOIN vocabulary v ON v.id = r.vocabulary_id
		WHERE r.user_id = $1 AND v.learning_plan_id = ANY($2) AND r.score IS NULL AND r.planned_for <= $3
		GROUP BY v.learning_plan_id`,
		userID, planIDs, dueAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to count due reviews: %w", err)
	}
	defer rows.Close()

	counts := make(map[int64]int)
	for rows.Next() {
		var planID int64
		var count int
		if err := rows.Scan(&planID, &count); err != nil {
			return nil, err
		}
		counts[planID] = count
	}

	return counts, rows.Err()
}

func (r *repository) UpdateVocabulary(ctx context.Context, word *domain.VocabularyWord) error {
	word.UpdatedAt = time.Now().UTC()

	result, err := r.db.Exec(ctx, `
		UPDATE vocabulary
		SET term = $1, translation = $2, notes = $3, skill_id = $4, updated_at = $5
		WHERE id = $6 AND user_id = $7`,
		word.Term, word.Translation, word.Notes, word.SkillID, word.UpdatedAt, word.ID, word.UserID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("vocabulary word not found")
	}

	return nil
}

func (r *repository) DeleteVocabulary(ctx context.Context, wordID int64, userID int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM vocabulary WHERE id = $1 AND user_id = $2`, wordID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("vocabulary word not found")
	}

	return nil
}

// ---------------------------------------------------------------------------
// Vocabulary reviews
// ---------------------------------------------------------------------------

func applyVocabularyReviewFilter(query squirrel.SelectBuilder, filter domain.VocabularyReviewFilter) squirrel.SelectBuilder {
	query = query.Where(squirrel.Eq{"r.user_id": filter.UserID})

	if filter.LearningPlanID != nil {
		query = query.Where(squirrel.Eq{"v.learning_plan_id": *filter.LearningPlanID})
	}
	if filter.VocabularyID != nil {
		query = query.Where(squirrel.Eq{"r.vocabulary_id": *filter.VocabularyID})
	}
	if len(filter.IDs) > 0 {
		query = query.Where(squirrel.Eq{"r.id": filter.IDs})
	}
	if filter.Pending != nil {
		if *filter.Pending {
			query = query.Where("r.score IS NULL")
		} else {
			query = query.Where("r.score IS NOT NULL")
		}
	}
	if filter.DueAt != nil {
		query = query.Where(squirrel.LtOrEq{"r.planned_for": *filter.DueAt})
	}
	if filter.ReviewedFrom != nil {
		query = query.Where(squirrel.GtOrEq{"r.reviewed_at": *filter.ReviewedFrom})
	}
	if filter.ReviewedTo != nil {
		query = query.Where(squirrel.Lt{"r.reviewed_at": *filter.ReviewedTo})
	}

	return query
}

func (r *repository) CreateVocabularyReviews(ctx context.Context, reviews []domain.VocabularyReview) ([]int64, error) {
	now := time.Now().UTC()
	ids := make([]int64, 0, len(reviews))

	err := r.inTx(ctx, func(tx pgx.Tx) error {
		for _, rv := range reviews {
			var id int64
			err := tx.QueryRow(ctx, `
				INSERT INTO vocabulary_reviews (user_id, vocabulary_id, planned_for, score, reviewed_at, created_at)
				VALUES ($1, $2, $3, $4, $5, $6)
				RETURNING id`,
				rv.UserID, rv.VocabularyID, rv.PlannedFor, rv.Score, rv.ReviewedAt, now,
			).Scan(&id)
			if err != nil {
				return err
			}
			ids = append(ids, id)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func (r *repository) ListVocabularyReviews(ctx context.Context, filter domain.VocabularyReviewFilter) ([]domain.VocabularyReview, error) {
	query := applyVocabularyReviewFilter(
		psql().
			Select(
				"r.id", "r.user_id", "r.vocabulary_id", "r.planned_for", "r.score::int", "r.reviewed_at", "r.created_at",
				"v.term", "v.translation", "v.notes",
			).
			From("vocabulary_reviews r").
			Join("vocabulary v ON v.id = r.vocabulary_id"),
		filter,
	)

	if filter.DueAt != nil || (filter.Pending != nil && *filter.Pending) {
		query = query.OrderBy("r.planned_for ASC", "r.id ASC")
	} else {
		query = query.OrderBy("r.reviewed_at DESC NULLS LAST", "r.planned_for ASC", "r.id DESC")
	}
	if filter.Limit > 0 {
		query = query.Limit(uint64(filter.Limit))
	}
	if filter.Offset > 0 {
		query = query.Offset(uint64(filter.Offset))
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query vocabulary reviews: %w", err)
	}
	defer rows.Close()

	var reviews []domain.VocabularyReview
	for rows.Next() {
		var rv domain.VocabularyReview
		err := rows.Scan(
			&rv.ID, &rv.UserID, &rv.VocabularyID, &rv.PlannedFor, &rv.Score, &rv.ReviewedAt, &rv.CreatedAt,
			&rv.Term, &rv.Translation, &rv.Notes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan vocabulary review: %w", err)
		}
		reviews = append(reviews, rv)
	}

	return reviews, rows.Err()
}

func (r *repository) CountVocabularyReviews(ctx context.Context, filter domain.VocabularyReviewFilter) (int, error) {
	query := applyVocabularyReviewFilter(
		psql().Select("r.id").From("vocabulary_reviews r").Join("vocabulary v ON v.id = r.vocabulary_id"),
		filter,
	)
	return r.countRows(ctx, query)
}

func (r *repository) RecordVocabularyReview(ctx context.Context, reviewID int64, userID int64, score int, reviewedAt time.Time) error {
	result, err := r.db.Exec(ctx, `
		UPDATE vocabulary_reviews
		SET score = $1, reviewed_at = $2
		WHERE id = $3 AND user_id = $4 AND score IS NULL`,
		score, reviewedAt, reviewID, userID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("pending vocabulary review not found")
	}

	return nil
}

func (r *repository) DeleteVocabularyReviews(ctx context.Context, userID int64, reviewIDs []int64) error {
	return r.inTx(ctx, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx,
			`DELETE FROM vocabulary_reviews WHERE id = ANY($1) AND user_id = $2 AND score IS NULL`,
			reviewIDs, userID,
		)
		if err != nil {
			return err
		}
		if int(result.RowsAffected()) != len(reviewIDs) {
			return fmt.Errorf("some reviews are not pending or not found")
		}
		return nil
	})
}
