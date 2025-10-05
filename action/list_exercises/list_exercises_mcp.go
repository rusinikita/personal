package list_exercises

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
)

var MCPDefinition = mcp.Tool{
	Name: "list_exercises",
	Annotations: &mcp.ToolAnnotations{
		Title: "List exercises",
	},
	Description: `List exercises sorted by last usage.

This tool returns up to 20 exercises sorted by when they were last used:
- Recently used exercises appear first
- Never-used exercises appear at the end, sorted by name
- Each exercise includes ID, name, equipment type, created timestamp, and last used timestamp

Returns an array of exercises with their details.`,
}

type ListExercisesInput struct {
	// No input parameters - uses default user_id from context
}

type ExerciseItem struct {
	ID         int64   `json:"id" jsonschema:"Exercise ID"`
	Name       string  `json:"name" jsonschema:"Exercise name"`
	Type       string  `json:"type" jsonschema:"Equipment type"`
	CreatedAt  string  `json:"created_at" jsonschema:"Creation timestamp (ISO8601)"`
	LastUsedAt *string `json:"last_used_at" jsonschema:"Last used timestamp (ISO8601), null if never used"`
}

type ListExercisesOutput struct {
	Exercises []ExerciseItem `json:"exercises" jsonschema:"List of exercises"`
}

func ListExercises(ctx context.Context, _ *mcp.CallToolRequest, _ ListExercisesInput) (*mcp.CallToolResult, ListExercisesOutput, error) {
	// Get database from context
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, ListExercisesOutput{}, fmt.Errorf("database not available in context")
	}

	// Get user ID from context
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, ListExercisesOutput{}, fmt.Errorf("user_id not available in context")
	}

	// Call repository to get exercises sorted by last_used_at
	exercises, err := db.ListExercises(ctx, userID, 20)
	if err != nil {
		return nil, ListExercisesOutput{}, fmt.Errorf("database error: %w", err)
	}

	// Convert to output format
	output := ListExercisesOutput{
		Exercises: make([]ExerciseItem, 0, len(exercises)),
	}

	for _, ex := range exercises {
		item := ExerciseItem{
			ID:        ex.ID,
			Name:      ex.Name,
			Type:      string(ex.EquipmentType),
			CreatedAt: ex.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		if ex.LastUsedAt != nil {
			lastUsed := ex.LastUsedAt.Format("2006-01-02T15:04:05Z07:00")
			item.LastUsedAt = &lastUsed
		}

		output.Exercises = append(output.Exercises, item)
	}

	return nil, output, nil
}
