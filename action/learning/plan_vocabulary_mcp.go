package learning

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/util"
)

var PlanVocabularyMCPDefinition = mcp.Tool{
	Name:        "plan_vocabulary",
	Annotations: &mcp.ToolAnnotations{Title: "Plan word reviews"},
	Description: `Plan word reviews for the coming days, both first contact and repetition. The user reviews them as flashcards on the web. All or nothing.

Each entry:
- word_id: required
- review_dates: required, 1+ days YYYY-MM-DD (Asia/Nicosia), today or later; one pending review per date

A word that already has a pending review on that day fails the call. Replan missed reviews with cancel_vocabulary_reviews first.`,
}

type PlanVocabularyEntry struct {
	WordID      int64    `json:"word_id" jsonschema:"Word ID"`
	ReviewDates []string `json:"review_dates" jsonschema:"Days YYYY-MM-DD, today or later"`
}

type PlanVocabularyInput struct {
	Entries []PlanVocabularyEntry `json:"entries" jsonschema:"1-100 entries"`
}

type PlanVocabularyOutput struct {
	Reviews []ReviewResult `json:"reviews" jsonschema:"Created pending reviews"`
}

func PlanVocabulary(ctx context.Context, _ *mcp.CallToolRequest, input PlanVocabularyInput) (*mcp.CallToolResult, PlanVocabularyOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, PlanVocabularyOutput{}, err
	}

	if len(input.Entries) == 0 || len(input.Entries) > maxReviewBatch {
		return nil, PlanVocabularyOutput{}, fmt.Errorf("entries must have 1-%d entries", maxReviewBatch)
	}

	now := time.Now()
	wordIDs := make([]int64, 0, len(input.Entries))
	for _, e := range input.Entries {
		wordIDs = append(wordIDs, e.WordID)
	}
	words, err := loadWords(ctx, db, userID, wordIDs, now)
	if err != nil {
		return nil, PlanVocabularyOutput{}, err
	}

	// pendingDays holds every day a word has a pending review on, already stored or planned in this call.
	pendingDays := make(map[int64]map[string]bool)
	var reviews []domain.VocabularyReview
	for _, e := range input.Entries {
		if len(e.ReviewDates) == 0 {
			return nil, PlanVocabularyOutput{}, fmt.Errorf("word %d: review_dates cannot be empty", e.WordID)
		}

		days, ok := pendingDays[e.WordID]
		if !ok {
			days = make(map[string]bool)
			pending, err := db.ListVocabularyReviews(ctx, domain.VocabularyReviewFilter{
				UserID:       userID,
				VocabularyID: util.Ptr(e.WordID),
				Pending:      util.Ptr(true),
			})
			if err != nil {
				return nil, PlanVocabularyOutput{}, fmt.Errorf("database error: %w", err)
			}
			for _, r := range pending {
				days[formatDay(r.PlannedFor)] = true
			}
			pendingDays[e.WordID] = days
		}

		for _, raw := range e.ReviewDates {
			day, err := parseFutureDay("review date", raw, now)
			if err != nil {
				return nil, PlanVocabularyOutput{}, fmt.Errorf("word %d: %w", e.WordID, err)
			}
			key := formatDay(day)
			if days[key] {
				return nil, PlanVocabularyOutput{}, fmt.Errorf("word %d (%s) already has a pending review on %s", e.WordID, words[e.WordID].Term, key)
			}
			days[key] = true
			reviews = append(reviews, domain.VocabularyReview{UserID: userID, VocabularyID: e.WordID, PlannedFor: day})
		}
	}

	ids, err := db.CreateVocabularyReviews(ctx, reviews)
	if err != nil {
		return nil, PlanVocabularyOutput{}, fmt.Errorf("failed to plan reviews: %w", err)
	}

	created, err := db.ListVocabularyReviews(ctx, domain.VocabularyReviewFilter{UserID: userID, IDs: ids, Pending: util.Ptr(true)})
	if err != nil {
		return nil, PlanVocabularyOutput{}, fmt.Errorf("failed to load planned reviews: %w", err)
	}

	return nil, PlanVocabularyOutput{Reviews: reviewsToResults(created)}, nil
}
