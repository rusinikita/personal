package search_exercises

import (
	"context"
	"fmt"
	"sort"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
)

var MCPDefinition = mcp.Tool{
	Name: "search_exercises",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint:   true,
		IdempotentHint: true,
		Title:          "Search exercises by name variants",
	},
	Description: `Search for exercises by 1-5 name variants (case-insensitive substring match).

Returns all matching exercises ranked by how many variants matched. Searches across ALL exercises regardless of recency, unlike list_exercises which only shows the 20 most recently used.

Use this tool before create_exercise to check if an exercise already exists.

Parameters:
- name_variants: 1-5 name strings to search for (e.g. ["bench", "press"])

Returns:
- exercises: array of matches with exercise_id, name, equipment_type, last_used_at, match_count
- error: validation error message if any`,
}

type SearchExercisesInput struct {
	NameVariants []string `json:"name_variants" jsonschema:"required,1-5 exercise name variants to search for"`
}

type ExerciseMatch struct {
	ExerciseID    int64   `json:"exercise_id"`
	Name          string  `json:"name"`
	EquipmentType string  `json:"equipment_type"`
	LastUsedAt    *string `json:"last_used_at"`
	MatchCount    int     `json:"match_count"`
}

type SearchExercisesOutput struct {
	Exercises []ExerciseMatch `json:"exercises"`
	Error     string          `json:"error,omitempty"`
}

func SearchExercises(ctx context.Context, _ *mcp.CallToolRequest, input SearchExercisesInput) (*mcp.CallToolResult, SearchExercisesOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, SearchExercisesOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, SearchExercisesOutput{}, fmt.Errorf("user_id not available in context")
	}

	if len(input.NameVariants) == 0 {
		return nil, SearchExercisesOutput{Error: "name_variants cannot be empty"}, nil
	}
	if len(input.NameVariants) > 5 {
		return nil, SearchExercisesOutput{Error: "maximum 5 name variants allowed"}, nil
	}
	for _, v := range input.NameVariants {
		if v == "" {
			return nil, SearchExercisesOutput{Error: "name variants cannot be empty strings"}, nil
		}
	}

	matches := make(map[int64]*ExerciseMatch)

	for _, variant := range input.NameVariants {
		exercises, err := db.SearchExercises(ctx, userID, variant)
		if err != nil {
			return nil, SearchExercisesOutput{}, fmt.Errorf("search failed: %w", err)
		}
		for _, ex := range exercises {
			if m, ok := matches[ex.ID]; ok {
				m.MatchCount++
			} else {
				var lastUsedAt *string
				if ex.LastUsedAt != nil {
					s := ex.LastUsedAt.Format("2006-01-02T15:04:05Z07:00")
					lastUsedAt = &s
				}
				matches[ex.ID] = &ExerciseMatch{
					ExerciseID:    ex.ID,
					Name:          ex.Name,
					EquipmentType: string(ex.EquipmentType),
					LastUsedAt:    lastUsedAt,
					MatchCount:    1,
				}
			}
		}
	}

	results := make([]ExerciseMatch, 0, len(matches))
	for _, m := range matches {
		results = append(results, *m)
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].MatchCount != results[j].MatchCount {
			return results[i].MatchCount > results[j].MatchCount
		}
		return results[i].ExerciseID < results[j].ExerciseID
	})

	return nil, SearchExercisesOutput{Exercises: results}, nil
}
