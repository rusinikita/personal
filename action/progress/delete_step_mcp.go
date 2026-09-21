package progress

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
	"personal/util"
)

var DeleteStepMCPDefinition = mcp.Tool{
	Name: "delete_step",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		IdempotentHint:  false,
		Title:           "Delete step",
	},
	Description: `Permanently delete a step.

Use this tool when the user abandons a next-action they no longer want tracked — a step has no "dropped" status, unlike an activity, so removing one is always a hard delete.

Required input:
- step_id: Get from get_step_list

Errors if the step doesn't exist or isn't owned by the user.

Cannot be undone.`,
}

type DeleteStepInput struct {
	StepID int64 `json:"step_id" jsonschema:"Step ID to delete"`
}

type DeleteStepOutput struct {
	Success bool `json:"success" jsonschema:"Whether the operation succeeded"`
}

func DeleteStep(ctx context.Context, _ *mcp.CallToolRequest, input DeleteStepInput) (*mcp.CallToolResult, DeleteStepOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, DeleteStepOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, DeleteStepOutput{}, fmt.Errorf("user_id not available in context")
	}

	step, err := db.GetStep(ctx, input.StepID, userID)
	if err != nil {
		return nil, DeleteStepOutput{}, fmt.Errorf("database error: %w", err)
	}
	if step == nil {
		return nil, DeleteStepOutput{}, fmt.Errorf("step not found")
	}

	if err := db.DeleteStep(ctx, input.StepID, userID); err != nil {
		return nil, DeleteStepOutput{}, fmt.Errorf("failed to delete step: %w", err)
	}

	return nil, DeleteStepOutput{Success: true}, nil
}
