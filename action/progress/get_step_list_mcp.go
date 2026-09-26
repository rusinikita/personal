package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var GetStepListMCPDefinition = mcp.Tool{
	Name: "get_step_list",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint: true,
		Title:        "Get step list",
	},
	Description: `List steps, each with its owning activity's name.

Only ever returns steps whose activity is status=active — a step belonging to a paused/finished/dropped activity is never returned, matching the same visibility rule the web browse/drill-down pages use.

Parameters:
- activity_id: Optional, filter to one activity's steps
- status: Optional, active|finished (defaults to active — finished steps aren't useful to re-surface here)

Each step also has last_executed_at and executions_last_30_days — how recently and how often a repeatable step was done (recorded via create_progress_point's executed_step_id). Monthly review: repeatable steps with empty or >30-day-old last_executed_at weren't done for a month.

Example:
User: "What's next on the move?"
You: [Call get_step_list(activity_id=42)]`,
}

type GetStepListInput struct {
	ActivityID int64  `json:"activity_id,omitempty" jsonschema:"Filter to one activity's steps (omit for all activities)"`
	Status     string `json:"status,omitempty" jsonschema:"Filter by status: active|finished (defaults to active)"`
}

type StepListItem struct {
	ID                   int64  `json:"id" jsonschema:"Step ID"`
	ActivityID           int64  `json:"activity_id" jsonschema:"Activity ID this step belongs to"`
	ActivityName         string `json:"activity_name" jsonschema:"Name of the owning activity"`
	Name                 string `json:"name" jsonschema:"Step name"`
	Type                 string `json:"type" jsonschema:"one_time|repeatable"`
	Status               string `json:"status" jsonschema:"active|finished"`
	CreatedAt            string `json:"created_at" jsonschema:"When the step was created (ISO8601)"`
	LastExecutedAt       string `json:"last_executed_at,omitempty" jsonschema:"When a progress point last executed this repeatable step (ISO8601, empty if never)"`
	ExecutionsLast30Days int    `json:"executions_last_30_days" jsonschema:"How many progress points executed this step in the past 30 days"`
}

type GetStepListOutput struct {
	Steps []StepListItem `json:"steps" jsonschema:"Matching steps"`
}

func GetStepList(ctx context.Context, _ *mcp.CallToolRequest, input GetStepListInput) (*mcp.CallToolResult, GetStepListOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, GetStepListOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, GetStepListOutput{}, fmt.Errorf("user_id not available in context")
	}

	status := input.Status
	if status == "" {
		status = "active"
	}
	if !validStepStatuses[status] {
		return nil, GetStepListOutput{}, fmt.Errorf("invalid status: must be active or finished")
	}

	filter := domain.StepFilter{
		UserID:     userID,
		ActivityID: input.ActivityID,
		Statuses:   []domain.StepStatus{domain.StepStatus(status)},
	}

	steps, err := db.ListStepsWithActivity(ctx, filter)
	if err != nil {
		return nil, GetStepListOutput{}, fmt.Errorf("database error: %w", err)
	}

	items := make([]StepListItem, 0, len(steps))
	for _, st := range steps {
		var lastExecutedAt string
		if st.LastExecutedAt != nil {
			lastExecutedAt = st.LastExecutedAt.Format(time.RFC3339)
		}
		items = append(items, StepListItem{
			ID:                   st.ID,
			ActivityID:           st.ActivityID,
			ActivityName:         st.ActivityName,
			Name:                 st.Name,
			Type:                 string(st.Type),
			Status:               string(st.Status),
			CreatedAt:            st.CreatedAt.Format(time.RFC3339),
			LastExecutedAt:       lastExecutedAt,
			ExecutionsLast30Days: st.ExecutionsLast30Days,
		})
	}

	return nil, GetStepListOutput{Steps: items}, nil
}
