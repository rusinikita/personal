package ideas

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var CreateIdeaMCPDefinition = mcp.Tool{
	Name: "create_idea",
	Annotations: &mcp.ToolAnnotations{
		Title: "Capture an idea",
	},
	Description: `Capture a raw idea outside any existing activity into the inbox.

Use this tool when:
- The user dictates a thought that isn't a note or step on an existing activity

Before creating, call search_ideas: if an older similar idea exists, create this one and then merge it into the older one with resolve_idea(merged), so the older idea gets +1 to surface_count.

Required inputs:
- body: the user's own words, not a paraphrase`,
}

type CreateIdeaInput struct {
	Body string `json:"body" jsonschema:"The user's own words"`
}

type CreateIdeaOutput struct {
	Idea IdeaResult `json:"idea" jsonschema:"Created idea"`
}

func CreateIdea(ctx context.Context, _ *mcp.CallToolRequest, input CreateIdeaInput) (*mcp.CallToolResult, CreateIdeaOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, CreateIdeaOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, CreateIdeaOutput{}, fmt.Errorf("user_id not available in context")
	}

	idea, err := createIdea(ctx, db, userID, input.Body)
	if err != nil {
		return nil, CreateIdeaOutput{}, err
	}

	return nil, CreateIdeaOutput{Idea: ideaToResult(*idea)}, nil
}

// createIdea is shared by create_idea and POST /web/ideas.
func createIdea(ctx context.Context, db gateways.DB, userID int64, body string) (*domain.Idea, error) {
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("body is required")
	}

	id, err := db.CreateIdea(ctx, &domain.Idea{UserID: userID, Body: body})
	if err != nil {
		return nil, fmt.Errorf("failed to create idea: %w", err)
	}

	idea, err := db.GetIdea(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch created idea: %w", err)
	}

	return idea, nil
}
