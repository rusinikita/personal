package learning

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var SubmitLearningExerciseMCPDefinition = mcp.Tool{
	Name:        "submit_learning_exercise",
	Annotations: &mcp.ToolAnnotations{Title: "Save the answer to an exercise"},
	Description: `Save the user's answer to an exercise given in chat. The exercise becomes answered and waits for the weekly analysis.

When the answer is a photo of handwriting, save the recognized text word for word, mistakes included — never correct it.
Fails on an empty answer or an exercise that already has one.

Inputs:
- exercise_id: required
- answer: required, the answer as written`,
}

type SubmitLearningExerciseInput struct {
	ExerciseID int64  `json:"exercise_id" jsonschema:"Exercise ID"`
	Answer     string `json:"answer" jsonschema:"The user's answer as written, word for word, not corrected"`
}

type SubmitLearningExerciseOutput struct {
	Exercise ExerciseResult `json:"exercise" jsonschema:"Answered exercise"`
}

func SubmitLearningExercise(ctx context.Context, _ *mcp.CallToolRequest, input SubmitLearningExerciseInput) (*mcp.CallToolResult, SubmitLearningExerciseOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, SubmitLearningExerciseOutput{}, err
	}

	if strings.TrimSpace(input.Answer) == "" {
		return nil, SubmitLearningExerciseOutput{}, fmt.Errorf("answer is required")
	}

	exercise, err := loadExercise(ctx, db, userID, input.ExerciseID)
	if err != nil {
		return nil, SubmitLearningExerciseOutput{}, err
	}
	if exercise.Answer != nil {
		return nil, SubmitLearningExerciseOutput{}, fmt.Errorf("exercise %d already has an answer", exercise.ID)
	}

	if err := db.SubmitLearningExerciseAnswer(ctx, exercise.ID, userID, input.Answer, time.Now().UTC()); err != nil {
		return nil, SubmitLearningExerciseOutput{}, fmt.Errorf("failed to save answer: %w", err)
	}

	exercise, err = loadExercise(ctx, db, userID, exercise.ID)
	if err != nil {
		return nil, SubmitLearningExerciseOutput{}, err
	}

	return nil, SubmitLearningExerciseOutput{Exercise: exerciseToResult(*exercise)}, nil
}
