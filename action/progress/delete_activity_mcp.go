package progress

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
	"personal/util"
)

var DeleteActivityMCPDefinition = mcp.Tool{
	Name: "delete_activity",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		IdempotentHint:  false,
		Title:           "Delete activity",
	},
	Description: `Permanently delete an activity and all its progress history.

This is DIFFERENT from edit_activity(status="dropped"):
- status="dropped"/"finished": keeps the activity and its history, just marked over — still shows in the finished/dropped list and in stats
- delete_activity: erases the activity row and all its progress points for good

Use this tool only when the user explicitly wants an activity gone, not just marked as not working out (use edit_activity with status="dropped" for that).

Required input:
- activity_id: Get from get_activity_list

Errors if the activity doesn't exist, isn't owned by the user, or if a goal still references it (the goal must be deleted or repointed first).

Cannot be undone.`,
}

type DeleteActivityInput struct {
	ActivityID int64 `json:"activity_id" jsonschema:"Activity ID to delete"`
}

type DeleteActivityOutput struct {
	Success bool `json:"success" jsonschema:"Whether the operation succeeded"`
}

func DeleteActivity(ctx context.Context, _ *mcp.CallToolRequest, input DeleteActivityInput) (*mcp.CallToolResult, DeleteActivityOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, DeleteActivityOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, DeleteActivityOutput{}, fmt.Errorf("user_id not available in context")
	}

	activity, err := db.GetActivity(ctx, input.ActivityID, userID)
	if err != nil {
		return nil, DeleteActivityOutput{}, fmt.Errorf("database error: %w", err)
	}
	if activity == nil {
		return nil, DeleteActivityOutput{}, fmt.Errorf("activity not found")
	}

	if err := db.DeleteActivity(ctx, input.ActivityID, userID); err != nil {
		return nil, DeleteActivityOutput{}, fmt.Errorf("failed to delete activity: %w", err)
	}

	return nil, DeleteActivityOutput{Success: true}, nil
}
