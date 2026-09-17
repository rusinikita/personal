package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var EditActivityMCPDefinition = mcp.Tool{
	Name: "edit_activity",
	Annotations: &mcp.ToolAnnotations{
		Title: "Edit activity",
	},
	Description: `Update mutable fields of an existing activity.

Use this tool when:
- Activity name, description, or check-in frequency needs updating
- User says "update the description of X" or "change frequency to weekly"
- Life area assignment needs to change

Required input:
- activity_id: Get from get_activity_list

Optional inputs (at least one required):
- name: New activity name
- description: New description (pass empty string "" to clear)
- frequency_days: New check-in frequency in days (1 = daily, 7 = weekly)
- life_part_ids: New life area IDs — replaces all existing (omit to keep current)
- progress_type: New progress type: mood|habit_progress|project_progress|promise_state (omit to keep current). Existing progress points keep their old values — use edit_progress_point per point to remap them to the new type's semantics
- status: New lifecycle status: active|paused|finished|dropped (omit to keep current). This is also how an activity is marked complete now that there's no separate finish_activity tool — pass status="finished" or status="dropped" together with ended_at
- deferred_until: New resume date for a paused activity (ISO8601, pass empty string "" to clear, omit to keep current). Only meaningful alongside status=paused
- started_at: New start date/time (ISO8601, omit to keep current)
- ended_at: New end date/time (ISO8601, pass empty string "" to reopen the activity, omit to keep current)

Example:
User: "Update the description of my driver's license activity - the exam is done"
You: [Call edit_activity(activity_id=42, description="Exam passed, license received")]

Example (completion):
User: "I finished the trainer project, deployed yesterday"
You: [Call edit_activity(activity_id=456, status="finished", ended_at=yesterday)]`,
}

type EditActivityInput struct {
	ActivityID    int64   `json:"activity_id" jsonschema:"Activity ID to edit"`
	Name          *string `json:"name,omitempty" jsonschema:"New name (omit to keep current)"`
	Description   *string `json:"description,omitempty" jsonschema:"New description, pass empty string to clear (omit to keep current)"`
	FrequencyDays *int    `json:"frequency_days,omitempty" jsonschema:"New check-in frequency in days (omit to keep current)"`
	LifePartIDs   []int64 `json:"life_part_ids,omitempty" jsonschema:"New life part IDs replacing existing (omit to keep current)"`
	ProgressType  *string `json:"progress_type,omitempty" jsonschema:"New progress type: mood|habit_progress|project_progress|promise_state (omit to keep current)"`
	Status        *string `json:"status,omitempty" jsonschema:"New lifecycle status: active|paused|finished|dropped (omit to keep current)"`
	DeferredUntil *string `json:"deferred_until,omitempty" jsonschema:"New resume date for a paused activity (ISO8601, pass empty string to clear, omit to keep current)"`
	StartedAt     *string `json:"started_at,omitempty" jsonschema:"New start date/time (ISO8601, omit to keep current)"`
	EndedAt       *string `json:"ended_at,omitempty" jsonschema:"New end date/time (ISO8601, pass empty string to reopen the activity, omit to keep current)"`
}

var validActivityStatuses = map[string]bool{
	"active":   true,
	"paused":   true,
	"finished": true,
	"dropped":  true,
}

type EditActivityOutput struct {
	Activity ActivityResult `json:"activity" jsonschema:"Updated activity"`
}

func EditActivity(ctx context.Context, _ *mcp.CallToolRequest, input EditActivityInput) (*mcp.CallToolResult, EditActivityOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, EditActivityOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, EditActivityOutput{}, fmt.Errorf("user_id not available in context")
	}

	if input.Name == nil && input.Description == nil && input.FrequencyDays == nil &&
		input.LifePartIDs == nil && input.ProgressType == nil && input.Status == nil &&
		input.DeferredUntil == nil && input.StartedAt == nil && input.EndedAt == nil {
		return nil, EditActivityOutput{}, fmt.Errorf("at least one field must be provided to update")
	}

	if input.FrequencyDays != nil && *input.FrequencyDays < 1 {
		return nil, EditActivityOutput{}, fmt.Errorf("frequency_days must be at least 1")
	}

	if input.ProgressType != nil && !validProgressTypes[*input.ProgressType] {
		return nil, EditActivityOutput{}, fmt.Errorf("invalid progress_type: must be one of mood, habit_progress, project_progress, promise_state")
	}

	if input.Status != nil && !validActivityStatuses[*input.Status] {
		return nil, EditActivityOutput{}, fmt.Errorf("invalid status: must be one of active, paused, finished, dropped")
	}

	var startedAt time.Time
	if input.StartedAt != nil {
		t, err := time.Parse(time.RFC3339, *input.StartedAt)
		if err != nil {
			return nil, EditActivityOutput{}, fmt.Errorf("invalid started_at format, expected RFC3339: %w", err)
		}
		startedAt = t
	}

	var endedAt *time.Time
	if input.EndedAt != nil && *input.EndedAt != "" {
		t, err := time.Parse(time.RFC3339, *input.EndedAt)
		if err != nil {
			return nil, EditActivityOutput{}, fmt.Errorf("invalid ended_at format, expected RFC3339: %w", err)
		}
		endedAt = &t
	}

	var deferredUntil *time.Time
	if input.DeferredUntil != nil && *input.DeferredUntil != "" {
		t, err := time.Parse(time.RFC3339, *input.DeferredUntil)
		if err != nil {
			return nil, EditActivityOutput{}, fmt.Errorf("invalid deferred_until format, expected RFC3339: %w", err)
		}
		deferredUntil = &t
	}

	activity, err := db.GetActivity(ctx, input.ActivityID, userID)
	if err != nil {
		return nil, EditActivityOutput{}, fmt.Errorf("database error: %w", err)
	}
	if activity == nil {
		return nil, EditActivityOutput{}, fmt.Errorf("activity not found")
	}

	if input.Name != nil {
		activity.Name = *input.Name
	}
	if input.Description != nil {
		activity.Description = *input.Description
	}
	if input.FrequencyDays != nil {
		activity.FrequencyDays = *input.FrequencyDays
	}
	if input.LifePartIDs != nil {
		activity.LifePartIDs = input.LifePartIDs
	}
	if input.ProgressType != nil {
		activity.ProgressType = domain.ProgressType(*input.ProgressType)
	}
	if input.Status != nil {
		activity.Status = domain.ActivityStatus(*input.Status)
	}
	if input.DeferredUntil != nil {
		activity.DeferredUntil = deferredUntil
	}
	if input.StartedAt != nil {
		activity.StartedAt = startedAt
	}
	if input.EndedAt != nil {
		activity.EndedAt = endedAt
	}

	if err := db.UpdateActivity(ctx, activity); err != nil {
		return nil, EditActivityOutput{}, fmt.Errorf("failed to update activity: %w", err)
	}

	updated, err := db.GetActivity(ctx, input.ActivityID, userID)
	if err != nil {
		return nil, EditActivityOutput{}, fmt.Errorf("failed to fetch updated activity: %w", err)
	}

	return nil, EditActivityOutput{Activity: activityToResult(updated)}, nil
}
