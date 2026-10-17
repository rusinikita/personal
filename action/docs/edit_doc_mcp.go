package docs

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var EditDocMCPDefinition = mcp.Tool{
	Name: "edit_doc",
	Annotations: &mcp.ToolAnnotations{
		Title: "Edit a document's title or description",
	},
	Description: `Change the title and/or description of a document. The text of the document is edited chunk by chunk with edit_doc_chunk.

Inputs:
- doc_id: required, from list_docs
- title: (optional) new title
- description: (optional) new one-line description, max 200 characters

Only provided fields change. At least one field is required and it must differ from the current value.`,
}

type EditDocInput struct {
	DocID       int64   `json:"doc_id" jsonschema:"Document ID from list_docs"`
	Title       *string `json:"title,omitempty" jsonschema:"New title"`
	Description *string `json:"description,omitempty" jsonschema:"New one-line description, max 200 characters"`
}

type EditDocOutput struct {
	Doc DocResult `json:"doc" jsonschema:"Updated document"`
}

func EditDoc(ctx context.Context, _ *mcp.CallToolRequest, input EditDocInput) (*mcp.CallToolResult, EditDocOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, EditDocOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, EditDocOutput{}, fmt.Errorf("user_id not available in context")
	}

	doc, err := editDoc(ctx, db, userID, input.DocID, input.Title, input.Description)
	if err != nil {
		return nil, EditDocOutput{}, err
	}

	return nil, EditDocOutput{Doc: docToResult(*doc)}, nil
}

// editDoc is shared by edit_doc and POST /web/docs/{id}.
func editDoc(ctx context.Context, db gateways.DB, userID int64, docID int64, title, description *string) (*domain.Doc, error) {
	if title == nil && description == nil {
		return nil, fmt.Errorf("at least one of title or description is required")
	}

	doc, err := db.GetDoc(ctx, docID, userID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if doc == nil {
		return nil, fmt.Errorf("document not found")
	}

	changed := false

	if title != nil {
		newTitle := strings.TrimSpace(*title)
		if err := validateTitle(newTitle); err != nil {
			return nil, err
		}
		if newTitle != doc.Title {
			doc.Title = newTitle
			changed = true
		}
	}

	if description != nil {
		newDescription := strings.TrimSpace(*description)
		if err := validateDescription(newDescription); err != nil {
			return nil, err
		}
		if newDescription != doc.Description {
			doc.Description = newDescription
			changed = true
		}
	}

	if !changed {
		return nil, fmt.Errorf("nothing to change: the values are the same as the current ones")
	}

	if err := db.UpdateDoc(ctx, doc); err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	doc, err = db.GetDoc(ctx, docID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated document: %w", err)
	}

	return doc, nil
}
