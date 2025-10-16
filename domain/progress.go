package domain

import "time"

// ProgressType represents the value scale used for tracking
type ProgressType string

const (
	ProgressTypeMood            ProgressType = "mood"             // Emotional state scale
	ProgressTypeHabitProgress   ProgressType = "habit_progress"   // Adherence to habit scale
	ProgressTypeProjectProgress ProgressType = "project_progress" // Movement towards goal scale
	ProgressTypePromiseState    ProgressType = "promise_state"    // Commitment tracking scale
)

// LifePart represents a life area categorization
type LifePart struct {
	ID          int64     `json:"id" db:"id"`
	UserID      int64     `json:"user_id" db:"user_id"`
	Name        string    `json:"name" db:"name" jsonschema:"Life area name"`
	Description string    `json:"description,omitempty" db:"description" jsonschema:"Life area description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Activity represents a trackable goal or habit
type Activity struct {
	ID            int64        `json:"id" db:"id"`
	UserID        int64        `json:"user_id" db:"user_id"`
	LifePartIDs   []int64      `json:"life_part_ids,omitempty" db:"life_part_ids" jsonschema:"Array of life part IDs this activity belongs to"`
	Name          string       `json:"name" db:"name" jsonschema:"Activity name"`
	Description   string       `json:"description,omitempty" db:"description" jsonschema:"Activity description"`
	ProgressType  ProgressType `json:"progress_type" db:"progress_type" jsonschema:"Progress value scale type (mood|habit_progress|project_progress|promise_state)"`
	FrequencyDays int          `json:"frequency_days" db:"frequency_days" jsonschema:"Check-in frequency in days (1 = daily, 7 = weekly)"`
	StartedAt     time.Time    `json:"started_at" db:"started_at"`
	EndedAt       *time.Time   `json:"ended_at,omitempty" db:"ended_at"` // NULL if active
	CreatedAt     time.Time    `json:"created_at" db:"created_at"`
}

// ActivityPoint represents a single progress point
type ActivityPoint struct {
	ID         int64     `json:"id" db:"id"`
	ActivityID int64     `json:"activity_id" db:"activity_id" jsonschema:"Activity ID this progress point belongs to"`
	UserID     int64     `json:"user_id" db:"user_id"`
	Value      int       `json:"value" db:"value" jsonschema:"Progress value from -2 to +2"`
	HoursLeft  *float64  `json:"hours_left,omitempty" db:"hours_left" jsonschema:"Estimated hours remaining for projects (null if not tracking)"`
	Note       string    `json:"note,omitempty" db:"note" jsonschema:"Optional note about this progress point"`
	ProgressAt time.Time `json:"progress_at" db:"progress_at" jsonschema:"When progress was made (defaults to now if empty)"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// ActivityStats represents calculated statistics for an activity
type ActivityStats struct {
	ActivityID     int64           `json:"activity_id" jsonschema:"Activity ID"`
	Last3Points    []ActivityPoint `json:"last_3_points" jsonschema:"Last 3 progress points"`
	TrendOverall   TrendStats      `json:"trend_overall" jsonschema:"Statistics for all time"`
	TrendLastMonth TrendStats      `json:"trend_last_month" jsonschema:"Statistics for last 30 days"`
	TrendLastWeek  TrendStats      `json:"trend_last_week" jsonschema:"Statistics for last 7 days"`
}

// TrendStats represents aggregated trend data for a time period
type TrendStats struct {
	Count        int     `json:"count" jsonschema:"Number of progress points in this period"`
	Average      float64 `json:"average,omitempty" jsonschema:"Average progress value (0 if no data)"`
	Percentile80 float64 `json:"percentile_80,omitempty" jsonschema:"80th percentile value (0 if no data)"`
}

// ActivityFilter defines query parameters for listing activities
type ActivityFilter struct {
	UserID      int64   `json:"user_id"`
	ActiveOnly  bool    `json:"active_only" jsonschema:"Only return active activities (not finished)"`
	LifePartIDs []int64 `json:"life_part_ids,omitempty" jsonschema:"Filter by life part IDs"`
}

// ProgressFilter defines query parameters for listing progress points
type ProgressFilter struct {
	UserID     int64     `json:"user_id"`
	ActivityID int64     `json:"activity_id,omitempty" jsonschema:"Filter by activity ID (0 = all activities)"`
	From       time.Time `json:"from,omitempty" jsonschema:"Start date filter (empty = no start filter)"`
	To         time.Time `json:"to,omitempty" jsonschema:"End date filter (empty = no end filter)"`
	Limit      int64     `json:"limit,omitempty" jsonschema:"Limit of returned progresses sorted by progress_at DESC"`
}
