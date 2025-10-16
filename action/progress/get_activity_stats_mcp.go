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
	Description: `Get historical statistics and trends for a specific activity to provide context during reflection.

Use this tool when:
- Before asking progress question for an activity ("Last time you were at +1. How are you feeling today?")
- User asks "how am I doing on X?" or "show me my progress"
- Need to show trends and patterns

Input: activity_id (get from get_activity_list)

Returns:
1. LAST 3 POINTS - Most recent progress entries with values, notes, timestamps, hours_left
   - Use to show: "Last 3 times: +2 (yesterday), +1 (3 days ago), 0 (5 days ago)"

2. TREND OVERALL - All-time statistics since activity started
   - Count: total number of check-ins
   - Average: mean progress value (-2 to +2)
   - Percentile 80: you're in top 20% when above this value
   - Use to show: "Overall: 45 check-ins, averaging +1.2"

3. TREND LAST MONTH - Statistics for last 30 days
   - Shows recent patterns
   - Use to show: "This month: 12 check-ins, averaging +1.5 - trending up!"

4. TREND LAST WEEK - Statistics for last 7 days
   - Shows very recent changes
   - Use to show: "This week: 3 check-ins, averaging +1.8"

Example conversation:
"Let's check in on your Daily Mood. Last time you logged +1 (bright). This week you're averaging +1.8. How are you feeling today?"`,
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
