package docs

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var RevertDocChunkMCPDefinition = mcp.Tool{
	Name: "revert_doc_chunk",
	Annotations: &mcp.ToolAnnotations{
		Title: "Revert the last edit of a chunk",
	},
	Description: `Swap a chunk's text with its previous text, the one replaced by the last edit. One step back only; calling it again undoes the revert.

Required input:
- chunk_id: Get from get_doc

Errors if the chunk was never edited.`,
}

type RevertDocChunkInput struct {
	ChunkID int64 `json:"chunk_id" jsonschema:"Chunk ID from get_doc"`
}

type RevertDocChunkOutput struct {
	Chunk ChunkResult `json:"chunk" jsonschema:"Chunk with its previous text restored"`
}

func RevertDocChunk(ctx context.Context, _ *mcp.CallToolRequest, input RevertDocChunkInput) (*mcp.CallToolResult, RevertDocChunkOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, RevertDocChunkOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, RevertDocChunkOutput{}, fmt.Errorf("user_id not available in context")
	}

	chunk, err := revertChunk(ctx, db, userID, input.ChunkID)
	if err != nil {
		return nil, RevertDocChunkOutput{}, err
	}

	return nil, RevertDocChunkOutput{Chunk: chunkToResult(*chunk)}, nil
}

// revertChunk is shared by revert_doc_chunk and
// POST /web/docs/{id}/chunks/{chunk_id}/revert.
func revertChunk(ctx context.Context, db gateways.DB, userID int64, chunkID int64) (*domain.DocChunk, error) {
	chunk, err := db.GetDocChunk(ctx, chunkID, userID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if chunk == nil {
		return nil, fmt.Errorf("chunk not found")
	}
	if chunk.PreviousBody == nil {
		return nil, fmt.Errorf("chunk has no previous body to revert to")
	}

	body := chunk.Body
	chunk.Body = *chunk.PreviousBody
	chunk.PreviousBody = &body
	chunk.UpdatedAt = time.Now().UTC()

	if err := db.UpdateDocChunk(ctx, chunk); err != nil {
		return nil, fmt.Errorf("failed to revert chunk: %w", err)
	}

	chunk, err = db.GetDocChunk(ctx, chunkID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch reverted chunk: %w", err)
	}

	return chunk, nil
}
