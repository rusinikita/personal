package learning

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/util"
)

var DeleteVocabularyMCPDefinition = mcp.Tool{
	Name: "delete_vocabulary",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		Title:           "Delete a word",
	},
	Description: `Hard-delete one word together with its whole review history, returning what was deleted. Irreversible.

Inputs:
- word_id: required`,
}

type DeleteVocabularyInput struct {
	WordID int64 `json:"word_id" jsonschema:"Word ID"`
}

type DeleteVocabularyOutput struct {
	Word WordResult `json:"word" jsonschema:"Deleted word"`
}

func DeleteVocabulary(ctx context.Context, _ *mcp.CallToolRequest, input DeleteVocabularyInput) (*mcp.CallToolResult, DeleteVocabularyOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, DeleteVocabularyOutput{}, err
	}

	word, err := loadWord(ctx, db, userID, input.WordID, time.Now())
	if err != nil {
		return nil, DeleteVocabularyOutput{}, err
	}

	if err := db.DeleteVocabulary(ctx, word.ID, userID); err != nil {
		return nil, DeleteVocabularyOutput{}, fmt.Errorf("failed to delete word: %w", err)
	}

	return nil, DeleteVocabularyOutput{Word: wordToResult(*word)}, nil
}
