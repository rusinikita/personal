package ideas

import (
	"context"
	"fmt"
	"sort"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var SearchIdeasMCPDefinition = mcp.Tool{
	Name: "search_ideas",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint:   true,
		IdempotentHint: true,
		Title:          "Search ideas by keyword variants",
	},
	Description: `Search the user's ideas by body using 1-5 keyword variants (case-insensitive).

Returns matching ideas ranked by how many variants matched, then newest first.

Use this tool when:
- Before create_idea, to find an older similar idea to merge the new one into
- Weekly review: for each inbox idea, to find a duplicate in any status, not just inbox
- The user asks whether they already had a similar idea

Parameters:
- query_variants: 1-5 search terms (key words, synonyms, translations), each does ILIKE match on body
- statuses: (optional) inbox, someday, spike, resolved; default every status, resolved included, so a returning thought shows it was dropped, expired or is blocked

A merged idea is returned with merged_into_id: follow it to the idea it was merged into.
Returns error field (not Go error) for validation failures.`,
}

type SearchIdeasInput struct {
	QueryVariants []string `json:"query_variants" jsonschema:"required,1-5 search terms to match against body"`
	Statuses      []string `json:"statuses,omitempty" jsonschema:"inbox, someday, spike, resolved; default every status"`
}

type IdeaSearchResult struct {
	IdeaResult
	MatchCount int `json:"match_count" jsonschema:"Number of query variants that matched this idea"`
}

type SearchIdeasOutput struct {
	Results []IdeaSearchResult `json:"results" jsonschema:"Matching ideas ranked by match count"`
	Error   string             `json:"error,omitempty" jsonschema:"Validation error message if any"`
}

func SearchIdeas(ctx context.Context, _ *mcp.CallToolRequest, input SearchIdeasInput) (*mcp.CallToolResult, SearchIdeasOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, SearchIdeasOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, SearchIdeasOutput{}, fmt.Errorf("user_id not available in context")
	}

	if err := validateQueryVariants(input.QueryVariants); err != nil {
		return nil, SearchIdeasOutput{Error: err.Error()}, nil
	}

	statuses, err := parseStatuses(input.Statuses)
	if err != nil {
		return nil, SearchIdeasOutput{Error: err.Error()}, nil
	}

	matches, err := searchIdeas(ctx, db, userID, input.QueryVariants, statuses)
	if err != nil {
		return nil, SearchIdeasOutput{}, err
	}

	results := make([]IdeaSearchResult, len(matches))
	for i, m := range matches {
		results[i] = IdeaSearchResult{IdeaResult: ideaToResult(m.Idea), MatchCount: m.MatchCount}
	}

	return nil, SearchIdeasOutput{Results: results}, nil
}

// ideaMatch is one idea found by searchIdeas with the number of variants
// that matched it.
type ideaMatch struct {
	Idea       domain.Idea
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
		if v == "" {
			return fmt.Errorf("query variants cannot be empty strings")
		}
	}
	return nil
}

// searchIdeas runs one ILIKE search per variant and merges the results,
// ranked by match count DESC, then created_at DESC. Shared by search_ideas
// and GET /web/ideas/search; variants must pass validateQueryVariants.
func searchIdeas(ctx context.Context, db gateways.DB, userID int64, variants []string, statuses []domain.IdeaStatus) ([]ideaMatch, error) {
	matches := make(map[int64]*ideaMatch)

	for _, variant := range variants {
		ideas, err := db.SearchIdeas(ctx, domain.IdeaSearchFilter{
			UserID:   userID,
			Query:    variant,
			Statuses: statuses,
		})
		if err != nil {
			return nil, fmt.Errorf("search failed: %w", err)
		}

		for _, idea := range ideas {
			if m, ok := matches[idea.ID]; ok {
				m.MatchCount++
			} else {
				matches[idea.ID] = &ideaMatch{Idea: idea, MatchCount: 1}
			}
		}
	}

	results := make([]ideaMatch, 0, len(matches))
	for _, m := range matches {
		results = append(results, *m)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].MatchCount != results[j].MatchCount {
			return results[i].MatchCount > results[j].MatchCount
		}
		a, b := results[i].Idea, results[j].Idea
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.After(b.CreatedAt)
		}
		return a.ID > b.ID
	})

	return results, nil
}
