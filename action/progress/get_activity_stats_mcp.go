package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var GetActivityStatsMCPDefinition = mcp.Tool{
	Name: "get_activity_stats",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint: true,
		Title:        "Get activity statistics",
	},
	Description: `Returns comprehensive statistics for a specific activity.

Fetches last 3 points, calculates trend averages and 80th percentiles for three time windows:
- Overall (from started_at)
- Last 30 days
- Last 7 days

Returns NULL for periods with no data.`,
}

type GetActivityStatsInput struct {
	ActivityID int64 `json:"activity_id" jsonschema:"Activity ID to get statistics for"`
}

type ProgressPoint struct {
	ID         int64    `json:"id" jsonschema:"Progress point ID"`
	Value      int      `json:"value" jsonschema:"Progress value from -2 to +2"`
	HoursLeft  *float64 `json:"hours_left,omitempty" jsonschema:"Estimated hours remaining"`
	Note       string   `json:"note,omitempty" jsonschema:"Note about this progress point"`
	ProgressAt string   `json:"progress_at" jsonschema:"When progress was made (ISO8601)"`
}

type TrendStatsOutput struct {
	Count        int     `json:"count" jsonschema:"Number of progress points in this period"`
	Average      float64 `json:"average,omitempty" jsonschema:"Average progress value (0 if no data)"`
	Percentile80 float64 `json:"percentile_80,omitempty" jsonschema:"80th percentile value (0 if no data)"`
}

type GetActivityStatsOutput struct {
	ActivityID     int64            `json:"activity_id" jsonschema:"Activity ID"`
	Last3Points    []ProgressPoint  `json:"last_3_points" jsonschema:"Last 3 progress points"`
	TrendOverall   TrendStatsOutput `json:"trend_overall" jsonschema:"Statistics for all time"`
	TrendLastMonth TrendStatsOutput `json:"trend_last_month" jsonschema:"Statistics for last 30 days"`
	TrendLastWeek  TrendStatsOutput `json:"trend_last_week" jsonschema:"Statistics for last 7 days"`
}

func GetActivityStats(ctx context.Context, _ *mcp.CallToolRequest, input GetActivityStatsInput) (*mcp.CallToolResult, GetActivityStatsOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, GetActivityStatsOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, GetActivityStatsOutput{}, fmt.Errorf("user_id not available in context")
	}

	// Get activity to verify ownership and get started_at
	activity, err := db.GetActivity(ctx, input.ActivityID, userID)
	if err != nil {
		return nil, GetActivityStatsOutput{}, fmt.Errorf("database error: %w", err)
	}
	if activity == nil {
		return nil, GetActivityStatsOutput{}, fmt.Errorf("activity not found or unauthorized")
	}

	// Get last 3 points
	last3Filter := domain.ProgressFilter{
		UserID:     userID,
		ActivityID: input.ActivityID,
		Limit:      3,
	}
	last3Points, err := db.ListProgress(ctx, last3Filter)
	if err != nil {
		return nil, GetActivityStatsOutput{}, fmt.Errorf("failed to get last 3 points: %w", err)
	}

	// Convert to output format
	output := GetActivityStatsOutput{
		ActivityID:  input.ActivityID,
		Last3Points: make([]ProgressPoint, 0, len(last3Points)),
	}

	for _, p := range last3Points {
		point := ProgressPoint{
			ID:         p.ID,
			Value:      p.Value,
			HoursLeft:  p.HoursLeft,
			Note:       p.Note,
			ProgressAt: p.ProgressAt.Format(time.RFC3339),
		}
		output.Last3Points = append(output.Last3Points, point)
	}

	// Calculate trend stats
	now := time.Now()

	// Overall stats (from started_at to now)
	overallStats, err := db.GetTrendStats(ctx, input.ActivityID, userID, activity.StartedAt, now)
	if err != nil {
		return nil, GetActivityStatsOutput{}, fmt.Errorf("failed to get overall stats: %w", err)
	}
	output.TrendOverall = TrendStatsOutput{
		Count:        overallStats.Count,
		Average:      overallStats.Average,
		Percentile80: overallStats.Percentile80,
	}

	// Last 30 days stats
	lastMonthStats, err := db.GetTrendStats(ctx, input.ActivityID, userID, now.AddDate(0, 0, -30), now)
	if err != nil {
		return nil, GetActivityStatsOutput{}, fmt.Errorf("failed to get last month stats: %w", err)
	}
	output.TrendLastMonth = TrendStatsOutput{
		Count:        lastMonthStats.Count,
		Average:      lastMonthStats.Average,
		Percentile80: lastMonthStats.Percentile80,
	}

	// Last 7 days stats
	lastWeekStats, err := db.GetTrendStats(ctx, input.ActivityID, userID, now.AddDate(0, 0, -7), now)
	if err != nil {
		return nil, GetActivityStatsOutput{}, fmt.Errorf("failed to get last week stats: %w", err)
	}
	output.TrendLastWeek = TrendStatsOutput{
		Count:        lastWeekStats.Count,
		Average:      lastWeekStats.Average,
		Percentile80: lastWeekStats.Percentile80,
	}

	return nil, output, nil
}
