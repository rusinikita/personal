package learning

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/util"
)

var CancelVocabularyReviewsMCPDefinition = mcp.Tool{
	Name: "cancel_vocabulary_reviews",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		Title:           "Cancel pending word reviews",
	},
	Description: `Delete 1-100 pending reviews by id, e.g. missed reviews to replan with plan_vocabulary. All or nothing.
A review with a result cannot be deleted: it is part of the word's history.

Inputs:
- review_ids: required`,
}

type CancelVocabularyReviewsInput struct {
	ReviewIDs []int64 `json:"review_ids" jsonschema:"1-100 pending review IDs"`
}

type CancelVocabularyReviewsOutput struct {
	Reviews []ReviewResult `json:"reviews" jsonschema:"Deleted reviews"`
}

func CancelVocabularyReviews(ctx context.Context, _ *mcp.CallToolRequest, input CancelVocabularyReviewsInput) (*mcp.CallToolResult, CancelVocabularyReviewsOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, CancelVocabularyReviewsOutput{}, err
	}

	if len(input.ReviewIDs) == 0 || len(input.ReviewIDs) > maxReviewBatch {
		return nil, CancelVocabularyReviewsOutput{}, fmt.Errorf("review_ids must have 1-%d entries", maxReviewBatch)
	}
	seen := make(map[int64]bool)
	for _, id := range input.ReviewIDs {
		if seen[id] {
			return nil, CancelVocabularyReviewsOutput{}, fmt.Errorf("review %d is repeated in the call", id)
		}
		seen[id] = true
	}

	reviews, err := db.ListVocabularyReviews(ctx, domain.VocabularyReviewFilter{UserID: userID, IDs: input.ReviewIDs})
	if err != nil {
		return nil, CancelVocabularyReviewsOutput{}, fmt.Errorf("database error: %w", err)
	}
	found := make(map[int64]domain.VocabularyReview, len(reviews))
	for _, r := range reviews {
		found[r.ID] = r
	}
	for _, id := range input.ReviewIDs {
		r, ok := found[id]
		if !ok {
			return nil, CancelVocabularyReviewsOutput{}, fmt.Errorf("review %d not found", id)
		}
		if r.Score != nil {
			return nil, CancelVocabularyReviewsOutput{}, fmt.Errorf("review %d already has a result and can't be deleted", id)
		}
	}

	if err := db.DeleteVocabularyReviews(ctx, userID, input.ReviewIDs); err != nil {
		return nil, CancelVocabularyReviewsOutput{}, fmt.Errorf("failed to cancel reviews: %w", err)
	}

	return nil, CancelVocabularyReviewsOutput{Reviews: reviewsToResults(reviews)}, nil
}
