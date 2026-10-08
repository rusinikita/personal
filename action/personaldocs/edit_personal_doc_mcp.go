package personaldocs

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var EditPersonalDocMCPDefinition = mcp.Tool{
	Name: "edit_personal_doc",
	Annotations: &mcp.ToolAnnotations{
		Title: "Edit a personal document's title or description",
	},
	Description: `Change the title and/or description of a personal document. The text of the document is edited chunk by chunk with edit_personal_doc_chunk.

Inputs:
- doc_id: required, from list_personal_docs
- title: (optional) new title
- description: (optional) new one-line description, max 200 characters

Only provided fields change. At least one field is required and it must differ from the current value.`,
}

type EditPersonalDocInput struct {
	DocID       int64   `json:"doc_id" jsonschema:"Document ID from list_personal_docs"`
	Title       *string `json:"title,omitempty" jsonschema:"New title"`
	Description *string `json:"description,omitempty" jsonschema:"New one-line description, max 200 characters"`
}

type EditPersonalDocOutput struct {
	Doc DocResult `json:"doc" jsonschema:"Updated document"`
}

func EditPersonalDoc(ctx context.Context, _ *mcp.CallToolRequest, input EditPersonalDocInput) (*mcp.CallToolResult, EditPersonalDocOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, EditPersonalDocOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, EditPersonalDocOutput{}, fmt.Errorf("user_id not available in context")
	}

	doc, err := editDoc(ctx, db, userID, input.DocID, input.Title, input.Description)
	if err != nil {
		return nil, EditPersonalDocOutput{}, err
	}

	return nil, EditPersonalDocOutput{Doc: docToResult(*doc)}, nil
}

// editDoc is shared by edit_personal_doc and POST /web/personal-docs/{id}.
func editDoc(ctx context.Context, db gateways.DB, userID int64, docID int64, title, description *string) (*domain.PersonalDoc, error) {
	if title == nil && description == nil {
		return nil, fmt.Errorf("at least one of title or description is required")
	}

	doc, err := db.GetPersonalDoc(ctx, docID, userID)
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

	if err := db.UpdatePersonalDoc(ctx, doc); err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	doc, err = db.GetPersonalDoc(ctx, docID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated document: %w", err)
	}

	return doc, nil
}
