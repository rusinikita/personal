package delete_transaction

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
	"personal/util"
)

var MCPDefinition = mcp.Tool{
	Name:        "delete_transaction",
	Description: "Delete a transaction by ID. Only transactions owned by the current user can be deleted.",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		Title:           "Delete transaction",
	},
}

// DeleteTransactionInput is the MCP tool input.
type DeleteTransactionInput struct {
	ID int64 `json:"id"`
}

// DeleteTransactionOutput is the MCP tool output.
type DeleteTransactionOutput struct {
	Deleted bool   `json:"deleted"`
	Error   string `json:"error,omitempty"`
}

func DeleteTransaction(ctx context.Context, _ *mcp.CallToolRequest, input DeleteTransactionInput) (*mcp.CallToolResult, DeleteTransactionOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, DeleteTransactionOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, DeleteTransactionOutput{}, fmt.Errorf("user_id not available in context")
	}

	if input.ID == 0 {
		return nil, DeleteTransactionOutput{Error: "id is required"}, nil
	}

	if err := db.DeleteTransaction(ctx, input.ID, userID); err != nil {
		return nil, DeleteTransactionOutput{Error: err.Error()}, nil
	}

	return nil, DeleteTransactionOutput{Deleted: true}, nil
}
