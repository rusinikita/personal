package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var EditStepMCPDefinition = mcp.Tool{
	Name: "edit_step",
	Annotations: &mcp.ToolAnnotations{
		Title: "Edit step",
	},
	Description: `Update mutable fields of an existing step.

Use this tool when:
- A step needs renaming
- A step is done and needs closing — pass status="finished", optionally linked to the progress point that closed it
- Reopening a mistakenly closed step — pass status="active"

Dropping a step is NOT a status here — steps have no "dropped" state, use delete_step instead.

Required input:
- step_id: Get from get_step_list

Optional inputs (at least one required):
- name: New step name
- status: New status: active|finished (moving to finished sets closed_at; moving back to active clears it)
- completed_by_progress_point_id: Link the closure to a progress point (e.g. one just created earlier in this chat turn)

Example:
User: "I booked the moving truck"
You: [Call edit_step(step_id=17, status="finished")]`,
}

type EditStepInput struct {
	StepID                     int64   `json:"step_id" jsonschema:"Step ID to edit"`
	Name                       *string `json:"name,omitempty" jsonschema:"New step name (omit to keep current)"`
	Status                     *string `json:"status,omitempty" jsonschema:"New status: active|finished (omit to keep current)"`
	CompletedByProgressPointID *int64  `json:"completed_by_progress_point_id,omitempty" jsonschema:"Progress point that closed this step (omit to keep current)"`
}

var validStepStatuses = map[string]bool{
	"active":   true,
	"finished": true,
}

type EditStepOutput struct {
	Step StepResult `json:"step" jsonschema:"Updated step"`
}

func EditStep(ctx context.Context, _ *mcp.CallToolRequest, input EditStepInput) (*mcp.CallToolResult, EditStepOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, EditStepOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, EditStepOutput{}, fmt.Errorf("user_id not available in context")
	}

	if input.Name == nil && input.Status == nil && input.CompletedByProgressPointID == nil {
		return nil, EditStepOutput{}, fmt.Errorf("at least one field must be provided to update")
	}

	if input.Status != nil && !validStepStatuses[*input.Status] {
		return nil, EditStepOutput{}, fmt.Errorf("invalid status: must be active or finished")
	}

	step, err := db.GetStep(ctx, input.StepID, userID)
	if err != nil {
		return nil, EditStepOutput{}, fmt.Errorf("database error: %w", err)
	}
	if step == nil {
		return nil, EditStepOutput{}, fmt.Errorf("step not found")
	}

	if input.Name != nil {
		step.Name = *input.Name
	}
	if input.Status != nil {
		step.Status = domain.StepStatus(*input.Status)
		if step.Status == domain.StepStatusFinished {
			now := time.Now()
			step.ClosedAt = &now
		} else {
			step.ClosedAt = nil
		}
	}
	if input.CompletedByProgressPointID != nil {
		step.CompletedByProgressPointID = input.CompletedByProgressPointID
	}

	if err := db.UpdateStep(ctx, step); err != nil {
		return nil, EditStepOutput{}, fmt.Errorf("failed to update step: %w", err)
	}

	updated, err := db.GetStep(ctx, input.StepID, userID)
	if err != nil {
		return nil, EditStepOutput{}, fmt.Errorf("failed to fetch updated step: %w", err)
	}

	return nil, EditStepOutput{Step: stepToResult(updated)}, nil
}
