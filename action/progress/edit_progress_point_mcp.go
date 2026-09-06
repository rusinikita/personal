package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
)

var EditProgressPointMCPDefinition = mcp.Tool{
	Name: "edit_progress_point",
	Annotations: &mcp.ToolAnnotations{
		Title: "Edit progress point",
	},
	Description: `Update mutable fields of an existing progress point.

Use this tool when:
- A logged value or note was wrong ("that should've been -1, not +1")
- A progress point needs backdating to the correct day
- hours_left needs correcting for a project

Required input:
- progress_id: ID of the progress point to edit (from create_progress_point's result, get_activity_stats, or search_progress_notes)

Optional inputs (at least one required):
- value: New value from -2 to +2
- note: New note (pass empty string "" to clear)
- hours_left: New estimated hours remaining
- progress_at: New date/time (ISO8601, omit to keep current)

Example:
User: "That mood entry should've been -1, not +1"
You: [Call edit_progress_point(progress_id=789, value=-1)]`,
}

type EditProgressPointInput struct {
	ProgressID int64    `json:"progress_id" jsonschema:"Progress point ID to edit"`
	Value      *int     `json:"value,omitempty" jsonschema:"New progress value from -2 to +2 (omit to keep current)"`
	Note       *string  `json:"note,omitempty" jsonschema:"New note, pass empty string to clear (omit to keep current)"`
	HoursLeft  *float64 `json:"hours_left,omitempty" jsonschema:"New estimated hours remaining (omit to keep current)"`
	ProgressAt *string  `json:"progress_at,omitempty" jsonschema:"New date/time (ISO8601, omit to keep current)"`
}

type EditProgressPointOutput struct {
	Progress ProgressPoint `json:"progress" jsonschema:"Updated progress point"`
}

func EditProgressPoint(ctx context.Context, _ *mcp.CallToolRequest, input EditProgressPointInput) (*mcp.CallToolResult, EditProgressPointOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, EditProgressPointOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, EditProgressPointOutput{}, fmt.Errorf("user_id not available in context")
	}

	if input.Value == nil && input.Note == nil && input.HoursLeft == nil && input.ProgressAt == nil {
		return nil, EditProgressPointOutput{}, fmt.Errorf("at least one field must be provided to update")
	}

	if input.Value != nil && (*input.Value < -2 || *input.Value > 2) {
		return nil, EditProgressPointOutput{}, fmt.Errorf("value must be between -2 and +2")
	}

	var progressAt time.Time
	if input.ProgressAt != nil {
		t, err := time.Parse(time.RFC3339, *input.ProgressAt)
		if err != nil {
			return nil, EditProgressPointOutput{}, fmt.Errorf("invalid progress_at format, expected RFC3339: %w", err)
		}
		progressAt = t
	}

	point, err := db.GetProgress(ctx, input.ProgressID, userID)
	if err != nil {
		return nil, EditProgressPointOutput{}, fmt.Errorf("database error: %w", err)
	}
	if point == nil {
		return nil, EditProgressPointOutput{}, fmt.Errorf("progress point not found")
	}

	if input.Value != nil {
		point.Value = *input.Value
	}
	if input.Note != nil {
		point.Note = *input.Note
	}
	if input.HoursLeft != nil {
		point.HoursLeft = input.HoursLeft
	}
	if input.ProgressAt != nil {
		point.ProgressAt = progressAt
	}

	if err := db.UpdateProgress(ctx, point); err != nil {
		return nil, EditProgressPointOutput{}, fmt.Errorf("failed to update progress point: %w", err)
	}

	return nil, EditProgressPointOutput{
		Progress: ProgressPoint{
			ID:         point.ID,
			Value:      point.Value,
			HoursLeft:  point.HoursLeft,
			Note:       point.Note,
			ProgressAt: point.ProgressAt.Format(time.RFC3339),
		},
	}, nil
}
