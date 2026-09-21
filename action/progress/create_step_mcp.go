package progress

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var CreateStepMCPDefinition = mcp.Tool{
	Name: "create_step",
	Annotations: &mcp.ToolAnnotations{
		Title: "Create step",
	},
	Description: `Create a next-action step for an activity.

Use this tool when:
- User mentions a concrete short-horizon next-action tied to an activity ("next I need to email the landlord")
- A reflection surfaces a follow-up worth tracking separately from the progress point itself

Required inputs:
- activity_id: Get from get_activity_list
- name: Short next-action description
- type: one_time|repeatable

Steps created this way always start active with no link back to a progress point — the web progress-point form links a step's creation to the point it's logged alongside internally, this tool does not.

Example:
User: "Next step on the move is booking the moving truck"
You: [Call create_step(activity_id=42, name="Book moving truck", type="one_time")]`,
}

type CreateStepInput struct {
	ActivityID int64  `json:"activity_id" jsonschema:"Activity ID this step belongs to"`
	Name       string `json:"name" jsonschema:"Short next-action description"`
	Type       string `json:"type" jsonschema:"one_time|repeatable"`
}

type StepResult struct {
	ID         int64  `json:"id" jsonschema:"Step ID"`
	ActivityID int64  `json:"activity_id" jsonschema:"Activity ID this step belongs to"`
	Name       string `json:"name" jsonschema:"Step name"`
	Type       string `json:"type" jsonschema:"one_time|repeatable"`
	Status     string `json:"status" jsonschema:"active|finished"`
	ClosedAt   string `json:"closed_at,omitempty" jsonschema:"When the step was closed (ISO8601), empty while active"`
	CreatedAt  string `json:"created_at" jsonschema:"When the step was created (ISO8601)"`
}

type CreateStepOutput struct {
	Step StepResult `json:"step" jsonschema:"Created step"`
}

var validStepTypes = map[string]bool{
	"one_time":   true,
	"repeatable": true,
}

func CreateStep(ctx context.Context, _ *mcp.CallToolRequest, input CreateStepInput) (*mcp.CallToolResult, CreateStepOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, CreateStepOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, CreateStepOutput{}, fmt.Errorf("user_id not available in context")
	}

	if strings.TrimSpace(input.Name) == "" {
		return nil, CreateStepOutput{}, fmt.Errorf("name is required")
	}
	if !validStepTypes[input.Type] {
		return nil, CreateStepOutput{}, fmt.Errorf("invalid type: must be one_time or repeatable")
	}

	activity, err := db.GetActivity(ctx, input.ActivityID, userID)
	if err != nil {
		return nil, CreateStepOutput{}, fmt.Errorf("database error: %w", err)
	}
	if activity == nil {
		return nil, CreateStepOutput{}, fmt.Errorf("activity not found")
	}

	step := &domain.Step{
		UserID:     userID,
		ActivityID: input.ActivityID,
		Name:       input.Name,
		Type:       domain.StepType(input.Type),
	}

	id, err := db.CreateStep(ctx, step)
	if err != nil {
		return nil, CreateStepOutput{}, fmt.Errorf("failed to create step: %w", err)
	}

	created, err := db.GetStep(ctx, id, userID)
	if err != nil {
		return nil, CreateStepOutput{}, fmt.Errorf("failed to fetch created step: %w", err)
	}

	return nil, CreateStepOutput{Step: stepToResult(created)}, nil
}

func stepToResult(st *domain.Step) StepResult {
	r := StepResult{
		ID:         st.ID,
		ActivityID: st.ActivityID,
		Name:       st.Name,
		Type:       string(st.Type),
		Status:     string(st.Status),
		CreatedAt:  st.CreatedAt.Format(time.RFC3339),
	}
	if st.ClosedAt != nil {
		r.ClosedAt = st.ClosedAt.Format(time.RFC3339)
	}
	return r
}
