package tests

import (
	"math"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/create_exercise"
	"personal/action/get_personal_records"
	"personal/domain"
)

func (s *IntegrationTestSuite) TestGetPersonalRecords_ReturnsCorrectRecords() {
	ctx := s.Context()

	_, ex, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Bench Press", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	// Workout 1: 3 sets of 5×80 → volume=1200
	w1ID, err := s.Repo().CreateWorkout(ctx, &domain.Workout{
		UserID: s.UserID(), StartedAt: time.Now().Add(-48 * time.Hour),
	})
	require.NoError(s.T(), err)
	for i := range 3 {
		_, err = s.Repo().CreateSet(ctx, &domain.Set{
			UserID: s.UserID(), WorkoutID: w1ID, ExerciseID: ex.ID,
			Reps: 5, WeightKg: 80, CreatedAt: time.Now().Add(-48*time.Hour + time.Duration(i)*time.Minute),
		})
		require.NoError(s.T(), err)
	}

	// Workout 2: 3×100 (max_weight), 8×80 (max_reps) → volume=940
	w2ID, err := s.Repo().CreateWorkout(ctx, &domain.Workout{
		UserID: s.UserID(), StartedAt: time.Now().Add(-24 * time.Hour),
	})
	require.NoError(s.T(), err)
	_, err = s.Repo().CreateSet(ctx, &domain.Set{
		UserID: s.UserID(), WorkoutID: w2ID, ExerciseID: ex.ID,
		Reps: 3, WeightKg: 100, CreatedAt: time.Now().Add(-24 * time.Hour),
	})
	require.NoError(s.T(), err)
	_, err = s.Repo().CreateSet(ctx, &domain.Set{
		UserID: s.UserID(), WorkoutID: w2ID, ExerciseID: ex.ID,
		Reps: 8, WeightKg: 80, CreatedAt: time.Now().Add(-24*time.Hour + time.Minute),
	})
	require.NoError(s.T(), err)

	_, output, err := get_personal_records.GetPersonalRecords(ctx, nil, get_personal_records.GetPersonalRecordsInput{
		ExerciseID: ex.ID,
	})
	require.NoError(s.T(), err)

	require.NotNil(s.T(), output.MaxWeight)
	assert.Equal(s.T(), 100.0, output.MaxWeight.WeightKg)
	assert.Equal(s.T(), int64(3), output.MaxWeight.Reps)

	require.NotNil(s.T(), output.MaxReps)
	assert.Equal(s.T(), int64(8), output.MaxReps.Reps)
	assert.Equal(s.T(), 80.0, output.MaxReps.WeightKg)

	require.NotNil(s.T(), output.MaxVolume)
	assert.Equal(s.T(), 1200.0, output.MaxVolume.Volume) // workout 1: 3×5×80

	// estimated_1rm: 100 * (1 + 3/30) = 110.0
	assert.InDelta(s.T(), 110.0, output.Estimated1RM, 0.01)
}

func (s *IntegrationTestSuite) TestGetPersonalRecords_NoSets() {
	ctx := s.Context()

	_, ex, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Unused Exercise", EquipmentType: "bodyweight",
	})
	require.NoError(s.T(), err)

	_, output, err := get_personal_records.GetPersonalRecords(ctx, nil, get_personal_records.GetPersonalRecordsInput{
		ExerciseID: ex.ID,
	})
	require.NoError(s.T(), err)
	assert.Nil(s.T(), output.MaxWeight)
	assert.Nil(s.T(), output.MaxReps)
	assert.Nil(s.T(), output.MaxVolume)
	assert.Equal(s.T(), 0.0, output.Estimated1RM)
}

func (s *IntegrationTestSuite) TestGetPersonalRecords_Estimated1RMEpley() {
	ctx := s.Context()

	_, ex, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Squat", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	wID, err := s.Repo().CreateWorkout(ctx, &domain.Workout{
		UserID: s.UserID(), StartedAt: time.Now(),
	})
	require.NoError(s.T(), err)
	_, err = s.Repo().CreateSet(ctx, &domain.Set{
		UserID: s.UserID(), WorkoutID: wID, ExerciseID: ex.ID,
		Reps: 5, WeightKg: 100, CreatedAt: time.Now(),
	})
	require.NoError(s.T(), err)

	_, output, err := get_personal_records.GetPersonalRecords(ctx, nil, get_personal_records.GetPersonalRecordsInput{
		ExerciseID: ex.ID,
	})
	require.NoError(s.T(), err)
	// 100 * (1 + 5/30) ≈ 116.67
	expected := 100.0 * (1 + float64(5)/30)
	assert.InDelta(s.T(), expected, output.Estimated1RM, 0.01)
	_ = math.Round // suppress unused import
}
