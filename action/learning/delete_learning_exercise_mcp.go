package learning

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/util"
)

var DeleteLearningExerciseMCPDefinition = mcp.Tool{
	Name: "delete_learning_exercise",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		Title:           "Delete a learning exercise",
	},
	Description: `Hard-delete one exercise in any state, returning what was deleted. Irreversible: a reviewed exercise takes its score out of the skill's mastery.

Inputs:
- exercise_id: required`,
}

type DeleteLearningExerciseInput struct {
	ExerciseID int64 `json:"exercise_id" jsonschema:"Exercise ID"`
}

type DeleteLearningExerciseOutput struct {
	Exercise ExerciseResult `json:"exercise" jsonschema:"Deleted exercise"`
}

func DeleteLearningExercise(ctx context.Context, _ *mcp.CallToolRequest, input DeleteLearningExerciseInput) (*mcp.CallToolResult, DeleteLearningExerciseOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, DeleteLearningExerciseOutput{}, err
	}

	exercise, err := loadExercise(ctx, db, userID, input.ExerciseID)
	if err != nil {
		return nil, DeleteLearningExerciseOutput{}, err
	}

	if err := db.DeleteLearningExercise(ctx, exercise.ID, userID); err != nil {
		return nil, DeleteLearningExerciseOutput{}, fmt.Errorf("failed to delete exercise: %w", err)
	}

	return nil, DeleteLearningExerciseOutput{Exercise: exerciseToResult(*exercise)}, nil
}
