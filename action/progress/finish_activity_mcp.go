package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
)

var FinishActivityMCPDefinition = mcp.Tool{
	Name: "finish_activity",
	Annotations: &mcp.ToolAnnotations{
		Title: "Finish activity",
	},
	Description: `Mark an activity as completed/finished when user says it's done.

Use this tool when user indicates completion:
- "I finished the project!"
- "I'm done with this goal"
- "That habit is complete"
- "I want to stop tracking this"

This is DIFFERENT from create_progress_point:
- create_progress_point: Regular check-in ("How's it going?")
- finish_activity: Permanent completion ("It's done!")

When to use:
✅ Projects: "Deployed to production", "Launch completed", "Project finished"
✅ Time-bound goals: "30-day challenge completed", "Goal achieved"
✅ Habits/maintenance: User wants to stop tracking

When NOT to use:
❌ Regular progress updates - use create_progress_point instead
❌ Temporary pauses - activity can still be tracked
❌ Bad outcomes - still finish the activity, just acknowledge it didn't go as planned

Required input:
- activity_id: Get from get_activity_list

Optional input:
- ended_at: When it finished (ISO8601 format, defaults to now)

Effects:
- Activity won't appear in get_activity_list anymore (only shows active)
- All historical progress points remain saved
- Cannot be undone - activity stays finished

Example conversation:
User: "We launched the website yesterday!"
You: "Congratulations! 🎉 Let me mark that project as completed."
[Call finish_activity(activity_id=456, ended_at=yesterday)]
You: "Done! Your website project is now marked as completed. Great work!"`,
}

type FinishActivityInput struct {
	ActivityID int64  `json:"activity_id" jsonschema:"Activity ID to finish"`
	EndedAt    string `json:"ended_at,omitempty" jsonschema:"When activity ended (ISO8601, defaults to now if empty)"`
}

type FinishActivityOutput struct {
	Success bool   `json:"success" jsonschema:"Whether the operation succeeded"`
	Message string `json:"message" jsonschema:"Success or error message"`
	EndedAt string `json:"ended_at" jsonschema:"When activity was finished (ISO8601)"`
}

func FinishActivity(ctx context.Context, _ *mcp.CallToolRequest, input FinishActivityInput) (*mcp.CallToolResult, FinishActivityOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, FinishActivityOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, FinishActivityOutput{}, fmt.Errorf("user_id not available in context")
	}

	// Parse ended_at or use now
	var endedAt time.Time
	var err error
	if input.EndedAt != "" {
		endedAt, err = time.Parse(time.RFC3339, input.EndedAt)
		if err != nil {
			return nil, FinishActivityOutput{}, fmt.Errorf("invalid ended_at format, expected RFC3339: %w", err)
		}
	} else {
		endedAt = time.Now()
	}

	// Finish activity
	err = db.FinishActivity(ctx, input.ActivityID, userID, endedAt)
	if err != nil {
		return nil, FinishActivityOutput{}, fmt.Errorf("failed to finish activity: %w", err)
	}

	output := FinishActivityOutput{
		Success: true,
		Message: "Activity finished",
		EndedAt: endedAt.Format(time.RFC3339),
	}

	return nil, output, nil
}
