package learning

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var AddVocabularyMCPDefinition = mcp.Tool{
	Name:        "add_vocabulary",
	Annotations: &mcp.ToolAnnotations{Title: "Add words to a plan"},
	Description: `Add 1-50 words to a plan's vocabulary. A new word has 0% mastery and no reviews; plan its first review with plan_vocabulary.
All or nothing: a term already in the plan or repeated in the call (ignoring case) fails the whole call, and the error lists those terms.

Inputs:
- learning_plan_id: required
- words: each with term and translation (required), notes and skill_id (optional, a skill of the same plan)`,
}

type WordInput struct {
	Term        string  `json:"term" jsonschema:"Term"`
	Translation string  `json:"translation" jsonschema:"Translation"`
	Notes       *string `json:"notes,omitempty" jsonschema:"Notes"`
	SkillID     *int64  `json:"skill_id,omitempty" jsonschema:"Optional topic skill of the same plan"`
}

type AddVocabularyInput struct {
	LearningPlanID int64       `json:"learning_plan_id" jsonschema:"Plan ID"`
	Words          []WordInput `json:"words" jsonschema:"1-50 words"`
}

type AddVocabularyOutput struct {
	Words []WordResult `json:"words" jsonschema:"Added words, in input order"`
}

func AddVocabulary(ctx context.Context, _ *mcp.CallToolRequest, input AddVocabularyInput) (*mcp.CallToolResult, AddVocabularyOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, AddVocabularyOutput{}, err
	}

	inputs := make([]wordInput, len(input.Words))
	for i, w := range input.Words {
		inputs[i] = wordInput{Term: w.Term, Translation: w.Translation, Notes: w.Notes, SkillID: w.SkillID}
	}

	words, err := addVocabulary(ctx, db, userID, input.LearningPlanID, inputs, time.Now())
	if err != nil {
		return nil, AddVocabularyOutput{}, err
	}

	return nil, AddVocabularyOutput{Words: wordsToResults(words)}, nil
}
