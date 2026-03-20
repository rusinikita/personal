package progress

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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

Example:
User: "Update the description of my driver's license activity - the exam is done"
You: [Call edit_activity(activity_id=42, description="Exam passed, license received")]`,
}

type EditActivityInput struct {
	ActivityID    int64   `json:"activity_id" jsonschema:"Activity ID to edit"`
	Name          *string `json:"name,omitempty" jsonschema:"New name (omit to keep current)"`
	Description   *string `json:"description,omitempty" jsonschema:"New description, pass empty string to clear (omit to keep current)"`
	FrequencyDays *int    `json:"frequency_days,omitempty" jsonschema:"New check-in frequency in days (omit to keep current)"`
	LifePartIDs   []int64 `json:"life_part_ids,omitempty" jsonschema:"New life part IDs replacing existing (omit to keep current)"`
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

	if input.Name == nil && input.Description == nil && input.FrequencyDays == nil && input.LifePartIDs == nil {
		return nil, EditActivityOutput{}, fmt.Errorf("at least one field must be provided to update")
	}

	if input.FrequencyDays != nil && *input.FrequencyDays < 1 {
		return nil, EditActivityOutput{}, fmt.Errorf("frequency_days must be at least 1")
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

	if err := db.UpdateActivity(ctx, activity); err != nil {
		return nil, EditActivityOutput{}, fmt.Errorf("failed to update activity: %w", err)
	}

	updated, err := db.GetActivity(ctx, input.ActivityID, userID)
	if err != nil {
		return nil, EditActivityOutput{}, fmt.Errorf("failed to fetch updated activity: %w", err)
	}

	return nil, EditActivityOutput{Activity: activityToResult(updated)}, nil
}
