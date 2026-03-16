package merge_exercises

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
	"personal/util"
)

var MCPDefinition = mcp.Tool{
	Name: "merge_exercises",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		IdempotentHint:  false,
		Title:           "Merge exercises",
	},
	Description: `Merge all sets from a source exercise into a target exercise, then delete the source.

Use this to clean up duplicate exercises without losing historical data.

Parameters:
- source_exercise_id: Exercise to merge FROM (will be deleted)
- target_exercise_id: Exercise to merge INTO (kept)

Returns:
- sets_moved: Number of sets reassigned to target
- deleted_exercise_name: Name of the deleted source exercise`,
}

type MergeExercisesInput struct {
	SourceExerciseID int64 `json:"source_exercise_id" jsonschema:"Exercise to merge FROM (will be deleted)"`
	TargetExerciseID int64 `json:"target_exercise_id" jsonschema:"Exercise to merge INTO (kept)"`
}

type MergeExercisesOutput struct {
	SetsMoved           int64  `json:"sets_moved"`
	DeletedExerciseName string `json:"deleted_exercise_name"`
}

func MergeExercises(ctx context.Context, _ *mcp.CallToolRequest, input MergeExercisesInput) (*mcp.CallToolResult, MergeExercisesOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, MergeExercisesOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, MergeExercisesOutput{}, fmt.Errorf("user_id not available in context")
	}

	if input.SourceExerciseID == input.TargetExerciseID {
		return nil, MergeExercisesOutput{}, fmt.Errorf("source_exercise_id and target_exercise_id must be different")
	}

	src, err := db.GetExercise(ctx, input.SourceExerciseID, userID)
	if err != nil {
		return nil, MergeExercisesOutput{}, fmt.Errorf("failed to get source exercise: %w", err)
	}
	if src == nil {
		return nil, MergeExercisesOutput{}, fmt.Errorf("source exercise not found: id=%d", input.SourceExerciseID)
	}

	target, err := db.GetExercise(ctx, input.TargetExerciseID, userID)
	if err != nil {
		return nil, MergeExercisesOutput{}, fmt.Errorf("failed to get target exercise: %w", err)
	}
	if target == nil {
		return nil, MergeExercisesOutput{}, fmt.Errorf("target exercise not found: id=%d", input.TargetExerciseID)
	}

	count, err := db.MoveSetsBetweenExercises(ctx, input.SourceExerciseID, input.TargetExerciseID, userID)
	if err != nil {
		return nil, MergeExercisesOutput{}, fmt.Errorf("failed to move sets: %w", err)
	}

	if err := db.DeleteExercise(ctx, input.SourceExerciseID, userID); err != nil {
		return nil, MergeExercisesOutput{}, fmt.Errorf("failed to delete source exercise: %w", err)
	}

	return nil, MergeExercisesOutput{SetsMoved: count, DeletedExerciseName: src.Name}, nil
}
