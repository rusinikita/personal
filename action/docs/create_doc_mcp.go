package docs

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var CreateDocMCPDefinition = mcp.Tool{
	Name: "create_doc",
	Annotations: &mcp.ToolAnnotations{
		Title: "Create a document",
	},
	Description: `Create an empty document. Its text is added afterwards, chunk by chunk, with create_doc_chunk.

Required inputs:
- title: document title
- description: one line saying what the document is for, max 200 characters`,
}

type CreateDocInput struct {
	Title       string `json:"title" jsonschema:"Document title"`
	Description string `json:"description" jsonschema:"One-line summary of what the document is for, max 200 characters"`
}

type CreateDocOutput struct {
	Doc DocResult `json:"doc" jsonschema:"Created document"`
}

func CreateDoc(ctx context.Context, _ *mcp.CallToolRequest, input CreateDocInput) (*mcp.CallToolResult, CreateDocOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, CreateDocOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, CreateDocOutput{}, fmt.Errorf("user_id not available in context")
	}

	doc, err := createDoc(ctx, db, userID, input.Title, input.Description)
	if err != nil {
		return nil, CreateDocOutput{}, err
	}

	return nil, CreateDocOutput{Doc: docToResult(*doc)}, nil
}

// createDoc is shared by create_doc and POST /web/docs.
func createDoc(ctx context.Context, db gateways.DB, userID int64, title, description string) (*domain.Doc, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if err := validateTitle(title); err != nil {
		return nil, err
	}
	if err := validateDescription(description); err != nil {
		return nil, err
	}

	id, err := db.CreateDoc(ctx, &domain.Doc{UserID: userID, Title: title, Description: description})
	if err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	doc, err := db.GetDoc(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch created document: %w", err)
	}

	return doc, nil
}
