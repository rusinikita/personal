package learning

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
)

var CreateLearningExercisesMCPDefinition = mcp.Tool{
	Name:        "create_learning_exercises",
	Annotations: &mcp.ToolAnnotations{Title: "Plan learning exercises"},
	Description: `Create 1-50 planned exercises for the coming days, at the weekly planning. All or nothing.

A task with several sentences is one exercise with one score later. Reviewing a skill means planning another exercise for it.

Each exercise:
- skill_id: required
- content: required, the full exercise text, kept word for word
- kind: (optional) free text, e.g. "put into the correct form", "choose the ending", "translate RU→EL", "kubectl"
- planned_for: required, YYYY-MM-DD (Asia/Nicosia), today or later`,
}

type ExerciseInput struct {
	SkillID    int64   `json:"skill_id" jsonschema:"Skill ID"`
	Content    string  `json:"content" jsonschema:"Full exercise text"`
	Kind       *string `json:"kind,omitempty" jsonschema:"Free-text kind of exercise"`
	PlannedFor string  `json:"planned_for" jsonschema:"Planned day, YYYY-MM-DD, not in the past"`
}

type CreateLearningExercisesInput struct {
	Exercises []ExerciseInput `json:"exercises" jsonschema:"1-50 exercises"`
}

type CreateLearningExercisesOutput struct {
	Exercises []ExerciseResult `json:"exercises" jsonschema:"Created exercises, in input order"`
}

func CreateLearningExercises(ctx context.Context, _ *mcp.CallToolRequest, input CreateLearningExercisesInput) (*mcp.CallToolResult, CreateLearningExercisesOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, CreateLearningExercisesOutput{}, err
	}

	if len(input.Exercises) == 0 || len(input.Exercises) > maxExerciseBatch {
		return nil, CreateLearningExercisesOutput{}, fmt.Errorf("exercises must have 1-%d entries", maxExerciseBatch)
	}

	now := time.Now()
	checkedSkills := make(map[int64]bool)
	exercises := make([]domain.LearningExercise, len(input.Exercises))
	for i, in := range input.Exercises {
		if strings.TrimSpace(in.Content) == "" {
			return nil, CreateLearningExercisesOutput{}, fmt.Errorf("exercise %d: content is required", i+1)
		}
		kind, err := validateKind(in.Kind)
		if err != nil {
			return nil, CreateLearningExercisesOutput{}, fmt.Errorf("exercise %d: %w", i+1, err)
		}
		plannedFor, err := parseFutureDay("planned_for", in.PlannedFor, now)
		if err != nil {
			return nil, CreateLearningExercisesOutput{}, fmt.Errorf("exercise %d: %w", i+1, err)
		}
		if !checkedSkills[in.SkillID] {
			if _, err := loadSkill(ctx, db, userID, in.SkillID); err != nil {
				return nil, CreateLearningExercisesOutput{}, fmt.Errorf("exercise %d: %w", i+1, err)
			}
			checkedSkills[in.SkillID] = true
		}
		exercises[i] = domain.LearningExercise{
			UserID:     userID,
			SkillID:    in.SkillID,
			Kind:       kind,
			Content:    in.Content,
			PlannedFor: plannedFor,
		}
	}

	ids, err := db.CreateLearningExercises(ctx, exercises)
	if err != nil {
		return nil, CreateLearningExercisesOutput{}, fmt.Errorf("failed to create exercises: %w", err)
	}

	results := make([]ExerciseResult, len(ids))
	for i, id := range ids {
		e, err := loadExercise(ctx, db, userID, id)
		if err != nil {
			return nil, CreateLearningExercisesOutput{}, err
		}
		results[i] = exerciseToResult(*e)
	}

	return nil, CreateLearningExercisesOutput{Exercises: results}, nil
}

// validateKind turns an empty kind into nil and checks its length.
func validateKind(kind *string) (*string, error) {
	kind = optionalText(kind)
	if kind != nil && len([]rune(*kind)) > maxKindLength {
		return nil, fmt.Errorf("kind must be at most %d characters", maxKindLength)
	}
	return kind, nil
}
