package docs

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var GetDocMCPDefinition = mcp.Tool{
	Name: "get_doc",
	Annotations: &mcp.ToolAnnotations{
		Title:        "Get doc",
		ReadOnlyHint: true,
	},
	Description: `Load one convention document by slug and return its full markdown text.

Call list_docs first to get a valid slug. Load a doc only when the current task's conventions are genuinely unclear from the server instructions — not by default at the start of every session.

Required input:
- topic: doc slug from list_docs (e.g. "activity-rituals")

An unknown topic is an error.`,
}

// GetDocInput selects which document to return, validated against the
// embedded content directory.
type GetDocInput struct {
	Topic string `json:"topic" jsonschema:"Doc slug to load, from list_docs"`
}

type GetDocOutput struct {
	Content string `json:"content" jsonschema:"Full markdown text of the requested doc"`
}

func GetDoc(_ context.Context, _ *mcp.CallToolRequest, input GetDocInput) (*mcp.CallToolResult, GetDocOutput, error) {
	if input.Topic == "" {
		return nil, GetDocOutput{}, fmt.Errorf("topic must not be empty")
	}
	content, err := Content(input.Topic)
	if err != nil {
		return nil, GetDocOutput{}, err
	}
	return nil, GetDocOutput{Content: string(content)}, nil
}
