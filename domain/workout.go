package domain

import "time"

type Workout struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"` // NULL means active
}
