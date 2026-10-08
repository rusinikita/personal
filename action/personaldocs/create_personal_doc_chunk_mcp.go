package personaldocs

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var CreatePersonalDocChunkMCPDefinition = mcp.Tool{
	Name: "create_personal_doc_chunk",
	Annotations: &mcp.ToolAnnotations{
		Title: "Add a chunk to a personal document",
	},
	Description: `Add a chunk (one section of markdown text) to a personal document.

A chunk is small by rule: max 60 lines, max 200 characters per line. A section that doesn't fit is split into two chunks.

Inputs:
- doc_id: required, from list_personal_docs
- body: required, markdown; a section heading, if any, is a line of the body
- position: (optional) from 1 to the number of chunks + 1; default is the end. The chunk already at that position and all following ones move down by one`,
}

type CreatePersonalDocChunkInput struct {
	DocID    int64  `json:"doc_id" jsonschema:"Document ID from list_personal_docs"`
	Body     string `json:"body" jsonschema:"Markdown text, max 60 lines of max 200 characters"`
	Position *int   `json:"position,omitempty" jsonschema:"1-based position from 1 to the number of chunks + 1; default is the end"`
}

type CreatePersonalDocChunkOutput struct {
	Chunk ChunkResult `json:"chunk" jsonschema:"Created chunk"`
}

func CreatePersonalDocChunk(ctx context.Context, _ *mcp.CallToolRequest, input CreatePersonalDocChunkInput) (*mcp.CallToolResult, CreatePersonalDocChunkOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, CreatePersonalDocChunkOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, CreatePersonalDocChunkOutput{}, fmt.Errorf("user_id not available in context")
	}

	chunk, err := createChunk(ctx, db, userID, input.DocID, input.Body, input.Position)
	if err != nil {
		return nil, CreatePersonalDocChunkOutput{}, err
	}

	return nil, CreatePersonalDocChunkOutput{Chunk: chunkToResult(*chunk)}, nil
}

// createChunk is shared by create_personal_doc_chunk and
// POST /web/personal-docs/{id}/chunks. A nil position means the end.
func createChunk(ctx context.Context, db gateways.DB, userID int64, docID int64, body string, position *int) (*domain.PersonalDocChunk, error) {
	body = normalizeBody(body)
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("body is required")
	}
	if err := validateBody(body); err != nil {
		return nil, err
	}

	doc, err := db.GetPersonalDoc(ctx, docID, userID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if doc == nil {
		return nil, fmt.Errorf("document not found")
	}

	chunk := &domain.PersonalDocChunk{UserID: userID, DocID: docID, Body: body}
	if position != nil {
		chunks, err := db.ListPersonalDocChunks(ctx, docID, userID)
		if err != nil {
			return nil, fmt.Errorf("database error: %w", err)
		}
		if *position < 1 || *position > len(chunks)+1 {
			return nil, fmt.Errorf("position must be between 1 and %d", len(chunks)+1)
		}
		chunk.Position = *position
	}

	id, err := db.CreatePersonalDocChunk(ctx, chunk)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunk: %w", err)
	}

	chunk, err = db.GetPersonalDocChunk(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch created chunk: %w", err)
	}

	return chunk, nil
}
