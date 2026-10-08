package personaldocs

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var RevertPersonalDocChunkMCPDefinition = mcp.Tool{
	Name: "revert_personal_doc_chunk",
	Annotations: &mcp.ToolAnnotations{
		Title: "Revert the last edit of a chunk",
	},
	Description: `Swap a chunk's text with its previous text, the one replaced by the last edit. One step back only; calling it again undoes the revert.

Required input:
- chunk_id: Get from get_personal_doc

Errors if the chunk was never edited.`,
}

type RevertPersonalDocChunkInput struct {
	ChunkID int64 `json:"chunk_id" jsonschema:"Chunk ID from get_personal_doc"`
}

type RevertPersonalDocChunkOutput struct {
	Chunk ChunkResult `json:"chunk" jsonschema:"Chunk with its previous text restored"`
}

func RevertPersonalDocChunk(ctx context.Context, _ *mcp.CallToolRequest, input RevertPersonalDocChunkInput) (*mcp.CallToolResult, RevertPersonalDocChunkOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, RevertPersonalDocChunkOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, RevertPersonalDocChunkOutput{}, fmt.Errorf("user_id not available in context")
	}

	chunk, err := revertChunk(ctx, db, userID, input.ChunkID)
	if err != nil {
		return nil, RevertPersonalDocChunkOutput{}, err
	}

	return nil, RevertPersonalDocChunkOutput{Chunk: chunkToResult(*chunk)}, nil
}

// revertChunk is shared by revert_personal_doc_chunk and
// POST /web/personal-docs/{id}/chunks/{chunk_id}/revert.
func revertChunk(ctx context.Context, db gateways.DB, userID int64, chunkID int64) (*domain.PersonalDocChunk, error) {
	chunk, err := db.GetPersonalDocChunk(ctx, chunkID, userID)
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

	if err := db.UpdatePersonalDocChunk(ctx, chunk); err != nil {
		return nil, fmt.Errorf("failed to revert chunk: %w", err)
	}

	chunk, err = db.GetPersonalDocChunk(ctx, chunkID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch reverted chunk: %w", err)
	}

	return chunk, nil
}
