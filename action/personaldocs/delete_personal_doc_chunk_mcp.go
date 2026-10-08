package personaldocs

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
	"personal/util"
)

var DeletePersonalDocChunkMCPDefinition = mcp.Tool{
	Name: "delete_personal_doc_chunk",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		IdempotentHint:  false,
		Title:           "Delete a chunk of a personal document",
	},
	Description: `Permanently delete one chunk of a personal document. The following chunks move up.

Required input:
- chunk_id: Get from get_personal_doc

Errors if the chunk doesn't exist or isn't owned by the user.

Cannot be undone: the chunk's text and its previous text are both gone.`,
}

type DeletePersonalDocChunkInput struct {
	ChunkID int64 `json:"chunk_id" jsonschema:"Chunk ID to delete"`
}

type DeletePersonalDocChunkOutput struct {
	Success bool `json:"success" jsonschema:"Whether the operation succeeded"`
}

func DeletePersonalDocChunk(ctx context.Context, _ *mcp.CallToolRequest, input DeletePersonalDocChunkInput) (*mcp.CallToolResult, DeletePersonalDocChunkOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, DeletePersonalDocChunkOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, DeletePersonalDocChunkOutput{}, fmt.Errorf("user_id not available in context")
	}

	if err := deleteChunk(ctx, db, userID, input.ChunkID); err != nil {
		return nil, DeletePersonalDocChunkOutput{}, err
	}

	return nil, DeletePersonalDocChunkOutput{Success: true}, nil
}

// deleteChunk is shared by delete_personal_doc_chunk and
// POST /web/personal-docs/{id}/chunks/{chunk_id}/delete.
func deleteChunk(ctx context.Context, db gateways.DB, userID int64, chunkID int64) error {
	chunk, err := db.GetPersonalDocChunk(ctx, chunkID, userID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if chunk == nil {
		return fmt.Errorf("chunk not found")
	}

	if err := db.DeletePersonalDocChunk(ctx, chunkID, userID); err != nil {
		return fmt.Errorf("failed to delete chunk: %w", err)
	}

	return nil
}
