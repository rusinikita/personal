package learning

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
)

var ListVocabularyMCPDefinition = mcp.Tool{
	Name: "list_vocabulary",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint:   true,
		IdempotentHint: true,
		Title:          "List a plan's words",
	},
	Description: `List a plan's words with mastery, uncertainty, debuff, effective_mastery and review fields (scored review count, last review and its score, next planned day).
Least recently reviewed first, never reviewed on top.

Use this tool when:
- Weekly planning: max_mastery 79 for words not mastered or slipping; unscheduled true for words with no pending review

Parameters:
- learning_plan_id: required
- max_mastery: (optional) only words with effective_mastery <= max_mastery
- unscheduled: (optional) only words with no pending review
- limit: (optional) default 50; offset: (optional)`,
}

type ListVocabularyInput struct {
	LearningPlanID int64 `json:"learning_plan_id" jsonschema:"Plan ID"`
	MaxMastery     *int  `json:"max_mastery,omitempty" jsonschema:"Only words with effective mastery at most this"`
	Unscheduled    bool  `json:"unscheduled,omitempty" jsonschema:"Only words with no pending review"`
	Limit          int   `json:"limit,omitempty" jsonschema:"Default 50"`
	Offset         int   `json:"offset,omitempty" jsonschema:"Default 0"`
}

type ListVocabularyOutput struct {
	Words []WordResult `json:"words" jsonschema:"Matching words"`
	Total int          `json:"total" jsonschema:"All matching words, ignoring limit and offset"`
}

func ListVocabulary(ctx context.Context, _ *mcp.CallToolRequest, input ListVocabularyInput) (*mcp.CallToolResult, ListVocabularyOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, ListVocabularyOutput{}, err
	}

	if input.Limit < 0 || input.Offset < 0 {
		return nil, ListVocabularyOutput{}, fmt.Errorf("limit and offset cannot be negative")
	}
	limit := input.Limit
	if limit == 0 {
		limit = defaultWordLimit
	}

	plan, err := loadPlan(ctx, db, userID, input.LearningPlanID)
	if err != nil {
		return nil, ListVocabularyOutput{}, err
	}

	now := time.Now()
	filter := domain.VocabularyFilter{
		UserID:             userID,
		LearningPlanID:     &plan.ID,
		Unscheduled:        input.Unscheduled,
		LeastRecentlyFirst: true,
	}

	// Mastery is not in SQL: with max_mastery load every word, filter, then page.
	if input.MaxMastery != nil {
		words, err := db.ListVocabulary(ctx, filter)
		if err != nil {
			return nil, ListVocabularyOutput{}, fmt.Errorf("failed to list vocabulary: %w", err)
		}
		var matched []domain.VocabularyWord
		for _, w := range withWordMastery(words, now) {
			if w.EffectiveMastery <= *input.MaxMastery {
				matched = append(matched, w)
			}
		}
		start := min(input.Offset, len(matched))
		end := min(start+limit, len(matched))
		return nil, ListVocabularyOutput{Words: wordsToResults(matched[start:end]), Total: len(matched)}, nil
	}

	filter.Limit = limit
	filter.Offset = input.Offset
	words, err := db.ListVocabulary(ctx, filter)
	if err != nil {
		return nil, ListVocabularyOutput{}, fmt.Errorf("failed to list vocabulary: %w", err)
	}
	total, err := db.CountVocabulary(ctx, filter)
	if err != nil {
		return nil, ListVocabularyOutput{}, fmt.Errorf("failed to count vocabulary: %w", err)
	}

	return nil, ListVocabularyOutput{Words: wordsToResults(withWordMastery(words, now)), Total: total}, nil
}
