package domain

import "time"

type Set struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	WorkoutID       int64     `json:"workout_id"`
	ExerciseID      int64     `json:"exercise_id"`
	Reps            int64     `json:"reps,omitempty"`             // 0 for static exercises
	DurationSeconds int64     `json:"duration_seconds,omitempty"` // 0 for rep-based exercises
	WeightKg        float64   `json:"weight_kg,omitempty"`        // 0 for bodyweight
	CreatedAt       time.Time `json:"created_at"`
}

type WorkoutSet struct {
	Workout Workout
	Set     Set
}

type SetWithExercise struct {
	Set
	ExerciseName string `json:"exercise_name"`
}

type SetRecord struct {
	WeightKg  float64
	Reps      int64
	CreatedAt time.Time
}

type VolumeRecord struct {
	Volume    float64
	StartedAt time.Time
}

type PersonalRecords struct {
	MaxWeight *SetRecord
	MaxReps   *SetRecord
	MaxVolume *VolumeRecord
}
