package progress

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var SearchProgressNotesMCPDefinition = mcp.Tool{
	Name: "search_progress_notes",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint:   true,
		IdempotentHint: true,
		Title:          "Search progress notes by keyword variants",
	},
	Description: `Search across all progress point notes using 1-5 keyword variants (case-insensitive).

Returns matching progress points ranked by how many variants matched, with activity name included.

Use this tool when:
- User wants to find past reflections mentioning a topic ("show me when I wrote about stress")
- Searching with synonym variants (e.g. ["gym", "тренировка", "workout"])
- Reviewing notes for a specific activity

Parameters:
- query_variants: 1-5 search terms, each does ILIKE match on note field
- activity_id: (optional) filter to a specific activity
- from: (optional) ISO8601 start date for progress_at
- to: (optional) ISO8601 end date for progress_at
- value_min: (optional) filter by minimum value (-2 to +2)
- value_max: (optional) filter by maximum value (-2 to +2)

Returns results ranked by match_count DESC, then progress_at DESC.
Returns error field (not Go error) for validation failures.`,
}

type SearchProgressNotesInput struct {
	QueryVariants []string `json:"query_variants" jsonschema:"required,1-5 search terms to match against note field"`
	ActivityID    int64    `json:"activity_id,omitempty" jsonschema:"Filter to specific activity ID (0 = all activities)"`
	From          string   `json:"from,omitempty" jsonschema:"ISO8601 start date filter for progress_at"`
	To            string   `json:"to,omitempty" jsonschema:"ISO8601 end date filter for progress_at"`
	ValueMin      *int     `json:"value_min,omitempty" jsonschema:"Minimum value filter (-2 to +2)"`
	ValueMax      *int     `json:"value_max,omitempty" jsonschema:"Maximum value filter (-2 to +2)"`
}

type NoteSearchResult struct {
	ID           int64    `json:"id" jsonschema:"Progress point ID"`
	ActivityID   int64    `json:"activity_id" jsonschema:"Activity ID"`
	ActivityName string   `json:"activity_name" jsonschema:"Activity name"`
	Value        int      `json:"value" jsonschema:"Progress value from -2 to +2"`
	HoursLeft    *float64 `json:"hours_left,omitempty" jsonschema:"Estimated hours remaining"`
	Note         string   `json:"note" jsonschema:"Note text"`
	ProgressAt   string   `json:"progress_at" jsonschema:"When progress was made (ISO8601)"`
	MatchCount   int      `json:"match_count" jsonschema:"Number of query variants that matched this note"`
}

type SearchProgressNotesOutput struct {
	Results []NoteSearchResult `json:"results" jsonschema:"Matching progress notes ranked by match count"`
	Error   string             `json:"error,omitempty" jsonschema:"Validation error message if any"`
}

func SearchProgressNotes(ctx context.Context, _ *mcp.CallToolRequest, input SearchProgressNotesInput) (*mcp.CallToolResult, SearchProgressNotesOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, SearchProgressNotesOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, SearchProgressNotesOutput{}, fmt.Errorf("user_id not available in context")
	}

	if len(input.QueryVariants) == 0 {
		return nil, SearchProgressNotesOutput{Error: "query_variants cannot be empty"}, nil
	}
	if len(input.QueryVariants) > 5 {
		return nil, SearchProgressNotesOutput{Error: "maximum 5 query variants allowed"}, nil
	}
	for _, v := range input.QueryVariants {
		if v == "" {
			return nil, SearchProgressNotesOutput{Error: "query variants cannot be empty strings"}, nil
		}
	}

	var from, to time.Time
	var err error
	if input.From != "" {
		from, err = time.Parse(time.RFC3339, input.From)
		if err != nil {
			return nil, SearchProgressNotesOutput{Error: fmt.Sprintf("invalid from format, expected RFC3339: %v", err)}, nil
		}
	}
	if input.To != "" {
		to, err = time.Parse(time.RFC3339, input.To)
		if err != nil {
			return nil, SearchProgressNotesOutput{Error: fmt.Sprintf("invalid to format, expected RFC3339: %v", err)}, nil
		}
	}

	// Search per variant, deduplicate by point ID with match count
	matches := make(map[int64]*NoteSearchResult)

	for _, variant := range input.QueryVariants {
		points, err := db.SearchProgressNotes(ctx, domain.ProgressNoteSearchFilter{
			UserID:     userID,
			Query:      variant,
			ActivityID: input.ActivityID,
			From:       from,
			To:         to,
			ValueMin:   input.ValueMin,
			ValueMax:   input.ValueMax,
		})
		if err != nil {
			return nil, SearchProgressNotesOutput{}, fmt.Errorf("search failed: %w", err)
		}

		for _, p := range points {
			if m, ok := matches[p.ID]; ok {
				m.MatchCount++
			} else {
				matches[p.ID] = &NoteSearchResult{
					ID:           p.ID,
					ActivityID:   p.ActivityID,
					ActivityName: p.ActivityName,
					Value:        p.Value,
					HoursLeft:    p.HoursLeft,
					Note:         p.Note,
					ProgressAt:   p.ProgressAt.Format(time.RFC3339),
					MatchCount:   1,
				}
			}
		}
	}

	results := make([]NoteSearchResult, 0, len(matches))
	for _, m := range matches {
		results = append(results, *m)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].MatchCount != results[j].MatchCount {
			return results[i].MatchCount > results[j].MatchCount
		}
		return results[i].ProgressAt > results[j].ProgressAt
	})

	return nil, SearchProgressNotesOutput{Results: results}, nil
}
