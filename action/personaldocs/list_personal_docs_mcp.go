package personaldocs

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
)

var ListPersonalDocsMCPDefinition = mcp.Tool{
	Name: "list_personal_docs",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint: true,
		Title:        "List personal documents",
	},
	Description: `List the user's personal documents (e.g. development plan, personal instructions): id, title, one-line description and last update. No chunks.

Use this tool when:
- You need the id of a personal document before reading or editing it

Then call get_personal_doc with the id to read the document.`,
}

type ListPersonalDocsInput struct{}

type ListPersonalDocsOutput struct {
	Docs []DocResult `json:"docs" jsonschema:"The user's documents ordered by id"`
}

func ListPersonalDocs(ctx context.Context, _ *mcp.CallToolRequest, _ ListPersonalDocsInput) (*mcp.CallToolResult, ListPersonalDocsOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, ListPersonalDocsOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, ListPersonalDocsOutput{}, fmt.Errorf("user_id not available in context")
	}

	docs, err := db.ListPersonalDocs(ctx, userID)
	if err != nil {
		return nil, ListPersonalDocsOutput{}, fmt.Errorf("database error: %w", err)
	}

	results := make([]DocResult, len(docs))
	for i, doc := range docs {
		results[i] = docToResult(doc)
	}

	return nil, ListPersonalDocsOutput{Docs: results}, nil
}
