package tests

import (
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/progress"
	"personal/domain"
)

func (s *IntegrationTestSuite) TestGetActivityList_Empty() {
	ctx := s.Context()

	// Call get_activity_list with no activities
	_, output, err := progress.GetActivityList(ctx, nil, progress.GetActivityListInput{})
	require.NoError(s.T(), err)
	assert.Empty(s.T(), output.Activities)
}

func (s *IntegrationTestSuite) TestGetActivityList_WithActivities() {
	ctx := s.Context()
	db := s.Repo()
	userID := s.UserID()
	now := time.Now()

	// Create 3 activities with different frequencies and progress points
	activity1 := &domain.Activity{
		UserID:        userID,
		Name:          "Daily Mood",
		ProgressType:  domain.ProgressTypeMood,
		FrequencyDays: 1,
		StartedAt:     now.AddDate(0, 0, -5),
	}
	id1, err := db.CreateActivity(ctx, activity1)
	require.NoError(s.T(), err)

	// Add progress point 2 days ago -> distance = -1 (overdue)
	point1 := &domain.ActivityPoint{
		ActivityID: id1,
		UserID:     userID,
		Value:      1,
		ProgressAt: now.AddDate(0, 0, -2),
	}
	_, err = db.CreateProgress(ctx, point1)
	require.NoError(s.T(), err)

	activity2 := &domain.Activity{
		UserID:        userID,
		Name:          "Weekly Review",
		ProgressType:  domain.ProgressTypeHabitProgress,
		FrequencyDays: 7,
		StartedAt:     now.AddDate(0, 0, -10),
	}
	id2, err := db.CreateActivity(ctx, activity2)
	require.NoError(s.T(), err)

	// Add progress point 3 days ago -> distance = +4 (due in 4 days)
	point2 := &domain.ActivityPoint{
		ActivityID: id2,
		UserID:     userID,
		Value:      2,
		ProgressAt: now.AddDate(0, 0, -3),
	}
	_, err = db.CreateProgress(ctx, point2)
	require.NoError(s.T(), err)

	activity3 := &domain.Activity{
		UserID:        userID,
		Name:          "Project Alpha",
		ProgressType:  domain.ProgressTypeProjectProgress,
		FrequencyDays: 1,
		StartedAt:     now.AddDate(0, 0, -3),
	}
	id3, err := db.CreateActivity(ctx, activity3)
	require.NoError(s.T(), err)

	// Add progress point 1 day ago -> distance = 0 (due today)
	point3 := &domain.ActivityPoint{
		ActivityID: id3,
		UserID:     userID,
		Value:      1,
		ProgressAt: now.AddDate(0, 0, -1),
	}
	_, err = db.CreateProgress(ctx, point3)
	require.NoError(s.T(), err)

	// Call get_activity_list
	_, output, err := progress.GetActivityList(ctx, nil, progress.GetActivityListInput{})
	require.NoError(s.T(), err)
	require.Len(s.T(), output.Activities, 3)

	// Verify order: sorted by days until check-in (distance)
	// Daily Mood: -2 days + 1 = -1 (overdue by 1 day) - should be 1st
	// Project Alpha: -1 day + 1 = 0 (due today) - should be 2nd
	// Weekly Review: -3 days + 7 = +4 (due in 4 days) - should be 3rd
	assert.Equal(s.T(), id1, output.Activities[0].ID)
	assert.Equal(s.T(), "Daily Mood", output.Activities[0].Name)
	assert.Equal(s.T(), 1, output.Activities[0].FrequencyDays)

	assert.Equal(s.T(), id3, output.Activities[1].ID)
	assert.Equal(s.T(), "Project Alpha", output.Activities[1].Name)
	assert.Equal(s.T(), 1, output.Activities[1].FrequencyDays)

	assert.Equal(s.T(), id2, output.Activities[2].ID)
	assert.Equal(s.T(), "Weekly Review", output.Activities[2].Name)
	assert.Equal(s.T(), 7, output.Activities[2].FrequencyDays)
}

func (s *IntegrationTestSuite) TestGetActivityList_OnlyActive() {
	ctx := s.Context()
	db := s.Repo()
	userID := s.UserID()

	// Create active activity (started 1 day ago to avoid timezone/precision issues)
	activeActivity := &domain.Activity{
		UserID:        userID,
		Name:          "Active Goal",
		ProgressType:  domain.ProgressTypeMood,
		FrequencyDays: 1,
		StartedAt:     time.Now().AddDate(0, 0, -1),
	}
	activeID, err := db.CreateActivity(ctx, activeActivity)
	require.NoError(s.T(), err)

	// Create finished activity
	endedAt := time.Now()
	finishedActivity := &domain.Activity{
		UserID:        userID,
		Name:          "Finished Goal",
		ProgressType:  domain.ProgressTypeMood,
		FrequencyDays: 1,
		StartedAt:     time.Now().AddDate(0, 0, -30),
		EndedAt:       &endedAt,
	}
	_, err = db.CreateActivity(ctx, finishedActivity)
	require.NoError(s.T(), err)

	// Call get_activity_list (should only return active)
	_, output, err := progress.GetActivityList(ctx, nil, progress.GetActivityListInput{})
	require.NoError(s.T(), err)
	require.Len(s.T(), output.Activities, 1)
	assert.Equal(s.T(), activeID, output.Activities[0].ID)
	assert.Equal(s.T(), "Active Goal", output.Activities[0].Name)
}

func (s *IntegrationTestSuite) TestGetProgressTypeExamples() {
	ctx := s.Context()

	// Call get_progress_type_examples
	_, output, err := progress.GetProgressTypeExamples(ctx, nil, progress.ProgressTypeExamplesInput{})
	require.NoError(s.T(), err)

	// Verify all progress types are present
	require.Len(s.T(), output.Examples, 4)

	// Check mood mappings
	moodMapping := output.Examples[0]
	assert.Equal(s.T(), "mood", moodMapping.ProgressType)
	assert.Len(s.T(), moodMapping.Mappings, 3) // weather, light, colors

	// Check habit_progress mappings
	habitMapping := output.Examples[1]
	assert.Equal(s.T(), "habit_progress", habitMapping.ProgressType)
	assert.Len(s.T(), habitMapping.Mappings, 2) // consistency, garden

	// Check project_progress mappings
	projectMapping := output.Examples[2]
	assert.Equal(s.T(), "project_progress", projectMapping.ProgressType)
	assert.Len(s.T(), projectMapping.Mappings, 2) // momentum, journey

	// Check promise_state mappings
	promiseMapping := output.Examples[3]
	assert.Equal(s.T(), "promise_state", promiseMapping.ProgressType)
	assert.Len(s.T(), promiseMapping.Mappings, 2) // awareness, flame
}

func (s *IntegrationTestSuite) TestCreateProgressPoint_Success() {
	ctx := s.Context()
	db := s.Repo()
	userID := s.UserID()

	// Create activity
	activity := &domain.Activity{
		UserID:        userID,
		Name:          "Test Activity",
		ProgressType:  domain.ProgressTypeMood,
		FrequencyDays: 1,
		StartedAt:     time.Now(),
	}
	activityID, err := db.CreateActivity(ctx, activity)
	require.NoError(s.T(), err)

	// Create progress point
	hoursLeft := 5.5
	input := progress.CreateProgressPointInput{
		ActivityID: activityID,
		Value:      2,
		Note:       "Feeling great today",
		HoursLeft:  &hoursLeft,
	}

	_, output, err := progress.CreateProgressPoint(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.Greater(s.T(), output.Progress.ID, int64(0))
	assert.Equal(s.T(), 2, output.Progress.Value)
	assert.Equal(s.T(), "Feeling great today", output.Progress.Note)
	assert.NotNil(s.T(), output.Progress.HoursLeft)
	assert.Equal(s.T(), 5.5, *output.Progress.HoursLeft)

	// Verify in database
	points, err := db.ListProgress(ctx, domain.ProgressFilter{
		UserID:     userID,
		ActivityID: activityID,
		Limit:      10,
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), points, 1)
	assert.Equal(s.T(), 2, points[0].Value)
	assert.Equal(s.T(), "Feeling great today", points[0].Note)
}

func (s *IntegrationTestSuite) TestCreateProgressPoint_InvalidValue() {
	ctx := s.Context()
	db := s.Repo()
	userID := s.UserID()

	// Create activity
	activity := &domain.Activity{
		UserID:        userID,
		Name:          "Test Activity",
		ProgressType:  domain.ProgressTypeMood,
		FrequencyDays: 1,
		StartedAt:     time.Now(),
	}
	activityID, err := db.CreateActivity(ctx, activity)
	require.NoError(s.T(), err)

	// Try to create progress point with invalid value
	input := progress.CreateProgressPointInput{
		ActivityID: activityID,
		Value:      5, // Invalid: must be -2 to +2
	}

	_, _, err = progress.CreateProgressPoint(ctx, nil, input)
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "value must be between -2 and +2")
}

func (s *IntegrationTestSuite) TestGetActivityStats() {
	ctx := s.Context()
	db := s.Repo()
	userID := s.UserID()

	// Create activity
	activity := &domain.Activity{
		UserID:        userID,
		Name:          "Test Activity",
		ProgressType:  domain.ProgressTypeMood,
		FrequencyDays: 1,
		StartedAt:     time.Now().AddDate(0, 0, -60), // 60 days ago
	}
	activityID, err := db.CreateActivity(ctx, activity)
	require.NoError(s.T(), err)

	// Create progress points at different times
	now := time.Now()

	// Point 1: 45 days ago
	point1 := &domain.ActivityPoint{
		ActivityID: activityID,
		UserID:     userID,
		Value:      -1,
		ProgressAt: now.AddDate(0, 0, -45),
	}
	_, err = db.CreateProgress(ctx, point1)
	require.NoError(s.T(), err)

	// Point 2: 20 days ago
	point2 := &domain.ActivityPoint{
		ActivityID: activityID,
		UserID:     userID,
		Value:      1,
		ProgressAt: now.AddDate(0, 0, -20),
	}
	_, err = db.CreateProgress(ctx, point2)
	require.NoError(s.T(), err)

	// Point 3: 5 days ago
	point3 := &domain.ActivityPoint{
		ActivityID: activityID,
		UserID:     userID,
		Value:      2,
		ProgressAt: now.AddDate(0, 0, -5),
	}
	_, err = db.CreateProgress(ctx, point3)
	require.NoError(s.T(), err)

	// Point 4: today
	point4 := &domain.ActivityPoint{
		ActivityID: activityID,
		UserID:     userID,
		Value:      2,
		ProgressAt: now,
	}
	_, err = db.CreateProgress(ctx, point4)
	require.NoError(s.T(), err)

	// Call get_activity_stats
	input := progress.GetActivityStatsInput{
		ActivityID: activityID,
	}
	_, output, err := progress.GetActivityStats(ctx, nil, input)
	require.NoError(s.T(), err)

	// Verify last 3 points (ordered by progress_at DESC)
	require.Len(s.T(), output.Last3Points, 3)
	assert.Equal(s.T(), 2, output.Last3Points[0].Value) // Most recent
	assert.Equal(s.T(), 2, output.Last3Points[1].Value) // 5 days ago
	assert.Equal(s.T(), 1, output.Last3Points[2].Value) // 20 days ago

	// Verify overall stats (all 4 points)
	assert.Equal(s.T(), 4, output.TrendOverall.Count)
	assert.Equal(s.T(), 1.0, output.TrendOverall.Average) // (-1 + 1 + 2 + 2) / 4 = 1.0

	// Verify last month stats (3 points: 20 days, 5 days, today)
	assert.Equal(s.T(), 3, output.TrendLastMonth.Count)

	// Verify last week stats (2 points: 5 days, today)
	assert.Equal(s.T(), 2, output.TrendLastWeek.Count)
}

func (s *IntegrationTestSuite) TestFinishActivity() {
	ctx := s.Context()
	db := s.Repo()
	userID := s.UserID()

	// Create activity
	activity := &domain.Activity{
		UserID:        userID,
		Name:          "Test Project",
		ProgressType:  domain.ProgressTypeProjectProgress,
		FrequencyDays: 1,
		StartedAt:     time.Now(),
	}
	activityID, err := db.CreateActivity(ctx, activity)
	require.NoError(s.T(), err)

	// Finish activity
	input := progress.FinishActivityInput{
		ActivityID: activityID,
	}
	_, output, err := progress.FinishActivity(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.True(s.T(), output.Success)
	assert.Equal(s.T(), "Activity finished", output.Message)

	// Verify activity is finished
	finishedActivity, err := db.GetActivity(ctx, activityID, userID)
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), finishedActivity.EndedAt)

	// Verify it no longer appears in active list
	_, listOutput, err := progress.GetActivityList(ctx, nil, progress.GetActivityListInput{})
	require.NoError(s.T(), err)
	assert.Empty(s.T(), listOutput.Activities)
}

func (s *IntegrationTestSuite) TestFinishActivity_AlreadyFinished() {
	ctx := s.Context()
	db := s.Repo()
	userID := s.UserID()

	// Create and finish activity
	endedAt := time.Now()
	activity := &domain.Activity{
		UserID:        userID,
		Name:          "Test Project",
		ProgressType:  domain.ProgressTypeProjectProgress,
		FrequencyDays: 1,
		StartedAt:     time.Now().AddDate(0, 0, -30),
		EndedAt:       &endedAt,
	}
	activityID, err := db.CreateActivity(ctx, activity)
	require.NoError(s.T(), err)

	// Try to finish again
	input := progress.FinishActivityInput{
		ActivityID: activityID,
	}
	_, _, err = progress.FinishActivity(ctx, nil, input)
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "activity not found or already finished")
}

func (s *IntegrationTestSuite) TestGetActivityList_SortedByDaysUntilCheckIn() {
	ctx := s.Context()
	db := s.Repo()
	userID := s.UserID()
	now := time.Now()

	// Create Activity 1: "Overdue Daily" - freq=1, point 3 days ago → distance=-2 (should be 1st)
	activity1 := &domain.Activity{
		UserID:        userID,
		Name:          "Overdue Daily",
		ProgressType:  domain.ProgressTypeMood,
		FrequencyDays: 1,
		StartedAt:     now.AddDate(0, 0, -10),
	}
	id1, err := db.CreateActivity(ctx, activity1)
	require.NoError(s.T(), err)

	point1 := &domain.ActivityPoint{
		ActivityID: id1,
		UserID:     userID,
		Value:      1,
		ProgressAt: now.AddDate(0, 0, -3),
	}
	_, err = db.CreateProgress(ctx, point1)
	require.NoError(s.T(), err)

	// Create Activity 2: "Overdue Weekly" - freq=7, point 10 days ago → distance=-3 (should be 2nd)
	activity2 := &domain.Activity{
		UserID:        userID,
		Name:          "Overdue Weekly",
		ProgressType:  domain.ProgressTypeHabitProgress,
		FrequencyDays: 7,
		StartedAt:     now.AddDate(0, 0, -20),
	}
	id2, err := db.CreateActivity(ctx, activity2)
	require.NoError(s.T(), err)

	point2 := &domain.ActivityPoint{
		ActivityID: id2,
		UserID:     userID,
		Value:      1,
		ProgressAt: now.AddDate(0, 0, -10),
	}
	_, err = db.CreateProgress(ctx, point2)
	require.NoError(s.T(), err)

	// Create Activity 3: "Due Today" - freq=1, point 1 day ago → distance=0 (should be 3rd)
	activity3 := &domain.Activity{
		UserID:        userID,
		Name:          "Due Today",
		ProgressType:  domain.ProgressTypeMood,
		FrequencyDays: 1,
		StartedAt:     now.AddDate(0, 0, -5),
	}
	id3, err := db.CreateActivity(ctx, activity3)
	require.NoError(s.T(), err)

	point3 := &domain.ActivityPoint{
		ActivityID: id3,
		UserID:     userID,
		Value:      2,
		ProgressAt: now.AddDate(0, 0, -1),
	}
	_, err = db.CreateProgress(ctx, point3)
	require.NoError(s.T(), err)

	// Create Activity 4: "Due Soon" - freq=7, point 5 days ago → distance=+2 (should be 4th)
	activity4 := &domain.Activity{
		UserID:        userID,
		Name:          "Due Soon",
		ProgressType:  domain.ProgressTypeProjectProgress,
		FrequencyDays: 7,
		StartedAt:     now.AddDate(0, 0, -15),
	}
	id4, err := db.CreateActivity(ctx, activity4)
	require.NoError(s.T(), err)

	point4 := &domain.ActivityPoint{
		ActivityID: id4,
		UserID:     userID,
		Value:      1,
		ProgressAt: now.AddDate(0, 0, -5),
	}
	_, err = db.CreateProgress(ctx, point4)
	require.NoError(s.T(), err)

	// Create Activity 5: "Due Later" - freq=14, point 8 days ago → distance=+6 (should be 5th)
	activity5 := &domain.Activity{
		UserID:        userID,
		Name:          "Due Later",
		ProgressType:  domain.ProgressTypePromiseState,
		FrequencyDays: 14,
		StartedAt:     now.AddDate(0, 0, -30),
	}
	id5, err := db.CreateActivity(ctx, activity5)
	require.NoError(s.T(), err)

	point5 := &domain.ActivityPoint{
		ActivityID: id5,
		UserID:     userID,
		Value:      2,
		ProgressAt: now.AddDate(0, 0, -8),
	}
	_, err = db.CreateProgress(ctx, point5)
	require.NoError(s.T(), err)

	// Call get_activity_list
	_, output, err := progress.GetActivityList(ctx, nil, progress.GetActivityListInput{})
	require.NoError(s.T(), err)

	// Verify count
	require.Len(s.T(), output.Activities, 5)

	// Verify sorting order by days until check-in (distance calculation)
	// Distance = (last_point_at + frequency_days) - NOW()
	// Activity 2: (-10 days + 7 days) = -3 days (MOST overdue, comes first)
	// Activity 1: (-3 days + 1 day) = -2 days (overdue)
	// Activity 3: (-1 day + 1 day) = 0 days (due today)
	// Activity 4: (-5 days + 7 days) = +2 days (due soon)
	// Activity 5: (-8 days + 14 days) = +6 days (due later)
	assert.Equal(s.T(), "Overdue Weekly", output.Activities[0].Name)
	assert.Equal(s.T(), 7, output.Activities[0].FrequencyDays)

	assert.Equal(s.T(), "Overdue Daily", output.Activities[1].Name)
	assert.Equal(s.T(), 1, output.Activities[1].FrequencyDays)

	assert.Equal(s.T(), "Due Today", output.Activities[2].Name)
	assert.Equal(s.T(), 1, output.Activities[2].FrequencyDays)

	assert.Equal(s.T(), "Due Soon", output.Activities[3].Name)
	assert.Equal(s.T(), 7, output.Activities[3].FrequencyDays)

	assert.Equal(s.T(), "Due Later", output.Activities[4].Name)
	assert.Equal(s.T(), 14, output.Activities[4].FrequencyDays)
}
