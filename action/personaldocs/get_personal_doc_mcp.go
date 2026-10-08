package personaldocs

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/action/webui"
	"personal/domain"
	"personal/gateways"
)

const defaultChunkLimit = 5

var GetPersonalDocMCPDefinition = mcp.Tool{
	Name: "get_personal_doc",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint: true,
		Title:        "Read a personal document",
	},
	Description: `Read one personal document: its title, description, table of contents and one page of its chunks in order.

A document is read as its chunks in order and edited one chunk at a time, by chunk id.

The chunks come in pages: by default the first 5. Compare the number of returned chunks with total_chunks and call again with a bigger offset to read on. table_of_contents covers the whole document, so to jump to a section use offset = position - 1 of its heading.

Required input:
- doc_id: Get from list_personal_docs

Optional input:
- offset: number of chunks to skip from the start (default 0)
- limit: number of chunks to return (default 5)`,
}

type GetPersonalDocInput struct {
	DocID  int64 `json:"doc_id" jsonschema:"Document ID from list_personal_docs"`
	Offset *int  `json:"offset,omitempty" jsonschema:"Number of chunks to skip from the start (default 0)"`
	Limit  *int  `json:"limit,omitempty" jsonschema:"Number of chunks to return (default 5)"`
}

// ContentsItemResult is one heading of a document's table of contents.
type ContentsItemResult struct {
	Heading  string `json:"heading" jsonschema:"Heading text without markup"`
	Level    int    `json:"level" jsonschema:"Heading level from 1 to 6"`
	Position int    `json:"position" jsonschema:"Position of the chunk that holds the heading"`
}

type GetPersonalDocOutput struct {
	Doc             DocResult            `json:"doc" jsonschema:"The document"`
	TotalChunks     int                  `json:"total_chunks" jsonschema:"Number of chunks in the whole document"`
	TableOfContents []ContentsItemResult `json:"table_of_contents" jsonschema:"Headings of all chunks of the document in reading order"`
	Chunks          []ChunkResult        `json:"chunks" jsonschema:"One page of the document's chunks ordered by position"`
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

	offset := 0
	if input.Offset != nil {
		offset = *input.Offset
	}
	if offset < 0 {
		return nil, GetPersonalDocOutput{}, fmt.Errorf("offset must be 0 or greater")
	}

	limit := defaultChunkLimit
	if input.Limit != nil {
		limit = *input.Limit
	}
	if limit < 1 {
		return nil, GetPersonalDocOutput{}, fmt.Errorf("limit must be 1 or greater")
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

	contents := tableOfContents(chunks)

	start := min(offset, len(chunks))
	// limit has no upper bound, so the end is counted from what is left
	page := chunks[start:]
	if limit < len(page) {
		page = page[:limit]
	}

	results := make([]ChunkResult, len(page))
	for i, chunk := range page {
		results[i] = chunkToResult(chunk)
	}

	return nil, GetPersonalDocOutput{
		Doc:             docToResult(*doc),
		TotalChunks:     len(chunks),
		TableOfContents: contents,
		Chunks:          results,
	}, nil
}

// tableOfContents lists the headings of the chunks in reading order. They
// are found by the renderer of the document page, so both tables of
// contents show the same headings.
func tableOfContents(chunks []domain.PersonalDocChunk) []ContentsItemResult {
	items := []ContentsItemResult{}
	for _, chunk := range chunks {
		_, headings := webui.RenderMarkdownWithHeadings(chunk.Body, true, "")
		for _, heading := range headings {
			items = append(items, ContentsItemResult{
				Heading:  heading.Text,
				Level:    heading.Level,
				Position: chunk.Position,
			})
		}
	}
	return items
}
