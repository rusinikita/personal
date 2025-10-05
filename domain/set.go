package domain

import "time"

type Set struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	WorkoutID       int64     `json:"workout_id"`
	ExerciseID      int64     `json:"exercise_id"`
	Reps            *int64    `json:"reps"`             // NULL for static exercises
	DurationSeconds *int64    `json:"duration_seconds"` // NULL for rep-based exercises
	WeightKg        *float64  `json:"weight_kg"`        // NULL for bodyweight
	CreatedAt       time.Time `json:"created_at"`
}

type WorkoutSet struct {
	Workout
	Set
}
