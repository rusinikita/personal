package learning

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
)

var ListLearningExercisesMCPDefinition = mcp.Tool{
	Name: "list_learning_exercises",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint:   true,
		IdempotentHint: true,
		Title:          "List learning exercises",
	},
	Description: `List exercises with full text and skill names, for one skill or one plan.

Use this tool when:
- Daily: give today's exercises in chat — learning_plan_id, states [planned], due_only true
- Weekly analysis: states [answered] to score them with review_learning_exercises
- Weekly planning: states [reviewed] to see what went wrong

Parameters:
- skill_id or learning_plan_id: exactly one of them
- states: (optional) planned, answered, reviewed; default every state
- due_only: (optional) only exercises whose planned day has come
- limit: (optional) default 20; offset: (optional)

Newest planned day first, or earliest first when states is only planned.`,
}

type ListLearningExercisesInput struct {
	SkillID        *int64   `json:"skill_id,omitempty" jsonschema:"Skill ID; exactly one of skill_id and learning_plan_id"`
	LearningPlanID *int64   `json:"learning_plan_id,omitempty" jsonschema:"Plan ID; exactly one of skill_id and learning_plan_id"`
	States         []string `json:"states,omitempty" jsonschema:"planned, answered, reviewed; default every state"`
	DueOnly        bool     `json:"due_only,omitempty" jsonschema:"Only exercises whose planned day has come"`
	Limit          int      `json:"limit,omitempty" jsonschema:"Default 20"`
	Offset         int      `json:"offset,omitempty" jsonschema:"Default 0"`
}

type ListLearningExercisesOutput struct {
	Exercises []ExerciseResult `json:"exercises" jsonschema:"Matching exercises"`
	Total     int              `json:"total" jsonschema:"All matching exercises, ignoring limit and offset"`
}

var validExerciseStates = map[domain.LearningExerciseState]bool{
	domain.LearningExercisePlanned:  true,
	domain.LearningExerciseAnswered: true,
	domain.LearningExerciseReviewed: true,
}

func ListLearningExercises(ctx context.Context, _ *mcp.CallToolRequest, input ListLearningExercisesInput) (*mcp.CallToolResult, ListLearningExercisesOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, ListLearningExercisesOutput{}, err
	}

	if (input.SkillID == nil) == (input.LearningPlanID == nil) {
		return nil, ListLearningExercisesOutput{}, fmt.Errorf("exactly one of skill_id and learning_plan_id is required")
	}
	if input.Limit < 0 || input.Offset < 0 {
		return nil, ListLearningExercisesOutput{}, fmt.Errorf("limit and offset cannot be negative")
	}

	filter := domain.LearningExerciseFilter{
		UserID:         userID,
		SkillID:        input.SkillID,
		LearningPlanID: input.LearningPlanID,
		Limit:          input.Limit,
		Offset:         input.Offset,
	}
	if filter.Limit == 0 {
		filter.Limit = defaultExerciseLimit
	}

	if input.SkillID != nil {
		if _, err := loadSkill(ctx, db, userID, *input.SkillID); err != nil {
			return nil, ListLearningExercisesOutput{}, err
		}
	} else if _, err := loadPlan(ctx, db, userID, *input.LearningPlanID); err != nil {
		return nil, ListLearningExercisesOutput{}, err
	}

	for _, raw := range input.States {
		state := domain.LearningExerciseState(raw)
		if !validExerciseStates[state] {
			return nil, ListLearningExercisesOutput{}, fmt.Errorf("invalid state %q", raw)
		}
		filter.States = append(filter.States, state)
	}
	filter.OldestFirst = len(filter.States) > 0
	for _, state := range filter.States {
		if state != domain.LearningExercisePlanned {
			filter.OldestFirst = false
		}
	}

	if input.DueOnly {
		now := time.Now()
		filter.DueAt = &now
	}

	exercises, err := db.ListLearningExercises(ctx, filter)
	if err != nil {
		return nil, ListLearningExercisesOutput{}, fmt.Errorf("failed to list exercises: %w", err)
	}
	total, err := db.CountLearningExercises(ctx, filter)
	if err != nil {
		return nil, ListLearningExercisesOutput{}, fmt.Errorf("failed to count exercises: %w", err)
	}

	return nil, ListLearningExercisesOutput{Exercises: exercisesToResults(exercises), Total: total}, nil
}
