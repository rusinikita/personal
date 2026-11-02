package learning

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
)

var EditSkillMCPDefinition = mcp.Tool{
	Name:        "edit_skill",
	Annotations: &mcp.ToolAnnotations{Title: "Rename a skill"},
	Description: `Rename a skill.

Inputs:
- skill_id: required
- name: required, unique inside the plan ignoring case`,
}

type EditSkillInput struct {
	SkillID int64  `json:"skill_id" jsonschema:"Skill ID"`
	Name    string `json:"name" jsonschema:"New name"`
}

type EditSkillOutput struct {
	Skill SkillResult `json:"skill" jsonschema:"Renamed skill"`
}

func EditSkill(ctx context.Context, _ *mcp.CallToolRequest, input EditSkillInput) (*mcp.CallToolResult, EditSkillOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, EditSkillOutput{}, err
	}

	skill, err := loadSkill(ctx, db, userID, input.SkillID)
	if err != nil {
		return nil, EditSkillOutput{}, err
	}

	name, err := validateSkillName(ctx, db, userID, skill.LearningPlanID, skill.ID, input.Name)
	if err != nil {
		return nil, EditSkillOutput{}, err
	}
	skill.Name = name

	if err := db.UpdateSkill(ctx, skill); err != nil {
		return nil, EditSkillOutput{}, fmt.Errorf("failed to update skill: %w", err)
	}

	skill, err = loadSkill(ctx, db, userID, skill.ID)
	if err != nil {
		return nil, EditSkillOutput{}, err
	}
	skill.MasteryState = domain.ComputeMastery(skill.Scores, skill.LastPracticedAt, time.Now())

	return nil, EditSkillOutput{Skill: skillToResult(*skill)}, nil
}
