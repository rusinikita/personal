package personaldocs

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
)

var GetPersonalDocMCPDefinition = mcp.Tool{
	Name: "get_personal_doc",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint: true,
		Title:        "Read a personal document",
	},
	Description: `Read one personal document: its title, description and every chunk in order.

A document is read as its chunks in order and edited one chunk at a time, by chunk id.

Required input:
- doc_id: Get from list_personal_docs`,
}

type GetPersonalDocInput struct {
	DocID int64 `json:"doc_id" jsonschema:"Document ID from list_personal_docs"`
}

type GetPersonalDocOutput struct {
	Doc    DocResult     `json:"doc" jsonschema:"The document"`
	Chunks []ChunkResult `json:"chunks" jsonschema:"The document's chunks ordered by position"`
}

func GetPersonalDoc(ctx context.Context, _ *mcp.CallToolRequest, input GetPersonalDocInput) (*mcp.CallToolResult, GetPersonalDocOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, GetPersonalDocOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, GetPersonalDocOutput{}, fmt.Errorf("user_id not available in context")
	}

	doc, err := db.GetPersonalDoc(ctx, input.DocID, userID)
	if err != nil {
		return nil, GetPersonalDocOutput{}, fmt.Errorf("database error: %w", err)
	}
	if doc == nil {
		return nil, GetPersonalDocOutput{}, fmt.Errorf("document not found")
	}

	chunks, err := db.ListPersonalDocChunks(ctx, doc.ID, userID)
	if err != nil {
		return nil, GetPersonalDocOutput{}, fmt.Errorf("database error: %w", err)
	}

	results := make([]ChunkResult, len(chunks))
	for i, chunk := range chunks {
		results[i] = chunkToResult(chunk)
	}

	return nil, GetPersonalDocOutput{Doc: docToResult(*doc), Chunks: results}, nil
}
