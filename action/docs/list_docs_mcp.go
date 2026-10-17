package docs

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
)

var ListDocsMCPDefinition = mcp.Tool{
	Name: "list_docs",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint: true,
		Title:        "List documents",
	},
	Description: `List the user's documents (e.g. development plan, personal instructions, the tracking system's mechanics and rituals): id, title, one-line description and last update. No chunks.

Use this tool when:
- You need the id of a document before reading or editing it

Then call get_doc with the id to read the document.`,
}

type ListDocsInput struct{}

type ListDocsOutput struct {
	Docs []DocResult `json:"docs" jsonschema:"The user's documents ordered by id"`
}

func ListDocs(ctx context.Context, _ *mcp.CallToolRequest, _ ListDocsInput) (*mcp.CallToolResult, ListDocsOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, ListDocsOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, ListDocsOutput{}, fmt.Errorf("user_id not available in context")
	}

	docs, err := db.ListDocs(ctx, userID)
	if err != nil {
		return nil, ListDocsOutput{}, fmt.Errorf("database error: %w", err)
	}

	results := make([]DocResult, len(docs))
	for i, doc := range docs {
		results[i] = docToResult(doc)
	}

	return nil, ListDocsOutput{Docs: results}, nil
}
