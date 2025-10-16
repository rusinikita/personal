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
	Description: `Creates a progress point for an activity.

Validates activity ownership, validates value is between -2 and +2, sets progress_at to current time if not provided.
Optional fields: note, hours_left (for projects tracking remaining work), progress_at (defaults to now).`,
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
