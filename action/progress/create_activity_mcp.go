package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var CreateActivityMCPDefinition = mcp.Tool{
	Name: "create_activity",
	Annotations: &mcp.ToolAnnotations{
		Title: "Create activity",
	},
	Description: `Create a new activity for progress tracking.

Use this tool when:
- User wants to start tracking a new project, habit, goal, or promise
- User describes something new they want to monitor periodically

Required inputs:
- name: Activity name
- progress_type: One of mood, habit_progress, project_progress, promise_state
- frequency_days: How often to check in (1 = daily, 7 = weekly)

Optional inputs:
- description: Brief description of the activity
- life_part_ids: Array of life area IDs this belongs to
- started_at: When tracking started (ISO8601, defaults to now)

Example:
User: "I want to track my user outreach project"
You: [Call create_activity(name="User Outreach", progress_type="project_progress", frequency_days=1)]`,
}

type CreateActivityInput struct {
	Name          string  `json:"name" jsonschema:"Activity name"`
	ProgressType  string  `json:"progress_type" jsonschema:"Progress type: mood|habit_progress|project_progress|promise_state"`
	FrequencyDays int     `json:"frequency_days" jsonschema:"Check-in frequency in days (1 = daily, 7 = weekly)"`
	Description   string  `json:"description,omitempty" jsonschema:"Activity description"`
	LifePartIDs   []int64 `json:"life_part_ids,omitempty" jsonschema:"Life area IDs this activity belongs to"`
	StartedAt     string  `json:"started_at,omitempty" jsonschema:"When tracking started (ISO8601, defaults to now)"`
}

type ActivityResult struct {
	ID            int64   `json:"id" jsonschema:"Activity ID"`
	Name          string  `json:"name" jsonschema:"Activity name"`
	Description   string  `json:"description,omitempty" jsonschema:"Activity description"`
	ProgressType  string  `json:"progress_type" jsonschema:"Progress type"`
	FrequencyDays int     `json:"frequency_days" jsonschema:"Check-in frequency in days"`
	LifePartIDs   []int64 `json:"life_part_ids" jsonschema:"Life area IDs"`
	StartedAt     string  `json:"started_at" jsonschema:"When tracking started (ISO8601)"`
	CreatedAt     string  `json:"created_at" jsonschema:"When activity was created (ISO8601)"`
}

type CreateActivityOutput struct {
	Activity ActivityResult `json:"activity" jsonschema:"Created activity"`
}

var validProgressTypes = map[string]bool{
	"mood":             true,
	"habit_progress":   true,
	"project_progress": true,
	"promise_state":    true,
}

func CreateActivity(ctx context.Context, _ *mcp.CallToolRequest, input CreateActivityInput) (*mcp.CallToolResult, CreateActivityOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, CreateActivityOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, CreateActivityOutput{}, fmt.Errorf("user_id not available in context")
	}

	if input.Name == "" {
		return nil, CreateActivityOutput{}, fmt.Errorf("name is required")
	}
	if !validProgressTypes[input.ProgressType] {
		return nil, CreateActivityOutput{}, fmt.Errorf("invalid progress_type: must be one of mood, habit_progress, project_progress, promise_state")
	}
	if input.FrequencyDays < 1 {
		return nil, CreateActivityOutput{}, fmt.Errorf("frequency_days must be at least 1")
	}

	startedAt := time.Now()
	if input.StartedAt != "" {
		var err error
		startedAt, err = time.Parse(time.RFC3339, input.StartedAt)
		if err != nil {
			return nil, CreateActivityOutput{}, fmt.Errorf("invalid started_at format, expected RFC3339: %w", err)
		}
	}

	lifePartIDs := input.LifePartIDs
	if lifePartIDs == nil {
		lifePartIDs = []int64{}
	}

	activity := &domain.Activity{
		UserID:        userID,
		Name:          input.Name,
		Description:   input.Description,
		ProgressType:  domain.ProgressType(input.ProgressType),
		FrequencyDays: input.FrequencyDays,
		LifePartIDs:   lifePartIDs,
		StartedAt:     startedAt,
	}

	id, err := db.CreateActivity(ctx, activity)
	if err != nil {
		return nil, CreateActivityOutput{}, fmt.Errorf("failed to create activity: %w", err)
	}

	created, err := db.GetActivity(ctx, id, userID)
	if err != nil {
		return nil, CreateActivityOutput{}, fmt.Errorf("failed to fetch created activity: %w", err)
	}

	return nil, CreateActivityOutput{Activity: activityToResult(created)}, nil
}

func activityToResult(a *domain.Activity) ActivityResult {
	lifePartIDs := a.LifePartIDs
	if lifePartIDs == nil {
		lifePartIDs = []int64{}
	}
	return ActivityResult{
		ID:            a.ID,
		Name:          a.Name,
		Description:   a.Description,
		ProgressType:  string(a.ProgressType),
		FrequencyDays: a.FrequencyDays,
		LifePartIDs:   lifePartIDs,
		StartedAt:     a.StartedAt.Format(time.RFC3339),
		CreatedAt:     a.CreatedAt.Format(time.RFC3339),
	}
}
