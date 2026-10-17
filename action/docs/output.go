package docs

import (
	"time"

	"personal/domain"
)

// DocResult is one document as returned by every docs MCP tool.
type DocResult struct {
	ID          int64  `json:"id" jsonschema:"Document ID"`
	Title       string `json:"title" jsonschema:"Document title"`
	Description string `json:"description" jsonschema:"One-line summary of what the document is for"`
	CreatedAt   string `json:"created_at" jsonschema:"When the document was created (ISO8601, UTC)"`
	UpdatedAt   string `json:"updated_at" jsonschema:"Last write to the document or any of its chunks (ISO8601, UTC)"`
}

// ChunkResult is one chunk as returned by every docs MCP tool.
type ChunkResult struct {
	ID        int64  `json:"id" jsonschema:"Chunk ID"`
	Position  int    `json:"position" jsonschema:"1-based order inside the document"`
	Body      string `json:"body" jsonschema:"Markdown text"`
	UpdatedAt string `json:"updated_at" jsonschema:"Last body edit (ISO8601, UTC)"`
}

func docToResult(doc domain.Doc) DocResult {
	return DocResult{
		ID:          doc.ID,
		Title:       doc.Title,
		Description: doc.Description,
		CreatedAt:   doc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   doc.UpdatedAt.Format(time.RFC3339),
	}
}

func chunkToResult(chunk domain.DocChunk) ChunkResult {
	return ChunkResult{
		ID:        chunk.ID,
		Position:  chunk.Position,
		Body:      chunk.Body,
		UpdatedAt: chunk.UpdatedAt.Format(time.RFC3339),
	}
}
