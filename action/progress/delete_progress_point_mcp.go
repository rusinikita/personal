package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
	"personal/util"
)

var DeleteProgressPointMCPDefinition = mcp.Tool{
	Name: "delete_progress_point",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		IdempotentHint:  false,
		Title:           "Delete progress point",
	},
	Description: `Delete a single mis-logged or duplicate progress point by its ID.

Returns the deleted point's details as confirmation.

Parameters:
- progress_id: ID of the progress point to delete

Returns:
- progress_id: ID of the deleted point
- value: Value the point had
- note: Note the point had
- progress_at: When the deleted point was logged (ISO8601)`,
}

type DeleteProgressPointInput struct {
	ProgressID int64 `json:"progress_id" jsonschema:"Progress point ID to delete"`
}

type DeleteProgressPointOutput struct {
	ProgressID int64  `json:"progress_id" jsonschema:"ID of the deleted progress point"`
	Value      int    `json:"value" jsonschema:"Value the deleted point had"`
	Note       string `json:"note,omitempty" jsonschema:"Note the deleted point had"`
	ProgressAt string `json:"progress_at" jsonschema:"When the deleted point was logged (ISO8601)"`
}

func DeleteProgressPoint(ctx context.Context, _ *mcp.CallToolRequest, input DeleteProgressPointInput) (*mcp.CallToolResult, DeleteProgressPointOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, DeleteProgressPointOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, DeleteProgressPointOutput{}, fmt.Errorf("user_id not available in context")
	}

	point, err := db.GetProgress(ctx, input.ProgressID, userID)
	if err != nil {
		return nil, DeleteProgressPointOutput{}, fmt.Errorf("database error: %w", err)
	}
	if point == nil {
		return nil, DeleteProgressPointOutput{}, fmt.Errorf("progress point not found")
	}

	if err := db.DeleteProgress(ctx, input.ProgressID, userID); err != nil {
		return nil, DeleteProgressPointOutput{}, fmt.Errorf("failed to delete progress point: %w", err)
	}

	return nil, DeleteProgressPointOutput{
		ProgressID: point.ID,
		Value:      point.Value,
		Note:       point.Note,
		ProgressAt: point.ProgressAt.Format(time.RFC3339),
	}, nil
}
