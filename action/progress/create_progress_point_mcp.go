package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var CreateProgressPointMCPDefinition = mcp.Tool{
	Name: "create_progress_point",
	Annotations: &mcp.ToolAnnotations{
		Title: "Create progress point",
	},
	Description: `Log a progress check-in for an activity after user provides their current state.

Use this tool to:
- Record user's progress after asking "How's your [activity] going?"
- Save progress value after interpreting natural language response
- Log notes and context about the progress

Required inputs:
- activity_id: Get from get_activity_list
- value: Integer from -2 to +2 (convert from natural language using get_progress_type_examples)
  -2 = worst state (hell, missing, changed plans, forgot)
  -1 = bad state (dark, rarely, setback, forgot)
   0 = neutral (gray, trying, stuck, remember)
  +1 = good state (bright, mostly doing, moving forward, did something)
  +2 = best state (happy, crushing it, breakthrough, did something)

Optional inputs:
- note: Save user's explanation (e.g., "Feeling great after morning run")
- hours_left: For projects only - estimated hours remaining (e.g., 5.5)
- progress_at: Timestamp for backdating (ISO8601 format, defaults to now)

Validation:
- Automatically verifies activity exists and user owns it
- Rejects values outside -2 to +2 range
- Returns error if activity not found

Example flow:
1. User says: "I'm feeling sunny today!"
2. You map "sunny" → mood type → value +2 (from get_progress_type_examples)
3. Call create_progress_point(activity_id=123, value=2, note="Feeling sunny!")
4. Confirm: "Great! Logged your mood as sunny ☀️ (+2)"`,
}

type CreateProgressPointInput struct {
	ActivityID int64    `json:"activity_id" jsonschema:"Activity ID to log progress for"`
	Value      int      `json:"value" jsonschema:"Progress value from -2 to +2"`
	Note       string   `json:"note,omitempty" jsonschema:"Optional note about this progress point"`
	HoursLeft  *float64 `json:"hours_left,omitempty" jsonschema:"Estimated hours remaining for projects (omit if not tracking)"`
	ProgressAt string   `json:"progress_at,omitempty" jsonschema:"When progress was made (ISO8601, defaults to now if empty)"`
}

type CreateProgressPointOutput struct {
	Progress ProgressPoint `json:"progress" jsonschema:"Created progress point"`
}

func CreateProgressPoint(ctx context.Context, _ *mcp.CallToolRequest, input CreateProgressPointInput) (*mcp.CallToolResult, CreateProgressPointOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, CreateProgressPointOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, CreateProgressPointOutput{}, fmt.Errorf("user_id not available in context")
	}

	// Validate value range
	if input.Value < -2 || input.Value > 2 {
		return nil, CreateProgressPointOutput{}, fmt.Errorf("value must be between -2 and +2")
	}

	// Verify activity ownership
	activity, err := db.GetActivity(ctx, input.ActivityID, userID)
	if err != nil {
		return nil, CreateProgressPointOutput{}, fmt.Errorf("database error: %w", err)
	}
	if activity == nil {
		return nil, CreateProgressPointOutput{}, fmt.Errorf("activity not found or unauthorized")
	}

	// Parse progress_at or use now
	var progressAt time.Time
	if input.ProgressAt != "" {
		progressAt, err = time.Parse(time.RFC3339, input.ProgressAt)
		if err != nil {
			return nil, CreateProgressPointOutput{}, fmt.Errorf("invalid progress_at format, expected RFC3339: %w", err)
		}
	} else {
		progressAt = time.Now()
	}

	// Create progress point
	point := &domain.ActivityPoint{
		ActivityID: input.ActivityID,
		UserID:     userID,
		Value:      input.Value,
		HoursLeft:  input.HoursLeft,
		Note:       input.Note,
		ProgressAt: progressAt,
	}

	id, err := db.CreateProgress(ctx, point)
	if err != nil {
		return nil, CreateProgressPointOutput{}, fmt.Errorf("failed to create progress point: %w", err)
	}

	// Return created point
	output := CreateProgressPointOutput{
		Progress: ProgressPoint{
			ID:         id,
			Value:      point.Value,
			HoursLeft:  point.HoursLeft,
			Note:       point.Note,
			ProgressAt: point.ProgressAt.Format(time.RFC3339),
		},
	}

	return nil, output, nil
}
