package personaldocs

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var CreatePersonalDocMCPDefinition = mcp.Tool{
	Name: "create_personal_doc",
	Annotations: &mcp.ToolAnnotations{
		Title: "Create a personal document",
	},
	Description: `Create an empty personal document. Its text is added afterwards, chunk by chunk, with create_personal_doc_chunk.

Required inputs:
- title: document title
- description: one line saying what the document is for, max 200 characters`,
}

type CreatePersonalDocInput struct {
	Title       string `json:"title" jsonschema:"Document title"`
	Description string `json:"description" jsonschema:"One-line summary of what the document is for, max 200 characters"`
}

type CreatePersonalDocOutput struct {
	Doc DocResult `json:"doc" jsonschema:"Created document"`
}

func CreatePersonalDoc(ctx context.Context, _ *mcp.CallToolRequest, input CreatePersonalDocInput) (*mcp.CallToolResult, CreatePersonalDocOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, CreatePersonalDocOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, CreatePersonalDocOutput{}, fmt.Errorf("user_id not available in context")
	}

	doc, err := createDoc(ctx, db, userID, input.Title, input.Description)
	if err != nil {
		return nil, CreatePersonalDocOutput{}, err
	}

	return nil, CreatePersonalDocOutput{Doc: docToResult(*doc)}, nil
}

// createDoc is shared by create_personal_doc and POST /web/personal-docs.
func createDoc(ctx context.Context, db gateways.DB, userID int64, title, description string) (*domain.PersonalDoc, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if err := validateTitle(title); err != nil {
		return nil, err
	}
	if err := validateDescription(description); err != nil {
		return nil, err
	}

	id, err := db.CreatePersonalDoc(ctx, &domain.PersonalDoc{UserID: userID, Title: title, Description: description})
	if err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	doc, err := db.GetPersonalDoc(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch created document: %w", err)
	}

	return doc, nil
}
