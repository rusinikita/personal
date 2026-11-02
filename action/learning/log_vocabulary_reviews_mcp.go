package learning

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/util"
)

var LogVocabularyReviewsMCPDefinition = mcp.Tool{
	Name:        "log_vocabulary_reviews",
	Annotations: &mcp.ToolAnnotations{Title: "Log indirect word reviews"},
	Description: `Record indirect reviews found at the weekly analysis: words the user used in an exercise answer. All or nothing.
Each entry becomes a scored review dated today and moves the word's mastery like a flashcard. Pending reviews of those words are untouched.

Each entry:
- word_id: required
- score: required, 1-5, by how well the word was used`,
}

type LogVocabularyReviewEntry struct {
	WordID int64 `json:"word_id" jsonschema:"Word ID"`
	Score  int   `json:"score" jsonschema:"1-5"`
}

type LogVocabularyReviewsInput struct {
	Entries []LogVocabularyReviewEntry `json:"entries" jsonschema:"1-100 entries"`
}

type LogVocabularyReviewsOutput struct {
	Reviews []ReviewResult `json:"reviews" jsonschema:"Created scored reviews"`
}

func LogVocabularyReviews(ctx context.Context, _ *mcp.CallToolRequest, input LogVocabularyReviewsInput) (*mcp.CallToolResult, LogVocabularyReviewsOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, LogVocabularyReviewsOutput{}, err
	}

	if len(input.Entries) == 0 || len(input.Entries) > maxReviewBatch {
		return nil, LogVocabularyReviewsOutput{}, fmt.Errorf("entries must have 1-%d entries", maxReviewBatch)
	}

	now := time.Now()
	wordIDs := make([]int64, len(input.Entries))
	for i, e := range input.Entries {
		if !validScore(e.Score) {
			return nil, LogVocabularyReviewsOutput{}, fmt.Errorf("word %d: score must be 1-5", e.WordID)
		}
		wordIDs[i] = e.WordID
	}
	if _, err := loadWords(ctx, db, userID, wordIDs, now); err != nil {
		return nil, LogVocabularyReviewsOutput{}, err
	}

	reviewedAt := now.UTC()
	reviews := make([]domain.VocabularyReview, len(input.Entries))
	for i, e := range input.Entries {
		reviews[i] = domain.VocabularyReview{
			UserID:       userID,
			VocabularyID: e.WordID,
			PlannedFor:   today(now),
			Score:        util.Ptr(e.Score),
			ReviewedAt:   &reviewedAt,
		}
	}

	ids, err := db.CreateVocabularyReviews(ctx, reviews)
	if err != nil {
		return nil, LogVocabularyReviewsOutput{}, fmt.Errorf("failed to log reviews: %w", err)
	}

	created, err := db.ListVocabularyReviews(ctx, domain.VocabularyReviewFilter{UserID: userID, IDs: ids})
	if err != nil {
		return nil, LogVocabularyReviewsOutput{}, fmt.Errorf("failed to load logged reviews: %w", err)
	}

	return nil, LogVocabularyReviewsOutput{Reviews: reviewsToResults(created)}, nil
}
