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

// ActivityStatus is the explicit lifecycle state of an activity — source of
// truth going forward, instead of inferring state purely from
// started_at/ended_at.
type ActivityStatus string

const (
	ActivityStatusActive   ActivityStatus = "active"
	ActivityStatusPaused   ActivityStatus = "paused"
	ActivityStatusFinished ActivityStatus = "finished" // goal reached
	ActivityStatusDropped  ActivityStatus = "dropped"  // abandoned
)

// Activity represents a trackable goal or habit
type Activity struct {
	ID            int64          `json:"id" db:"id"`
	UserID        int64          `json:"user_id" db:"user_id"`
	LifePartIDs   []int64        `json:"life_part_ids,omitempty" db:"life_part_ids" jsonschema:"Array of life part IDs this activity belongs to"`
	Name          string         `json:"name" db:"name" jsonschema:"Activity name"`
	Description   string         `json:"description,omitempty" db:"description" jsonschema:"Activity description"`
	ProgressType  ProgressType   `json:"progress_type" db:"progress_type" jsonschema:"Progress value scale type (mood|habit_progress|project_progress|promise_state)"`
	Status        ActivityStatus `json:"status" db:"status" jsonschema:"Lifecycle status (active|paused|finished|dropped)"`
	DeferredUntil *time.Time     `json:"deferred_until,omitempty" db:"deferred_until" jsonschema:"When a paused activity should resume (null unless paused with a resume date)"`
	FrequencyDays int            `json:"frequency_days" db:"frequency_days" jsonschema:"Check-in frequency in days (1 = daily, 7 = weekly)"`
	StartedAt     time.Time      `json:"started_at" db:"started_at"`
	EndedAt       *time.Time     `json:"ended_at,omitempty" db:"ended_at"` // NULL unless status is finished or dropped
	LastPointAt   *time.Time     `json:"last_point_at,omitempty" db:"last_point_at"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
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
	UserID       int64            `json:"user_id"`
	Statuses     []ActivityStatus `json:"statuses,omitempty" jsonschema:"Only return activities whose status is one of these (empty = no status filter); replaces the old ActiveOnly/PausedOnly booleans — callers pass e.g. []ActivityStatus{ActivityStatusActive} or {ActivityStatusFinished, ActivityStatusDropped}"`
	FutureOnly   bool             `json:"future_only,omitempty" jsonschema:"Only return not-yet-started activities (started_at in the future) instead of started_at<=NOW(); status is still filtered separately via Statuses"`
	ProgressType ProgressType     `json:"progress_type,omitempty" jsonschema:"Only return activities of this progress_type (empty = all types)"`
	LifePartIDs  []int64          `json:"life_part_ids,omitempty" jsonschema:"Filter by life part IDs"`
	Limit        int64            `json:"limit,omitempty" jsonschema:"Page size for browse-view pagination (0 = no limit)"`
	Offset       int64            `json:"offset,omitempty" jsonschema:"Row offset for browse-view pagination"`
}

// ProgressFilter defines query parameters for listing progress points
type ProgressFilter struct {
	UserID     int64     `json:"user_id"`
	ActivityID int64     `json:"activity_id,omitempty" jsonschema:"Filter by activity ID (0 = all activities)"`
	From       time.Time `json:"from,omitempty" jsonschema:"Start date filter (empty = no start filter)"`
	To         time.Time `json:"to,omitempty" jsonschema:"End date filter (empty = no end filter)"`
	Limit      int64     `json:"limit,omitempty" jsonschema:"Limit of returned progresses sorted by progress_at DESC"`
	Offset     int64     `json:"offset,omitempty" jsonschema:"Row offset for browse-view drill-down pagination"`
}

// ActivityPointWithActivity is ActivityPoint enriched with the parent activity name
type ActivityPointWithActivity struct {
	ActivityPoint
	ActivityName string `json:"activity_name" db:"activity_name"`
}

// ProgressNoteSearchFilter defines parameters for a single-variant note ILIKE search
type ProgressNoteSearchFilter struct {
	UserID     int64     `json:"user_id"`
	Query      string    `json:"query"`
	ActivityID int64     `json:"activity_id,omitempty"` // 0 = all activities
	From       time.Time `json:"from,omitempty"`
	To         time.Time `json:"to,omitempty"`
	ValueMin   *int      `json:"value_min,omitempty"`
	ValueMax   *int      `json:"value_max,omitempty"`
}
