package nutrition_stats

import "personal/domain"

const DEFAULT_USER_ID = int64(1)

// GetNutritionStatsOutput is the output structure for the get_nutrition_stats MCP tool
type GetNutritionStatsOutput struct {
	LastMeal  domain.NutritionStats   `json:"last_meal" jsonschema:"Statistics for the last meal (1 hour before and including the last consumption record). Zero values if no data"`
	Last4Days []domain.NutritionStats `json:"last_4_days" jsonschema:"Statistics for last 4 days with data: day before yesterday's yesterday, day before yesterday, yesterday, today. Only includes days that have consumption records, sorted chronologically from oldest to newest"`
}
