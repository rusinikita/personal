package docs

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var ListDocsMCPDefinition = mcp.Tool{
	Name: "list_docs",
	Annotations: &mcp.ToolAnnotations{
		Title:        "List docs",
		ReadOnlyHint: true,
	},
	Description: `List the convention documents available via get_doc: slug, title, and a one-line description each.

Call this when the always-loaded server instructions don't settle how something should be done (e.g. how an activity lifecycle or a weekly review works), then load the matching doc with get_doc(topic=<slug>).

No input.`,
}

// ListDocsInput takes nothing — the tool always returns the full current
// topic list.
type ListDocsInput struct{}

type ListDocsOutput struct {
	Topics []Topic `json:"topics" jsonschema:"Available docs: slug, title, and one-line description each"`
}

func ListDocs(_ context.Context, _ *mcp.CallToolRequest, _ ListDocsInput) (*mcp.CallToolResult, ListDocsOutput, error) {
	topics, err := Topics()
	if err != nil {
		return nil, ListDocsOutput{}, err
	}
	return nil, ListDocsOutput{Topics: topics}, nil
}
