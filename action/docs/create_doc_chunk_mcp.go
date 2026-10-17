package docs

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var CreateDocChunkMCPDefinition = mcp.Tool{
	Name: "create_doc_chunk",
	Annotations: &mcp.ToolAnnotations{
		Title: "Add a chunk to a document",
	},
	Description: `Add a chunk (one section of markdown text) to a document.

A chunk is small by rule: max 60 lines, max 200 characters per line. A section that doesn't fit is split into two chunks.

Inputs:
- doc_id: required, from list_docs
- body: required, markdown; a section heading, if any, is a line of the body
- position: (optional) from 1 to the number of chunks + 1; default is the end. The chunk already at that position and all following ones move down by one`,
}

type CreateDocChunkInput struct {
	DocID    int64  `json:"doc_id" jsonschema:"Document ID from list_docs"`
	Body     string `json:"body" jsonschema:"Markdown text, max 60 lines of max 200 characters"`
	Position *int   `json:"position,omitempty" jsonschema:"1-based position from 1 to the number of chunks + 1; default is the end"`
}

type CreateDocChunkOutput struct {
	Chunk ChunkResult `json:"chunk" jsonschema:"Created chunk"`
}

func CreateDocChunk(ctx context.Context, _ *mcp.CallToolRequest, input CreateDocChunkInput) (*mcp.CallToolResult, CreateDocChunkOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, CreateDocChunkOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, CreateDocChunkOutput{}, fmt.Errorf("user_id not available in context")
	}

	chunk, err := createChunk(ctx, db, userID, input.DocID, input.Body, input.Position)
	if err != nil {
		return nil, CreateDocChunkOutput{}, err
	}

	return nil, CreateDocChunkOutput{Chunk: chunkToResult(*chunk)}, nil
}

// createChunk is shared by create_doc_chunk and
// POST /web/docs/{id}/chunks. A nil position means the end.
func createChunk(ctx context.Context, db gateways.DB, userID int64, docID int64, body string, position *int) (*domain.DocChunk, error) {
	body = normalizeBody(body)
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("body is required")
	}
	if err := validateBody(body); err != nil {
		return nil, err
	}

	doc, err := db.GetDoc(ctx, docID, userID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if doc == nil {
		return nil, fmt.Errorf("document not found")
	}

	chunk := &domain.DocChunk{UserID: userID, DocID: docID, Body: body}
	if position != nil {
		chunks, err := db.ListDocChunks(ctx, docID, userID)
		if err != nil {
			return nil, fmt.Errorf("database error: %w", err)
		}
		if *position < 1 || *position > len(chunks)+1 {
			return nil, fmt.Errorf("position must be between 1 and %d", len(chunks)+1)
		}
		chunk.Position = *position
	}

	id, err := db.CreateDocChunk(ctx, chunk)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunk: %w", err)
	}

	chunk, err = db.GetDocChunk(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch created chunk: %w", err)
	}

	return chunk, nil
}
