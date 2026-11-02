package learning

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
)

var ListVocabularyReviewsMCPDefinition = mcp.Tool{
	Name: "list_vocabulary_reviews",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint:   true,
		IdempotentHint: true,
		Title:          "List word reviews",
	},
	Description: `List a plan's word reviews with term and translation.

Use this tool when:
- Weekly analysis: scored reviews of the week with reviewed_from / reviewed_to (newest first)
- Weekly planning: pending true for reviews still waiting, including missed ones (earliest first)
- One word's full history: vocabulary_id

Parameters:
- learning_plan_id: required
- pending: (optional) true = only pending, false = only scored
- reviewed_from / reviewed_to: (optional) RFC3339 or YYYY-MM-DD window on reviewed_at (from inclusive, to exclusive)
- vocabulary_id: (optional) one word
- limit: (optional) default 100; offset: (optional)`,
}

type ListVocabularyReviewsInput struct {
	LearningPlanID int64  `json:"learning_plan_id" jsonschema:"Plan ID"`
	Pending        *bool  `json:"pending,omitempty" jsonschema:"true = only pending, false = only scored"`
	ReviewedFrom   string `json:"reviewed_from,omitempty" jsonschema:"RFC3339 or YYYY-MM-DD, reviewed_at >= reviewed_from"`
	ReviewedTo     string `json:"reviewed_to,omitempty" jsonschema:"RFC3339 or YYYY-MM-DD, reviewed_at < reviewed_to"`
	VocabularyID   *int64 `json:"vocabulary_id,omitempty" jsonschema:"One word's history"`
	Limit          int    `json:"limit,omitempty" jsonschema:"Default 100"`
	Offset         int    `json:"offset,omitempty" jsonschema:"Default 0"`
}

type ListVocabularyReviewsOutput struct {
	Reviews []ReviewResult `json:"reviews" jsonschema:"Matching reviews"`
	Total   int            `json:"total" jsonschema:"All matching reviews, ignoring limit and offset"`
}

func ListVocabularyReviews(ctx context.Context, _ *mcp.CallToolRequest, input ListVocabularyReviewsInput) (*mcp.CallToolResult, ListVocabularyReviewsOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, ListVocabularyReviewsOutput{}, err
	}

	if input.Limit < 0 || input.Offset < 0 {
		return nil, ListVocabularyReviewsOutput{}, fmt.Errorf("limit and offset cannot be negative")
	}

	plan, err := loadPlan(ctx, db, userID, input.LearningPlanID)
	if err != nil {
		return nil, ListVocabularyReviewsOutput{}, err
	}

	filter := domain.VocabularyReviewFilter{
		UserID:         userID,
		LearningPlanID: &plan.ID,
		VocabularyID:   input.VocabularyID,
		Pending:        input.Pending,
		Limit:          input.Limit,
		Offset:         input.Offset,
	}
	if filter.Limit == 0 {
		filter.Limit = defaultReviewLimit
	}
	if input.ReviewedFrom != "" {
		from, err := parseInstant("reviewed_from", input.ReviewedFrom)
		if err != nil {
			return nil, ListVocabularyReviewsOutput{}, err
		}
		filter.ReviewedFrom = &from
	}
	if input.ReviewedTo != "" {
		to, err := parseInstant("reviewed_to", input.ReviewedTo)
		if err != nil {
			return nil, ListVocabularyReviewsOutput{}, err
		}
		filter.ReviewedTo = &to
	}

	reviews, err := db.ListVocabularyReviews(ctx, filter)
	if err != nil {
		return nil, ListVocabularyReviewsOutput{}, fmt.Errorf("failed to list reviews: %w", err)
	}
	total, err := db.CountVocabularyReviews(ctx, filter)
	if err != nil {
		return nil, ListVocabularyReviewsOutput{}, fmt.Errorf("failed to count reviews: %w", err)
	}

	return nil, ListVocabularyReviewsOutput{Reviews: reviewsToResults(reviews), Total: total}, nil
}
