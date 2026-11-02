package learning

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
)

var ReviewLearningExercisesMCPDefinition = mcp.Tool{
	Name:        "review_learning_exercises",
	Annotations: &mcp.ToolAnnotations{Title: "Score answered exercises"},
	Description: `Save the weekly analysis: 1-50 results of answered exercises. All or nothing.

Each skill's mastery follows from these scores (5 → +20, 4 → +10, 3 → 0, 2 → −10, 1 → −20, kept within 0-100).
A reviewed exercise can be reviewed again to correct it: score and note are overwritten.
Words used in the answers can be logged as indirect reviews with log_vocabulary_reviews.

Each result:
- exercise_id: required, an exercise with an answer
- score: required, 1-5
- note: (optional) mistakes and corrections`,
}

type ExerciseReviewInput struct {
	ExerciseID int64   `json:"exercise_id" jsonschema:"Exercise ID"`
	Score      int     `json:"score" jsonschema:"1-5"`
	Note       *string `json:"note,omitempty" jsonschema:"Mistakes and corrections"`
}

type ReviewLearningExercisesInput struct {
	Reviews []ExerciseReviewInput `json:"reviews" jsonschema:"1-50 results"`
}

type ReviewLearningExercisesOutput struct {
	Exercises []ExerciseResult `json:"exercises" jsonschema:"Reviewed exercises, in input order"`
}

func ReviewLearningExercises(ctx context.Context, _ *mcp.CallToolRequest, input ReviewLearningExercisesInput) (*mcp.CallToolResult, ReviewLearningExercisesOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, ReviewLearningExercisesOutput{}, err
	}

	if len(input.Reviews) == 0 || len(input.Reviews) > maxExerciseBatch {
		return nil, ReviewLearningExercisesOutput{}, fmt.Errorf("reviews must have 1-%d entries", maxExerciseBatch)
	}

	now := time.Now().UTC()
	seen := make(map[int64]bool)
	reviews := make([]domain.LearningExerciseReview, len(input.Reviews))
	for i, in := range input.Reviews {
		if !validScore(in.Score) {
			return nil, ReviewLearningExercisesOutput{}, fmt.Errorf("exercise %d: score must be 1-5", in.ExerciseID)
		}
		if seen[in.ExerciseID] {
			return nil, ReviewLearningExercisesOutput{}, fmt.Errorf("exercise %d is repeated in the call", in.ExerciseID)
		}
		seen[in.ExerciseID] = true

		exercise, err := loadExercise(ctx, db, userID, in.ExerciseID)
		if err != nil {
			return nil, ReviewLearningExercisesOutput{}, err
		}
		if exercise.Answer == nil {
			return nil, ReviewLearningExercisesOutput{}, fmt.Errorf("exercise %d has no answer yet", exercise.ID)
		}

		reviews[i] = domain.LearningExerciseReview{
			ExerciseID: in.ExerciseID,
			Score:      in.Score,
			Note:       optionalText(in.Note),
			ReviewedAt: now,
		}
	}

	if err := db.ReviewLearningExercises(ctx, userID, reviews); err != nil {
		return nil, ReviewLearningExercisesOutput{}, fmt.Errorf("failed to save reviews: %w", err)
	}

	results := make([]ExerciseResult, len(reviews))
	for i, r := range reviews {
		e, err := loadExercise(ctx, db, userID, r.ExerciseID)
		if err != nil {
			return nil, ReviewLearningExercisesOutput{}, err
		}
		results[i] = exerciseToResult(*e)
	}

	return nil, ReviewLearningExercisesOutput{Exercises: results}, nil
}
