package learning

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var EditLearningExerciseMCPDefinition = mcp.Tool{
	Name:        "edit_learning_exercise",
	Annotations: &mcp.ToolAnnotations{Title: "Edit a planned exercise"},
	Description: `Partial update of an exercise that has no answer yet: only the provided fields change, at least one is required.
Also moves a missed exercise to another day. Fails once the exercise is answered.

Inputs:
- exercise_id: required
- kind: (optional) "" clears it
- content: (optional) full text, not empty
- planned_for: (optional) YYYY-MM-DD, today or later`,
}

type EditLearningExerciseInput struct {
	ExerciseID int64   `json:"exercise_id" jsonschema:"Exercise ID"`
	Kind       *string `json:"kind,omitempty" jsonschema:"New kind; empty string clears it"`
	Content    *string `json:"content,omitempty" jsonschema:"New full text"`
	PlannedFor *string `json:"planned_for,omitempty" jsonschema:"New planned day, YYYY-MM-DD"`
}

type EditLearningExerciseOutput struct {
	Exercise ExerciseResult `json:"exercise" jsonschema:"Updated exercise"`
}

func EditLearningExercise(ctx context.Context, _ *mcp.CallToolRequest, input EditLearningExerciseInput) (*mcp.CallToolResult, EditLearningExerciseOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, EditLearningExerciseOutput{}, err
	}

	if input.Kind == nil && input.Content == nil && input.PlannedFor == nil {
		return nil, EditLearningExerciseOutput{}, fmt.Errorf("at least one of kind, content or planned_for is required")
	}

	exercise, err := loadExercise(ctx, db, userID, input.ExerciseID)
	if err != nil {
		return nil, EditLearningExerciseOutput{}, err
	}
	if exercise.Answer != nil {
		return nil, EditLearningExerciseOutput{}, fmt.Errorf("exercise %d is already answered and can't be edited", exercise.ID)
	}

	if input.Kind != nil {
		if exercise.Kind, err = validateKind(input.Kind); err != nil {
			return nil, EditLearningExerciseOutput{}, err
		}
	}
	if input.Content != nil {
		if strings.TrimSpace(*input.Content) == "" {
			return nil, EditLearningExerciseOutput{}, fmt.Errorf("content cannot be empty")
		}
		exercise.Content = *input.Content
	}
	if input.PlannedFor != nil {
		if exercise.PlannedFor, err = parseFutureDay("planned_for", *input.PlannedFor, time.Now()); err != nil {
			return nil, EditLearningExerciseOutput{}, err
		}
	}

	if err := db.UpdateLearningExercise(ctx, exercise); err != nil {
		return nil, EditLearningExerciseOutput{}, fmt.Errorf("failed to update exercise: %w", err)
	}

	exercise, err = loadExercise(ctx, db, userID, exercise.ID)
	if err != nil {
		return nil, EditLearningExerciseOutput{}, err
	}

	return nil, EditLearningExerciseOutput{Exercise: exerciseToResult(*exercise)}, nil
}
