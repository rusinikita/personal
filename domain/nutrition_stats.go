package domain

import "time"

// AggregationType defines how nutrition stats should be aggregated
type AggregationType string

const (
	AggregationTypeTotal AggregationType = "total"  // Sum all records into one object
	AggregationTypeByDay AggregationType = "by_day" // Group by date, array of objects per day
)

// NutritionStatsFilter defines the parameters for querying nutrition statistics
type NutritionStatsFilter struct {
	UserID      int64           // User ID to filter by
	From        time.Time       // Start of time window
	To          time.Time       // End of time window
	Timezone    *time.Location  // Timezone for correct GROUP BY date
	Aggregation AggregationType // Type of aggregation
}

// NutritionStats represents aggregated nutrition data for a time period
type NutritionStats struct {
	PeriodStart   time.Time `json:"period_start"`
	PeriodEnd     time.Time `json:"period_end"`
	TotalCalories float64   `json:"total_calories"`
	TotalProtein  float64   `json:"total_protein"`
	TotalFat      float64   `json:"total_fat"`
	TotalCarbs    float64   `json:"total_carbs"`
	TotalWeight   float64   `json:"total_weight"`
}
