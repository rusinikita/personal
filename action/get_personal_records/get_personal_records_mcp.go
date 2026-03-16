package get_personal_records

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
)

var MCPDefinition = mcp.Tool{
	Name: "get_personal_records",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint:   true,
		IdempotentHint: true,
		Title:          "Get personal records",
	},
	Description: `Return best results for a given exercise across all time.

Returns:
- max_weight: heaviest single set (weight_kg, reps, date)
- max_reps: most reps in a single set (weight_kg, reps, date)
- max_volume: highest total volume in one workout session (volume = sum of weight×reps, date)
- estimated_1rm: estimated one-rep max using Epley formula: weight × (1 + reps/30)

All fields are null if no sets have been logged for this exercise.
Only sets with reps > 0 and weight_kg > 0 count toward weight/volume metrics.`,
}

type GetPersonalRecordsInput struct {
	ExerciseID int64 `json:"exercise_id" jsonschema:"Exercise ID"`
}

type SetRecordOutput struct {
	WeightKg float64 `json:"weight_kg"`
	Reps     int64   `json:"reps"`
	Date     string  `json:"date"`
}

type VolumeRecordOutput struct {
	Volume float64 `json:"volume"`
	Date   string  `json:"date"`
}

type GetPersonalRecordsOutput struct {
	MaxWeight    *SetRecordOutput    `json:"max_weight"`
	MaxReps      *SetRecordOutput    `json:"max_reps"`
	MaxVolume    *VolumeRecordOutput `json:"max_volume"`
	Estimated1RM float64             `json:"estimated_1rm"`
}

func GetPersonalRecords(ctx context.Context, _ *mcp.CallToolRequest, input GetPersonalRecordsInput) (*mcp.CallToolResult, GetPersonalRecordsOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, GetPersonalRecordsOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, GetPersonalRecordsOutput{}, fmt.Errorf("user_id not available in context")
	}

	if input.ExerciseID == 0 {
		return nil, GetPersonalRecordsOutput{}, fmt.Errorf("exercise_id is required")
	}

	records, err := db.GetPersonalRecords(ctx, userID, input.ExerciseID)
	if err != nil {
		return nil, GetPersonalRecordsOutput{}, fmt.Errorf("failed to get personal records: %w", err)
	}

	output := GetPersonalRecordsOutput{}

	if records.MaxWeight != nil {
		output.MaxWeight = &SetRecordOutput{
			WeightKg: records.MaxWeight.WeightKg,
			Reps:     records.MaxWeight.Reps,
			Date:     records.MaxWeight.CreatedAt.Format("2006-01-02"),
		}
		output.Estimated1RM = records.MaxWeight.WeightKg * (1 + float64(records.MaxWeight.Reps)/30)
	}

	if records.MaxReps != nil {
		output.MaxReps = &SetRecordOutput{
			WeightKg: records.MaxReps.WeightKg,
			Reps:     records.MaxReps.Reps,
			Date:     records.MaxReps.CreatedAt.Format("2006-01-02"),
		}
	}

	if records.MaxVolume != nil {
		output.MaxVolume = &VolumeRecordOutput{
			Volume: records.MaxVolume.Volume,
			Date:   records.MaxVolume.StartedAt.Format("2006-01-02"),
		}
	}

	return nil, output, nil
}
