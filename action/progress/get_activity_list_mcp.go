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
- User wants to see completed/finished/dropped activities (active_only=false)

Parameters:
- active_only=true (default): returns active activities in "activities" (ordered by check-in urgency) plus every paused activity separately in "paused_activities" — a paused activity is never silently missing, just shown separately
- active_only=false: returns finished/dropped activities in "activities", ordered by ended_at DESC

Each activity includes ID, name, progress type (mood/habit_progress/project_progress/promise_state), status, frequency in days, and optional description.
Finished/dropped activities also include ended_at. Paused activities also include deferred_until when set.

Example workflow:
1. Call this tool with active_only=true to get activity list
2. Present activities to user: "Let's check in on: Daily Mood (daily), Morning Workout (daily), Weekly Review (weekly)"
3. For each activity, ask for current progress and use create_progress_point to log it`,
}

type GetActivityListInput struct {
	ActiveOnly bool `json:"active_only" jsonschema:"If true, return only active/paused activities; if false, return only finished/dropped activities"`
}

type ActivityItem struct {
	ID            int64  `json:"id" jsonschema:"Activity ID"`
	Name          string `json:"name" jsonschema:"Activity name"`
	ProgressType  string `json:"progress_type" jsonschema:"Progress type (mood|habit_progress|project_progress|promise_state)"`
	Status        string `json:"status" jsonschema:"Lifecycle status (active|paused|finished|dropped)"`
	FrequencyDays int    `json:"frequency_days" jsonschema:"Check-in frequency in days"`
	Description   string `json:"description,omitempty" jsonschema:"Activity description"`
	StartedAt     string `json:"started_at" jsonschema:"When activity was started (RFC3339)"`
	EndedAt       string `json:"ended_at,omitempty" jsonschema:"When activity was finished/dropped (RFC3339), only set for finished/dropped activities"`
	DeferredUntil string `json:"deferred_until,omitempty" jsonschema:"When a paused activity should resume (RFC3339), only set for paused activities with a resume date"`
}

type GetActivityListOutput struct {
	Activities       []ActivityItem `json:"activities" jsonschema:"List of active (or finished/dropped, depending on active_only) activities"`
	PausedActivities []ActivityItem `json:"paused_activities,omitempty" jsonschema:"List of paused activities, populated alongside activities when active_only=true"`
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

	var statuses []domain.ActivityStatus
	if input.ActiveOnly {
		statuses = []domain.ActivityStatus{domain.ActivityStatusActive}
	} else {
		statuses = []domain.ActivityStatus{domain.ActivityStatusFinished, domain.ActivityStatusDropped}
	}

	activities, err := db.ListActivities(ctx, domain.ActivityFilter{UserID: userID, Statuses: statuses})
	if err != nil {
		return nil, GetActivityListOutput{}, fmt.Errorf("database error: %w", err)
	}

	output := GetActivityListOutput{Activities: toActivityItems(activities)}

	if input.ActiveOnly {
		paused, err := db.ListActivities(ctx, domain.ActivityFilter{UserID: userID, Statuses: []domain.ActivityStatus{domain.ActivityStatusPaused}})
		if err != nil {
			return nil, GetActivityListOutput{}, fmt.Errorf("database error: %w", err)
		}
		output.PausedActivities = toActivityItems(paused)
	}

	return nil, output, nil
}

func toActivityItems(activities []domain.Activity) []ActivityItem {
	items := make([]ActivityItem, 0, len(activities))
	for _, a := range activities {
		item := ActivityItem{
			ID:            a.ID,
			Name:          a.Name,
			ProgressType:  string(a.ProgressType),
			Status:        string(a.Status),
			FrequencyDays: a.FrequencyDays,
			Description:   a.Description,
			StartedAt:     a.StartedAt.Format(time.RFC3339),
		}
		if a.EndedAt != nil {
			item.EndedAt = a.EndedAt.Format(time.RFC3339)
		}
		if a.DeferredUntil != nil {
			item.DeferredUntil = a.DeferredUntil.Format(time.RFC3339)
		}
		items = append(items, item)
	}
	return items
}
