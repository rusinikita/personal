package create_exercise

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
	Name: "create_exercise",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		IdempotentHint:  false,
		Title:           "Create new exercise",
	},
	Description: `Create a new exercise with name and equipment type.

This tool creates a new exercise entry with a name and equipment type.

Equipment types:
- machine: Weight machine or cable machine
- barbell: Barbell exercises
- dumbbells: Dumbbell exercises
- bodyweight: Bodyweight exercises (push-ups, pull-ups, etc.)

Returns the created exercise with ID, user_id, name, equipment_type, created_at, and last_used_at (initially null).`,
}

type CreateExerciseInput struct {
	Name          string `json:"name" jsonschema:"Exercise name"`
	EquipmentType string `json:"equipment_type" jsonschema:"Equipment type (machine|barbell|dumbbells|bodyweight)"`
}

type CreateExerciseOutput struct {
	ID            int64   `json:"id" jsonschema:"Created exercise ID"`
	UserID        int64   `json:"user_id" jsonschema:"User ID"`
	Name          string  `json:"name" jsonschema:"Exercise name"`
	EquipmentType string  `json:"equipment_type" jsonschema:"Equipment type"`
	CreatedAt     string  `json:"created_at" jsonschema:"Creation timestamp (ISO8601)"`
	LastUsedAt    *string `json:"last_used_at" jsonschema:"Last used timestamp (ISO8601), null for new exercises"`
}

func CreateExercise(ctx context.Context, _ *mcp.CallToolRequest, input CreateExerciseInput) (*mcp.CallToolResult, CreateExerciseOutput, error) {
	// Get database from context
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, CreateExerciseOutput{}, fmt.Errorf("database not available in context")
	}

	// 1. Validate input
	if err := validateInput(input); err != nil {
		return nil, CreateExerciseOutput{}, fmt.Errorf("validation error: %w", err)
	}

	// 2. Create domain Exercise object
	exercise := &domain.Exercise{
		UserID:        1, // Single-user mode
		Name:          input.Name,
		EquipmentType: domain.EquipmentType(input.EquipmentType),
	}

	// 3. Save to database
	id, err := db.CreateExercise(ctx, exercise)
	if err != nil {
		return nil, CreateExerciseOutput{}, fmt.Errorf("database error: %w", err)
	}

	// 4. Return success response
	return nil, CreateExerciseOutput{
		ID:            id,
		UserID:        exercise.UserID,
		Name:          exercise.Name,
		EquipmentType: string(exercise.EquipmentType),
		CreatedAt:     exercise.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		LastUsedAt:    nil,
	}, nil
}

func validateInput(input CreateExerciseInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return fmt.Errorf("name is required")
	}

	equipmentType := domain.EquipmentType(input.EquipmentType)
	if !equipmentType.IsValid() {
		return fmt.Errorf("equipment_type must be one of: machine, barbell, dumbbells, bodyweight (got: %s)", input.EquipmentType)
	}

	return nil
}
