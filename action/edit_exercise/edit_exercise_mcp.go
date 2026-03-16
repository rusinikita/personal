package edit_exercise

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
	"personal/util"
)

var MCPDefinition = mcp.Tool{
	Name: "edit_exercise",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		IdempotentHint:  true,
		Title:           "Edit exercise",
	},
	Description: `Edit the name and/or equipment type of an existing exercise.

At least one of name or equipment_type must be provided.

Equipment types: machine, barbell, dumbbells, bodyweight

Parameters:
- exercise_id: ID of the exercise to edit
- name: New name (optional)
- equipment_type: New equipment type (optional)

Returns the updated exercise object.`,
}

type EditExerciseInput struct {
	ExerciseID    int64  `json:"exercise_id" jsonschema:"Exercise ID"`
	Name          string `json:"name,omitempty" jsonschema:"New exercise name (optional)"`
	EquipmentType string `json:"equipment_type,omitempty" jsonschema:"New equipment type (optional): machine|barbell|dumbbells|bodyweight"`
}

type EditExerciseOutput struct {
	ID            int64   `json:"id"`
	UserID        int64   `json:"user_id"`
	Name          string  `json:"name"`
	EquipmentType string  `json:"equipment_type"`
	CreatedAt     string  `json:"created_at"`
	LastUsedAt    *string `json:"last_used_at"`
}

func EditExercise(ctx context.Context, _ *mcp.CallToolRequest, input EditExerciseInput) (*mcp.CallToolResult, EditExerciseOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, EditExerciseOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, EditExerciseOutput{}, fmt.Errorf("user_id not available in context")
	}

	if strings.TrimSpace(input.Name) == "" && strings.TrimSpace(input.EquipmentType) == "" {
		return nil, EditExerciseOutput{}, fmt.Errorf("at least one of name or equipment_type must be provided")
	}

	if input.EquipmentType != "" && !domain.EquipmentType(input.EquipmentType).IsValid() {
		return nil, EditExerciseOutput{}, fmt.Errorf("equipment_type must be one of: machine, barbell, dumbbells, bodyweight (got: %s)", input.EquipmentType)
	}

	ex, err := db.GetExercise(ctx, input.ExerciseID, userID)
	if err != nil {
		return nil, EditExerciseOutput{}, fmt.Errorf("failed to get exercise: %w", err)
	}
	if ex == nil {
		return nil, EditExerciseOutput{}, fmt.Errorf("exercise not found: id=%d", input.ExerciseID)
	}

	if strings.TrimSpace(input.Name) != "" {
		ex.Name = strings.TrimSpace(input.Name)
	}
	if input.EquipmentType != "" {
		ex.EquipmentType = domain.EquipmentType(input.EquipmentType)
	}

	if err := db.UpdateExercise(ctx, ex); err != nil {
		return nil, EditExerciseOutput{}, fmt.Errorf("failed to update exercise: %w", err)
	}

	updated, err := db.GetExercise(ctx, ex.ID, userID)
	if err != nil {
		return nil, EditExerciseOutput{}, fmt.Errorf("failed to fetch updated exercise: %w", err)
	}

	output := EditExerciseOutput{
		ID:            updated.ID,
		UserID:        updated.UserID,
		Name:          updated.Name,
		EquipmentType: string(updated.EquipmentType),
		CreatedAt:     updated.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if updated.LastUsedAt != nil {
		s := updated.LastUsedAt.Format("2006-01-02T15:04:05Z07:00")
		output.LastUsedAt = &s
	}

	return nil, output, nil
}
