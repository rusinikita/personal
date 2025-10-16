package progress

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var GetActivityListMCPDefinition = mcp.Tool{
	Name: "get_activity_list",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint: true,
		Title:        "Get activity list",
	},
	Description: `Get list of all active activities for daily/weekly reflection sessions.

Use this tool when:
- Starting a reflection session to see what needs to be checked in
- User asks "what activities do I have?" or "what should I track today?"
- Need to display activities for progress logging

Returns activities ordered by check-in frequency (daily first, then weekly, etc.) and alphabetically by name.
Each activity includes ID (needed for logging progress), name, progress type (mood/habit_progress/project_progress/promise_state), frequency in days, and optional description.

Only returns ACTIVE activities - finished/completed activities are excluded.

Example workflow:
1. Call this tool to get activity list
2. Present activities to user: "Let's check in on: Daily Mood (daily), Morning Workout (daily), Weekly Review (weekly)"
3. For each activity, ask for current progress and use create_progress_point to log it`,
}

type GetActivityListInput struct {
	// No input parameters - uses default user_id from context
}

type ActivityItem struct {
	ID            int64  `json:"id" jsonschema:"Activity ID"`
	Name          string `json:"name" jsonschema:"Activity name"`
	ProgressType  string `json:"progress_type" jsonschema:"Progress type (mood|habit_progress|project_progress|promise_state)"`
	FrequencyDays int    `json:"frequency_days" jsonschema:"Check-in frequency in days"`
	Description   string `json:"description,omitempty" jsonschema:"Activity description"`
}

type GetActivityListOutput struct {
	Activities []ActivityItem `json:"activities" jsonschema:"List of active activities"`
}

func GetActivityList(ctx context.Context, _ *mcp.CallToolRequest, _ GetActivityListInput) (*mcp.CallToolResult, GetActivityListOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, GetActivityListOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, GetActivityListOutput{}, fmt.Errorf("user_id not available in context")
	}

	filter := domain.ActivityFilter{
		UserID:     userID,
		ActiveOnly: true,
	}

	activities, err := db.ListActivities(ctx, filter)
	if err != nil {
		return nil, GetActivityListOutput{}, fmt.Errorf("database error: %w", err)
	}

	output := GetActivityListOutput{
		Activities: make([]ActivityItem, 0, len(activities)),
	}

	for _, a := range activities {
		item := ActivityItem{
			ID:            a.ID,
			Name:          a.Name,
			ProgressType:  string(a.ProgressType),
			FrequencyDays: a.FrequencyDays,
			Description:   a.Description,
		}
		output.Activities = append(output.Activities, item)
	}

	return nil, output, nil
}
