package nutrition_stats

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
	"personal/util"
)

var GetNutritionStatsMCPDefinition = mcp.Tool{
	Name: "get_nutrition_stats",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(false),
		Title:           "Get nutrition statistics",
	},
	Description: `Get nutrition statistics for the last meal and last 4 days.

This tool provides two groups of statistics:
1. Last meal: Aggregated nutrition data for the last hour before and including the last consumption record
2. Last 4 days: Daily aggregated nutrition data for the last 4 days (day before yesterday's yesterday, day before yesterday, yesterday, today)

Each statistics group contains:
- Total calories
- Total protein (grams)
- Total fat (grams)
- Total carbohydrates (grams)
- Total weight of food (grams)

The tool automatically:
- Uses Asia/Nicosia timezone for date calculations
- Returns only days with actual consumption data
- Sorts daily statistics chronologically from oldest to newest
- Returns zero values for last meal if no data exists

Use this tool when:
- You want to see recent nutrition intake summary
- You need to analyze daily consumption patterns
- You want to track nutrition trends over the last few days

This is a read-only operation and does not modify any data.`,
}

// GetNutritionStats is the MCP handler for getting nutrition statistics
func GetNutritionStats(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, GetNutritionStatsOutput, error) {
	// Get database from context
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, GetNutritionStatsOutput{}, fmt.Errorf("database not available in context")
	}

	// 1. Load timezone for Asia/Nicosia
	location, err := time.LoadLocation("Asia/Nicosia")
	if err != nil {
		return nil, GetNutritionStatsOutput{}, fmt.Errorf("failed to load timezone: %v", err)
	}

	// 2. Get current time in Asia/Nicosia timezone
	now := time.Now().In(location)

	// 3. Last meal calculation
	var lastMeal domain.NutritionStats

	lastTime, err := db.GetLastConsumptionTime(ctx, DEFAULT_USER_ID)
	if err != nil {
		return nil, GetNutritionStatsOutput{}, fmt.Errorf("failed to get last consumption time: %v", err)
	}

	if lastTime != nil {
		// Create filter for last meal (1 hour before and including last record)
		filter := domain.NutritionStatsFilter{
			UserID:      DEFAULT_USER_ID,
			From:        lastTime.Add(-1 * time.Hour),
			To:          *lastTime,
			Aggregation: domain.AggregationTypeTotal,
		}

		results, err := db.GetNutritionStats(ctx, filter)
		if err != nil {
			return nil, GetNutritionStatsOutput{}, fmt.Errorf("failed to get last meal stats: %v", err)
		}

		if len(results) > 0 {
			lastMeal = results[0]
		}
	}

	// 4. Last 4 days calculation
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	startDate := today.AddDate(0, 0, -3) // 3 days ago

	filter := domain.NutritionStatsFilter{
		UserID:      DEFAULT_USER_ID,
		From:        startDate,
		To:          time.Date(today.Year(), today.Month(), today.Day(), 23, 59, 59, 0, location),
		Timezone:    location,
		Aggregation: domain.AggregationTypeByDay,
	}

	last4Days, err := db.GetNutritionStats(ctx, filter)
	if err != nil {
		return nil, GetNutritionStatsOutput{}, fmt.Errorf("failed to get last 4 days stats: %v", err)
	}

	// 5. Return results
	output := GetNutritionStatsOutput{
		LastMeal:  lastMeal,
		Last4Days: last4Days,
	}

	return nil, output, nil
}
