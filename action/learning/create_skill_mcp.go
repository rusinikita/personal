package learning

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var CreateSkillMCPDefinition = mcp.Tool{
	Name:        "create_skill",
	Annotations: &mcp.ToolAnnotations{Title: "Add a skill to a learning plan"},
	Description: `Add a skill (a topic inside a plan, e.g. "present tense", "pods & deployments") to a learning plan. A new skill has 0% mastery.

Inputs:
- learning_plan_id: required
- name: required, unique inside the plan ignoring case`,
}

type CreateSkillInput struct {
	LearningPlanID int64  `json:"learning_plan_id" jsonschema:"Plan ID"`
	Name           string `json:"name" jsonschema:"Skill name"`
}

type CreateSkillOutput struct {
	Skill SkillResult `json:"skill" jsonschema:"Created skill"`
}

func CreateSkill(ctx context.Context, _ *mcp.CallToolRequest, input CreateSkillInput) (*mcp.CallToolResult, CreateSkillOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, CreateSkillOutput{}, err
	}

	plan, err := loadPlan(ctx, db, userID, input.LearningPlanID)
	if err != nil {
		return nil, CreateSkillOutput{}, err
	}

	name, err := validateSkillName(ctx, db, userID, plan.ID, 0, input.Name)
	if err != nil {
		return nil, CreateSkillOutput{}, err
	}

	id, err := db.CreateSkill(ctx, &domain.Skill{UserID: userID, LearningPlanID: plan.ID, Name: name})
	if err != nil {
		return nil, CreateSkillOutput{}, fmt.Errorf("failed to create skill: %w", err)
	}

	skill, err := loadSkill(ctx, db, userID, id)
	if err != nil {
		return nil, CreateSkillOutput{}, err
	}
	skill.MasteryState = domain.ComputeMastery(skill.Scores, skill.LastPracticedAt, time.Now())

	return nil, CreateSkillOutput{Skill: skillToResult(*skill)}, nil
}

// validateSkillName trims the name and checks it is not empty and unique in
// the plan ignoring case, skipping the skill being renamed (selfID).
func validateSkillName(ctx context.Context, db gateways.DB, userID, planID, selfID int64, raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	if len([]rune(name)) > maxTextLength {
		return "", fmt.Errorf("name must be at most %d characters", maxTextLength)
	}

	skills, err := db.ListSkills(ctx, planID, userID)
	if err != nil {
		return "", fmt.Errorf("database error: %w", err)
	}
	for _, s := range skills {
		if s.ID != selfID && strings.EqualFold(s.Name, name) {
			return "", fmt.Errorf("skill %q already exists in this plan", s.Name)
		}
	}

	return name, nil
}
