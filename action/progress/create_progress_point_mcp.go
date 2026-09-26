package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var CreateProgressPointMCPDefinition = mcp.Tool{
	Name: "create_progress_point",
	Annotations: &mcp.ToolAnnotations{
		Title: "Create progress point",
	},
	Description: `Log a progress check-in for an activity after user provides their current state.

Use this tool to:
- Record user's progress after asking "How's your [activity] going?"
- Save progress value after interpreting natural language response
- Log notes and context about the progress

Required inputs:
- activity_id: Get from get_activity_list
- value: Integer from -2 to +2 (convert from natural language using get_progress_type_examples)
  -2 = worst state (hell, missing, changed plans, forgot)
  -1 = bad state (dark, rarely, setback, forgot)
   0 = neutral (gray, trying, stuck, remember)
  +1 = good state (bright, mostly doing, moving forward, did something)
  +2 = best state (happy, crushing it, breakthrough, did something)

Optional inputs:
- note: Save user's explanation (e.g., "Feeling great after morning run")
- hours_left: For projects only - estimated hours remaining (e.g., 5.5)
- progress_at: Timestamp for backdating (ISO8601 format, defaults to now)
- executed_step_id: One repeatable step of this activity done in this check-in (from get_step_list). The step stays active — only its execution is recorded. At most one per point; if several repeatable steps were done, log them in separate points

Validation:
- Automatically verifies activity exists and user owns it
- Rejects values outside -2 to +2 range
- Returns error if activity not found
- executed_step_id must be an active repeatable step of the same activity (one_time steps are closed with edit_step instead)

Example flow:
1. User says: "I'm feeling sunny today!"
2. You map "sunny" → mood type → value +2 (from get_progress_type_examples)
3. Call create_progress_point(activity_id=123, value=2, note="Feeling sunny!")
4. Confirm: "Great! Logged your mood as sunny ☀️ (+2)"`,
}

type CreateProgressPointInput struct {
	ActivityID     int64    `json:"activity_id" jsonschema:"Activity ID to log progress for"`
	Value          int      `json:"value" jsonschema:"Progress value from -2 to +2"`
	Note           string   `json:"note,omitempty" jsonschema:"Optional note about this progress point"`
	HoursLeft      *float64 `json:"hours_left,omitempty" jsonschema:"Estimated hours remaining for projects (omit if not tracking)"`
	ProgressAt     string   `json:"progress_at,omitempty" jsonschema:"When progress was made (ISO8601, defaults to now if empty)"`
	ExecutedStepID *int64   `json:"executed_step_id,omitempty" jsonschema:"Active repeatable step of this activity done in this point (optional, at most one); the step stays active"`
}

type CreateProgressPointOutput struct {
	Progress ProgressPoint `json:"progress" jsonschema:"Created progress point"`
}

func CreateProgressPoint(ctx context.Context, _ *mcp.CallToolRequest, input CreateProgressPointInput) (*mcp.CallToolResult, CreateProgressPointOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, CreateProgressPointOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, CreateProgressPointOutput{}, fmt.Errorf("user_id not available in context")
	}

	var progressAt time.Time
	if input.ProgressAt != "" {
		var err error
		progressAt, err = time.Parse(time.RFC3339, input.ProgressAt)
		if err != nil {
			return nil, CreateProgressPointOutput{}, fmt.Errorf("invalid progress_at format, expected RFC3339: %w", err)
		}
	}

	point, err := createProgressPoint(ctx, db, userID, input.ActivityID, input.Value, input.Note, input.HoursLeft, progressAt, input.ExecutedStepID)
	if err != nil {
		return nil, CreateProgressPointOutput{}, err
	}

	output := CreateProgressPointOutput{
		Progress: ProgressPoint{
			ID:             point.ID,
			Value:          point.Value,
			HoursLeft:      point.HoursLeft,
			Note:           point.Note,
			ProgressAt:     point.ProgressAt.Format(time.RFC3339),
			ExecutedStepID: point.ExecutedStepID,
		},
	}

	return nil, output, nil
}

// createProgressPoint validates and writes one activity_progress row —
// shared by the create_progress_point MCP tool above and the "log a point"
// web form on /web/progress/browse/{id} (browse_web.go), so the two entry
// points can't drift out of sync on value-range/ownership/executed-step
// rules. A zero progressAt defaults to now, same as the MCP tool's empty
// progress_at. A non-nil executedStepID must be an active repeatable step of
// the same activity — checked before anything is written.
func createProgressPoint(ctx context.Context, db gateways.DB, userID, activityID int64, value int, note string, hoursLeft *float64, progressAt time.Time, executedStepID *int64) (*domain.ActivityPoint, error) {
	if value < -2 || value > 2 {
		return nil, fmt.Errorf("value must be between -2 and +2")
	}

	activity, err := db.GetActivity(ctx, activityID, userID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if activity == nil {
		return nil, fmt.Errorf("activity not found or unauthorized")
	}

	if executedStepID != nil {
		step, err := db.GetStep(ctx, *executedStepID, userID)
		if err != nil {
			return nil, fmt.Errorf("database error: %w", err)
		}
		if step == nil || step.ActivityID != activityID {
			return nil, fmt.Errorf("executed step not found for this activity")
		}
		if step.Type != domain.StepTypeRepeatable {
			return nil, fmt.Errorf("executed step must be repeatable; close one_time steps instead")
		}
		if step.Status != domain.StepStatusActive {
			return nil, fmt.Errorf("executed step must be active")
		}
	}

	if progressAt.IsZero() {
		progressAt = time.Now()
	}

	point := &domain.ActivityPoint{
		ActivityID:     activityID,
		UserID:         userID,
		Value:          value,
		HoursLeft:      hoursLeft,
		Note:           note,
		ProgressAt:     progressAt,
		ExecutedStepID: executedStepID,
	}

	id, err := db.CreateProgress(ctx, point)
	if err != nil {
		return nil, fmt.Errorf("failed to create progress point: %w", err)
	}
	point.ID = id

	return point, nil
}
