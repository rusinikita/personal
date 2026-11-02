package learning

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var SearchVocabularyMCPDefinition = mcp.Tool{
	Name: "search_vocabulary",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint:   true,
		IdempotentHint: true,
		Title:          "Search words",
	},
	Description: `Find words by 1-5 query variants, each a case-insensitive substring match on term, translation and notes.
Ranked by how many variants matched, then newest first. Returns the same mastery and review fields as list_vocabulary.

Use this tool when:
- Before add_vocabulary, to check a word is not there yet
- At the weekly analysis, to find the words used in an exercise answer before log_vocabulary_reviews

Parameters:
- query_variants: required, 1-5 terms (word forms, translations)
- learning_plan_id: (optional) search inside one plan`,
}

type SearchVocabularyInput struct {
	QueryVariants  []string `json:"query_variants" jsonschema:"1-5 search terms"`
	LearningPlanID *int64   `json:"learning_plan_id,omitempty" jsonschema:"Search inside one plan"`
}

type WordSearchResult struct {
	WordResult
	MatchCount int `json:"match_count" jsonschema:"Number of query variants that matched this word"`
}

type SearchVocabularyOutput struct {
	Results []WordSearchResult `json:"results" jsonschema:"Matching words ranked by match count"`
}

func SearchVocabulary(ctx context.Context, _ *mcp.CallToolRequest, input SearchVocabularyInput) (*mcp.CallToolResult, SearchVocabularyOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, SearchVocabularyOutput{}, err
	}

	if err := validateQueryVariants(input.QueryVariants); err != nil {
		return nil, SearchVocabularyOutput{}, err
	}
	if input.LearningPlanID != nil {
		if _, err := loadPlan(ctx, db, userID, *input.LearningPlanID); err != nil {
			return nil, SearchVocabularyOutput{}, err
		}
	}

	matches, err := searchVocabulary(ctx, db, userID, input.QueryVariants, input.LearningPlanID, time.Now())
	if err != nil {
		return nil, SearchVocabularyOutput{}, err
	}

	results := make([]WordSearchResult, len(matches))
	for i, m := range matches {
		results[i] = WordSearchResult{WordResult: wordToResult(m.Word), MatchCount: m.MatchCount}
	}

	return nil, SearchVocabularyOutput{Results: results}, nil
}
