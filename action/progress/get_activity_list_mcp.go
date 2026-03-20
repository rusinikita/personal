package progress

import (
	"context"
	"fmt"
	"time"

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
	Description: `Get list of activities for daily/weekly reflection sessions or history review.

Use this tool when:
- Starting a reflection session to see what needs to be checked in (active_only=true)
- User asks "what activities do I have?" or "what should I track today?" (active_only=true)
- User wants to see completed/finished activities (active_only=false)

Parameters:
- active_only=true (default): returns only active activities, ordered by check-in urgency
- active_only=false: returns only finished activities, ordered by ended_at DESC

Each activity includes ID, name, progress type (mood/habit_progress/project_progress/promise_state), frequency in days, and optional description.
Finished activities also include ended_at timestamp.

Example workflow:
1. Call this tool with active_only=true to get activity list
2. Present activities to user: "Let's check in on: Daily Mood (daily), Morning Workout (daily), Weekly Review (weekly)"
3. For each activity, ask for current progress and use create_progress_point to log it`,
}

type GetActivityListInput struct {
	ActiveOnly bool `json:"active_only" jsonschema:"If true, return only active activities; if false, return only finished activities"`
}

type ActivityItem struct {
	ID            int64  `json:"id" jsonschema:"Activity ID"`
	Name          string `json:"name" jsonschema:"Activity name"`
	ProgressType  string `json:"progress_type" jsonschema:"Progress type (mood|habit_progress|project_progress|promise_state)"`
	FrequencyDays int    `json:"frequency_days" jsonschema:"Check-in frequency in days"`
	Description   string `json:"description,omitempty" jsonschema:"Activity description"`
	StartedAt     string `json:"started_at" jsonschema:"When activity was started (RFC3339)"`
	EndedAt       string `json:"ended_at,omitempty" jsonschema:"When activity was finished (RFC3339), only set for finished activities"`
}

type GetActivityListOutput struct {
	Activities []ActivityItem `json:"activities" jsonschema:"List of active activities"`
}

func GetActivityList(ctx context.Context, _ *mcp.CallToolRequest, input GetActivityListInput) (*mcp.CallToolResult, GetActivityListOutput, error) {
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
		ActiveOnly: input.ActiveOnly,
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
			StartedAt:     a.StartedAt.Format(time.RFC3339),
		}
		if a.EndedAt != nil {
			item.EndedAt = a.EndedAt.Format(time.RFC3339)
		}
		output.Activities = append(output.Activities, item)
	}

	return nil, output, nil
}
