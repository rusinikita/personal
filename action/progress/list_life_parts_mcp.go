package progress

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
)

var ListLifePartsMCPDefinition = mcp.Tool{
	Name: "list_life_parts",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint: true,
		Title:        "List life parts",
	},
	Description: `List the user's life parts (life areas activities can be categorized under), each with its ID, name, and description.

Read-only — there is no tool to create a life part; rows are inserted by hand. Use this to resolve the life_part_ids seen on create_activity/edit_activity input and get_activity_list output into actual names, instead of writing or reading opaque IDs blind.

Example:
User: "What life parts do I have set up?"
You: [Call list_life_parts()]`,
}

type ListLifePartsInput struct {
	// No input parameters
}

type LifePartItem struct {
	ID          int64  `json:"id" jsonschema:"Life part ID"`
	Name        string `json:"name" jsonschema:"Life part name"`
	Description string `json:"description,omitempty" jsonschema:"Life part description"`
}

type ListLifePartsOutput struct {
	LifeParts []LifePartItem `json:"life_parts" jsonschema:"The user's life parts, ordered by name"`
}

func ListLifeParts(ctx context.Context, _ *mcp.CallToolRequest, _ ListLifePartsInput) (*mcp.CallToolResult, ListLifePartsOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, ListLifePartsOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, ListLifePartsOutput{}, fmt.Errorf("user_id not available in context")
	}

	lifeParts, err := db.ListLifeParts(ctx, userID)
	if err != nil {
		return nil, ListLifePartsOutput{}, fmt.Errorf("database error: %w", err)
	}

	items := make([]LifePartItem, 0, len(lifeParts))
	for _, lp := range lifeParts {
		items = append(items, LifePartItem{ID: lp.ID, Name: lp.Name, Description: lp.Description})
	}

	return nil, ListLifePartsOutput{LifeParts: items}, nil
}
