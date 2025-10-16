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
	Description: `Marks an activity as finished by setting ended_at timestamp.

Verifies ownership, validates activity is currently active (ended_at IS NULL).
Used for completing projects or ending habits/maintenance goals.`,
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
