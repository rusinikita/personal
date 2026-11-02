package learning

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var EditVocabularyMCPDefinition = mcp.Tool{
	Name:        "edit_vocabulary",
	Annotations: &mcp.ToolAnnotations{Title: "Edit a word"},
	Description: `Partial update of a word: only the provided fields change, at least one is required. Reviews are untouched.

Inputs:
- word_id: required
- term: (optional) not empty, unique in the plan ignoring case
- translation: (optional) not empty
- notes: (optional) "" clears them
- skill_id: (optional) a skill of the same plan; 0 clears it`,
}

type EditVocabularyInput struct {
	WordID      int64   `json:"word_id" jsonschema:"Word ID"`
	Term        *string `json:"term,omitempty" jsonschema:"New term"`
	Translation *string `json:"translation,omitempty" jsonschema:"New translation"`
	Notes       *string `json:"notes,omitempty" jsonschema:"New notes; empty string clears them"`
	SkillID     *int64  `json:"skill_id,omitempty" jsonschema:"New skill of the same plan; 0 clears it"`
}

type EditVocabularyOutput struct {
	Word WordResult `json:"word" jsonschema:"Updated word"`
}

func EditVocabulary(ctx context.Context, _ *mcp.CallToolRequest, input EditVocabularyInput) (*mcp.CallToolResult, EditVocabularyOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, EditVocabularyOutput{}, err
	}

	now := time.Now()
	word, err := loadWord(ctx, db, userID, input.WordID, now)
	if err != nil {
		return nil, EditVocabularyOutput{}, err
	}

	updated, err := editVocabulary(ctx, db, userID, word, wordChanges{
		Term:        input.Term,
		Translation: input.Translation,
		Notes:       input.Notes,
		SkillID:     input.SkillID,
	}, now)
	if err != nil {
		return nil, EditVocabularyOutput{}, err
	}

	return nil, EditVocabularyOutput{Word: wordToResult(*updated)}, nil
}
