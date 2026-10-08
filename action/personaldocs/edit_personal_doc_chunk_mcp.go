package personaldocs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var EditPersonalDocChunkMCPDefinition = mcp.Tool{
	Name: "edit_personal_doc_chunk",
	Annotations: &mcp.ToolAnnotations{
		Title: "Edit or move a chunk of a personal document",
	},
	Description: `Change the text of one chunk of a personal document and/or move it to another position.

A body edit overwrites the chunk's text and keeps the replaced text as the previous body, one step back: revert_personal_doc_chunk brings it back.

A chunk is small by rule: max 60 lines, max 200 characters per line. When the result doesn't fit, put the rest into a new chunk with create_personal_doc_chunk.

Inputs:
- chunk_id: required, from get_personal_doc
- body: (optional) full replacement of the text
- append_body: (optional) text added to the end after a blank line
- position: (optional) from 1 to the number of chunks; moves the chunk and shifts the chunks in between

body and append_body are mutually exclusive. At least one field is required and it must change something.`,
}

type EditPersonalDocChunkInput struct {
	ChunkID    int64   `json:"chunk_id" jsonschema:"Chunk ID from get_personal_doc"`
	Body       *string `json:"body,omitempty" jsonschema:"Full replacement of the chunk's markdown text"`
	AppendBody *string `json:"append_body,omitempty" jsonschema:"Text appended to the body after a blank line"`
	Position   *int    `json:"position,omitempty" jsonschema:"New 1-based position from 1 to the number of chunks"`
}

type EditPersonalDocChunkOutput struct {
	Chunk ChunkResult `json:"chunk" jsonschema:"Updated chunk"`
}

func EditPersonalDocChunk(ctx context.Context, _ *mcp.CallToolRequest, input EditPersonalDocChunkInput) (*mcp.CallToolResult, EditPersonalDocChunkOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, EditPersonalDocChunkOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, EditPersonalDocChunkOutput{}, fmt.Errorf("user_id not available in context")
	}

	chunk, err := editChunk(ctx, db, userID, chunkEdit{
		ChunkID:    input.ChunkID,
		Body:       input.Body,
		AppendBody: input.AppendBody,
		Position:   input.Position,
	})
	if err != nil {
		return nil, EditPersonalDocChunkOutput{}, err
	}

	return nil, EditPersonalDocChunkOutput{Chunk: chunkToResult(*chunk)}, nil
}

// chunkEdit is a partial update of a chunk; nil fields stay unchanged.
type chunkEdit struct {
	ChunkID    int64
	Body       *string
	AppendBody *string
	Position   *int
}

// editChunk is shared by edit_personal_doc_chunk and the web handlers that
// save a chunk's body or move it.
func editChunk(ctx context.Context, db gateways.DB, userID int64, edit chunkEdit) (*domain.PersonalDocChunk, error) {
	if edit.Body == nil && edit.AppendBody == nil && edit.Position == nil {
		return nil, fmt.Errorf("at least one of body, append_body or position is required")
	}
	if edit.Body != nil && edit.AppendBody != nil {
		return nil, fmt.Errorf("body and append_body are mutually exclusive")
	}

	chunk, err := db.GetPersonalDocChunk(ctx, edit.ChunkID, userID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if chunk == nil {
		return nil, fmt.Errorf("chunk not found")
	}

	newBody := chunk.Body
	switch {
	case edit.Body != nil:
		newBody = normalizeBody(*edit.Body)
		if strings.TrimSpace(newBody) == "" {
			return nil, fmt.Errorf("body cannot be empty")
		}
	case edit.AppendBody != nil:
		appended := normalizeBody(*edit.AppendBody)
		if strings.TrimSpace(appended) == "" {
			return nil, fmt.Errorf("append_body cannot be empty")
		}
		newBody = chunk.Body + appendSeparator + appended
	}

	changed := false

	if newBody != chunk.Body {
		if err := validateBody(newBody); err != nil {
			return nil, err
		}
		previousBody := chunk.Body
		chunk.PreviousBody = &previousBody
		chunk.Body = newBody
		chunk.UpdatedAt = time.Now().UTC()
		changed = true
	}

	if edit.Position != nil {
		chunks, err := db.ListPersonalDocChunks(ctx, chunk.DocID, userID)
		if err != nil {
			return nil, fmt.Errorf("database error: %w", err)
		}
		if *edit.Position < 1 || *edit.Position > len(chunks) {
			return nil, fmt.Errorf("position must be between 1 and %d", len(chunks))
		}
		if *edit.Position != chunk.Position {
			chunk.Position = *edit.Position
			changed = true
		}
	}

	if !changed {
		return nil, fmt.Errorf("nothing to change: the values are the same as the current ones")
	}

	if err := db.UpdatePersonalDocChunk(ctx, chunk); err != nil {
		return nil, fmt.Errorf("failed to update chunk: %w", err)
	}

	chunk, err = db.GetPersonalDocChunk(ctx, edit.ChunkID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated chunk: %w", err)
	}

	return chunk, nil
}
