// Package learning tracks long-running study: plans, skills, exercises done
// in chat, and vocabulary reviewed on the web (see
// docs/functions/learning-spec.md).
package learning

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"personal/domain"
	"personal/gateways"
)

const (
	// displayTimezone is where planned days start: they are stored as its midnight.
	displayTimezone = "Asia/Nicosia"

	maxExerciseBatch = 50
	maxVocabularyAdd = 50
	maxReviewBatch   = 100
	maxKindLength    = 100
	maxTextLength    = 255

	defaultExerciseLimit = 20
	defaultWordLimit     = 50
	defaultReviewLimit   = 100
)

func location() *time.Location {
	loc, err := time.LoadLocation(displayTimezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

// today is midnight of the current day in the display timezone.
func today(now time.Time) time.Time {
	y, m, d := now.In(location()).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, location())
}

// parseDay parses YYYY-MM-DD as midnight of that day in the display timezone.
func parseDay(field, raw string) (time.Time, error) {
	day, err := time.ParseInLocation(dayLayout, raw, location())
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid %s %q, expected YYYY-MM-DD", field, raw)
	}
	return day, nil
}

// parseFutureDay parses a planned day that cannot be in the past.
func parseFutureDay(field, raw string, now time.Time) (time.Time, error) {
	day, err := parseDay(field, raw)
	if err != nil {
		return time.Time{}, err
	}
	if day.Before(today(now)) {
		return time.Time{}, fmt.Errorf("%s %s is in the past", field, raw)
	}
	return day, nil
}

// parseInstant accepts RFC3339 or a YYYY-MM-DD day.
func parseInstant(field, raw string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	return parseDay(field, raw)
}

func dbAndUser(ctx context.Context) (gateways.DB, int64, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, 0, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, 0, fmt.Errorf("user_id not available in context")
	}
	return db, userID, nil
}

func validScore(score int) bool {
	return score >= 1 && score <= 5
}

// optionalText turns an empty or blank string into nil.
func optionalText(s *string) *string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	return s
}

func loadPlan(ctx context.Context, db gateways.DB, userID, planID int64) (*domain.LearningPlan, error) {
	plan, err := db.GetLearningPlan(ctx, planID, userID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("learning plan %d not found", planID)
	}
	return plan, nil
}

func loadSkill(ctx context.Context, db gateways.DB, userID, skillID int64) (*domain.Skill, error) {
	skill, err := db.GetSkill(ctx, skillID, userID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if skill == nil {
		return nil, fmt.Errorf("skill %d not found", skillID)
	}
	return skill, nil
}

func loadExercise(ctx context.Context, db gateways.DB, userID, exerciseID int64) (*domain.LearningExercise, error) {
	exercise, err := db.GetLearningExercise(ctx, exerciseID, userID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if exercise == nil {
		return nil, fmt.Errorf("learning exercise %d not found", exerciseID)
	}
	return exercise, nil
}

// loadWords loads the user's words by id with mastery computed; missing ids are an error.
func loadWords(ctx context.Context, db gateways.DB, userID int64, ids []int64, now time.Time) (map[int64]domain.VocabularyWord, error) {
	words, err := db.ListVocabulary(ctx, domain.VocabularyFilter{UserID: userID, IDs: ids})
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	byID := make(map[int64]domain.VocabularyWord, len(words))
	for _, w := range withWordMastery(words, now) {
		byID[w.ID] = w
	}
	for _, id := range ids {
		if _, ok := byID[id]; !ok {
			return nil, fmt.Errorf("vocabulary word %d not found", id)
		}
	}
	return byID, nil
}

func loadWord(ctx context.Context, db gateways.DB, userID, wordID int64, now time.Time) (*domain.VocabularyWord, error) {
	words, err := loadWords(ctx, db, userID, []int64{wordID}, now)
	if err != nil {
		return nil, err
	}
	w := words[wordID]
	return &w, nil
}

// checkSkillInPlan fails unless the skill is the user's and belongs to the plan.
func checkSkillInPlan(ctx context.Context, db gateways.DB, userID, planID, skillID int64) error {
	skill, err := db.GetSkill(ctx, skillID, userID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if skill == nil || skill.LearningPlanID != planID {
		return fmt.Errorf("skill %d not found in this plan", skillID)
	}
	return nil
}

func withSkillMastery(skills []domain.Skill, now time.Time) []domain.Skill {
	for i := range skills {
		skills[i].MasteryState = domain.ComputeMastery(skills[i].Scores, skills[i].LastPracticedAt, now)
	}
	return skills
}

func withWordMastery(words []domain.VocabularyWord, now time.Time) []domain.VocabularyWord {
	for i := range words {
		words[i].MasteryState = domain.ComputeMastery(words[i].Scores, words[i].LastReviewedAt, now)
	}
	return words
}

// exerciseStats returns each plan's exercise stats; plans without exercises get zeros.
func exerciseStats(ctx context.Context, db gateways.DB, userID int64, planIDs []int64, now time.Time) (map[int64]ExerciseStatsResult, error) {
	stats, err := db.GetLearningExerciseStats(ctx, userID, planIDs, now)
	if err != nil {
		return nil, fmt.Errorf("failed to load exercise stats: %w", err)
	}
	byPlan := make(map[int64]ExerciseStatsResult, len(planIDs))
	for _, st := range stats {
		byPlan[st.LearningPlanID] = ExerciseStatsResult{Planned: st.Planned, Due: st.Due, AwaitingReview: st.AwaitingReview}
	}
	return byPlan, nil
}

// vocabularyStats computes each plan's vocabulary stats from its words.
func vocabularyStats(ctx context.Context, db gateways.DB, userID int64, planIDs []int64, now time.Time) (map[int64]VocabularyStatsResult, error) {
	due, err := db.CountDueVocabularyReviews(ctx, userID, planIDs, now)
	if err != nil {
		return nil, fmt.Errorf("failed to count due reviews: %w", err)
	}

	byPlan := make(map[int64]VocabularyStatsResult, len(planIDs))
	for _, planID := range planIDs {
		words, err := db.ListVocabulary(ctx, domain.VocabularyFilter{UserID: userID, LearningPlanID: &planID})
		if err != nil {
			return nil, fmt.Errorf("failed to load vocabulary: %w", err)
		}
		st := VocabularyStatsResult{Total: len(words), Due: due[planID]}
		sum := 0
		for _, w := range withWordMastery(words, now) {
			if w.NextPlannedFor == nil {
				st.Unscheduled++
			}
			if w.EffectiveMastery >= domain.MasteredThreshold {
				st.Mastered++
			}
			sum += w.EffectiveMastery
		}
		if len(words) > 0 {
			st.AverageMastery = (sum + len(words)/2) / len(words)
		}
		byPlan[planID] = st
	}
	return byPlan, nil
}

// --- vocabulary writes shared by the MCP tools and the web pages -----------

// wordInput is one word to add.
type wordInput struct {
	Term        string
	Translation string
	Notes       *string
	SkillID     *int64
}

// addVocabulary validates and adds words to a plan, all or nothing; shared by
// add_vocabulary and POST /web/learning/:id/vocabulary.
func addVocabulary(ctx context.Context, db gateways.DB, userID, planID int64, inputs []wordInput, now time.Time) ([]domain.VocabularyWord, error) {
	if len(inputs) == 0 || len(inputs) > maxVocabularyAdd {
		return nil, fmt.Errorf("words must have 1-%d entries", maxVocabularyAdd)
	}
	if _, err := loadPlan(ctx, db, userID, planID); err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var repeated, terms []string
	words := make([]domain.VocabularyWord, len(inputs))
	for i, in := range inputs {
		term := strings.TrimSpace(in.Term)
		translation := strings.TrimSpace(in.Translation)
		if term == "" {
			return nil, fmt.Errorf("term is required")
		}
		if translation == "" {
			return nil, fmt.Errorf("translation is required for %q", term)
		}
		if len([]rune(term)) > maxTextLength || len([]rune(translation)) > maxTextLength {
			return nil, fmt.Errorf("term and translation must be at most %d characters", maxTextLength)
		}
		if in.SkillID != nil {
			if err := checkSkillInPlan(ctx, db, userID, planID, *in.SkillID); err != nil {
				return nil, err
			}
		}
		key := strings.ToLower(term)
		if seen[key] {
			repeated = append(repeated, term)
		}
		seen[key] = true
		terms = append(terms, term)
		words[i] = domain.VocabularyWord{
			UserID:         userID,
			LearningPlanID: planID,
			SkillID:        in.SkillID,
			Term:           term,
			Translation:    translation,
			Notes:          optionalText(in.Notes),
		}
	}
	if len(repeated) > 0 {
		return nil, fmt.Errorf("terms repeated in the call: %s", strings.Join(repeated, ", "))
	}

	existing, err := db.FindVocabularyTerms(ctx, planID, userID, terms)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if len(existing) > 0 {
		return nil, fmt.Errorf("terms already in the plan: %s", strings.Join(existing, ", "))
	}

	ids, err := db.CreateVocabulary(ctx, words)
	if err != nil {
		return nil, fmt.Errorf("failed to add vocabulary: %w", err)
	}

	created, err := loadWords(ctx, db, userID, ids, now)
	if err != nil {
		return nil, err
	}
	result := make([]domain.VocabularyWord, len(ids))
	for i, id := range ids {
		result[i] = created[id]
	}
	return result, nil
}

// wordChanges is a partial update of a word; notes "" and skill 0 clear them.
type wordChanges struct {
	Term        *string
	Translation *string
	Notes       *string
	SkillID     *int64
}

// editVocabulary validates and applies a partial update; shared by
// edit_vocabulary and POST /web/learning/:id/vocabulary/:word_id.
func editVocabulary(ctx context.Context, db gateways.DB, userID int64, word *domain.VocabularyWord, ch wordChanges, now time.Time) (*domain.VocabularyWord, error) {
	if ch.Term == nil && ch.Translation == nil && ch.Notes == nil && ch.SkillID == nil {
		return nil, fmt.Errorf("at least one of term, translation, notes or skill_id is required")
	}

	if ch.Term != nil {
		term := strings.TrimSpace(*ch.Term)
		if term == "" {
			return nil, fmt.Errorf("term cannot be empty")
		}
		if len([]rune(term)) > maxTextLength {
			return nil, fmt.Errorf("term must be at most %d characters", maxTextLength)
		}
		if !strings.EqualFold(term, word.Term) {
			existing, err := db.FindVocabularyTerms(ctx, word.LearningPlanID, userID, []string{term})
			if err != nil {
				return nil, fmt.Errorf("database error: %w", err)
			}
			if len(existing) > 0 {
				return nil, fmt.Errorf("terms already in the plan: %s", strings.Join(existing, ", "))
			}
		}
		word.Term = term
	}
	if ch.Translation != nil {
		translation := strings.TrimSpace(*ch.Translation)
		if translation == "" {
			return nil, fmt.Errorf("translation cannot be empty")
		}
		if len([]rune(translation)) > maxTextLength {
			return nil, fmt.Errorf("translation must be at most %d characters", maxTextLength)
		}
		word.Translation = translation
	}
	if ch.Notes != nil {
		word.Notes = optionalText(ch.Notes)
	}
	if ch.SkillID != nil {
		if *ch.SkillID == 0 {
			word.SkillID = nil
		} else {
			if err := checkSkillInPlan(ctx, db, userID, word.LearningPlanID, *ch.SkillID); err != nil {
				return nil, err
			}
			word.SkillID = ch.SkillID
		}
	}

	if err := db.UpdateVocabulary(ctx, word); err != nil {
		return nil, fmt.Errorf("failed to update word: %w", err)
	}
	return loadWord(ctx, db, userID, word.ID, now)
}

// wordMatch is one word found by searchVocabulary with the number of variants that matched it.
type wordMatch struct {
	Word       domain.VocabularyWord
	MatchCount int
}

func validateQueryVariants(variants []string) error {
	if len(variants) == 0 {
		return fmt.Errorf("query_variants cannot be empty")
	}
	if len(variants) > 5 {
		return fmt.Errorf("maximum 5 query variants allowed")
	}
	for _, v := range variants {
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("query variants cannot be empty strings")
		}
	}
	return nil
}

// searchVocabulary runs one ILIKE search per variant and merges the results,
// ranked by match count DESC, then created_at DESC; shared by
// search_vocabulary and GET /web/learning/:id/vocabulary?q=.
func searchVocabulary(ctx context.Context, db gateways.DB, userID int64, variants []string, planID *int64, now time.Time) ([]wordMatch, error) {
	matches := make(map[int64]*wordMatch)
	for _, variant := range variants {
		words, err := db.SearchVocabulary(ctx, domain.VocabularySearchFilter{UserID: userID, Query: variant, LearningPlanID: planID})
		if err != nil {
			return nil, fmt.Errorf("search failed: %w", err)
		}
		for _, w := range words {
			if m, ok := matches[w.ID]; ok {
				m.MatchCount++
			} else {
				matches[w.ID] = &wordMatch{Word: w, MatchCount: 1}
			}
		}
	}

	results := make([]wordMatch, 0, len(matches))
	for _, m := range matches {
		m.Word.MasteryState = domain.ComputeMastery(m.Word.Scores, m.Word.LastReviewedAt, now)
		results = append(results, *m)
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].MatchCount != results[j].MatchCount {
			return results[i].MatchCount > results[j].MatchCount
		}
		a, b := results[i].Word, results[j].Word
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.After(b.CreatedAt)
		}
		return a.ID > b.ID
	})
	return results, nil
}
