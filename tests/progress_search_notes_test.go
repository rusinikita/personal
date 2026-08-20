package tests

import (
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/progress"
	"personal/domain"
)

func (s *IntegrationTestSuite) TestSearchProgressNotes_ByVariants() {
	ctx := s.Context()
	db := s.Repo()
	userID := s.UserID()

	activityID, err := db.CreateActivity(ctx, &domain.Activity{
		UserID:        userID,
		Name:          "Daily Mood",
		ProgressType:  domain.ProgressTypeMood,
		FrequencyDays: 1,
		StartedAt:     time.Now(),
	})
	require.NoError(s.T(), err)

	// 2 points contain "gym", 1 doesn't
	_, err = db.CreateProgress(ctx, &domain.ActivityPoint{
		ActivityID: activityID,
		UserID:     userID,
		Value:      1,
		Note:       "went to gym today",
		ProgressAt: time.Now(),
	})
	require.NoError(s.T(), err)

	_, err = db.CreateProgress(ctx, &domain.ActivityPoint{
		ActivityID: activityID,
		UserID:     userID,
		Value:      2,
		Note:       "great gym session",
		ProgressAt: time.Now(),
	})
	require.NoError(s.T(), err)

	_, err = db.CreateProgress(ctx, &domain.ActivityPoint{
		ActivityID: activityID,
		UserID:     userID,
		Value:      0,
		Note:       "stayed home",
		ProgressAt: time.Now(),
	})
	require.NoError(s.T(), err)

	_, output, err := progress.SearchProgressNotes(ctx, nil, progress.SearchProgressNotesInput{
		QueryVariants: []string{"gym"},
	})
	require.NoError(s.T(), err)
	assert.Empty(s.T(), output.Error)
	require.Len(s.T(), output.Results, 2)
	for _, r := range output.Results {
		assert.Equal(s.T(), "Daily Mood", r.ActivityName)
		assert.Contains(s.T(), r.Note, "gym")
	}
}

func (s *IntegrationTestSuite) TestSearchProgressNotes_MultiVariantRanking() {
	ctx := s.Context()
	db := s.Repo()
	userID := s.UserID()

	activityID, err := db.CreateActivity(ctx, &domain.Activity{
		UserID:        userID,
		Name:          "Workout",
		ProgressType:  domain.ProgressTypeHabitProgress,
		FrequencyDays: 1,
		StartedAt:     time.Now(),
	})
	require.NoError(s.T(), err)

	// Note A matches both variants: "gym" and "workout"
	_, err = db.CreateProgress(ctx, &domain.ActivityPoint{
		ActivityID: activityID,
		UserID:     userID,
		Value:      2,
		Note:       "did gym workout today",
		ProgressAt: time.Now().Add(-time.Minute),
	})
	require.NoError(s.T(), err)

	// Note B matches only "gym"
	_, err = db.CreateProgress(ctx, &domain.ActivityPoint{
		ActivityID: activityID,
		UserID:     userID,
		Value:      1,
		Note:       "quick gym visit",
		ProgressAt: time.Now(),
	})
	require.NoError(s.T(), err)

	_, output, err := progress.SearchProgressNotes(ctx, nil, progress.SearchProgressNotesInput{
		QueryVariants: []string{"gym", "workout"},
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), output.Results, 2)
	// Note A has match_count=2, should be first
	assert.Equal(s.T(), "did gym workout today", output.Results[0].Note)
	assert.Equal(s.T(), 2, output.Results[0].MatchCount)
	assert.Equal(s.T(), 1, output.Results[1].MatchCount)
}

func (s *IntegrationTestSuite) TestSearchProgressNotes_CaseInsensitive() {
	ctx := s.Context()
	db := s.Repo()
	userID := s.UserID()

	activityID, err := db.CreateActivity(ctx, &domain.Activity{
		UserID:        userID,
		Name:          "Fitness",
		ProgressType:  domain.ProgressTypeHabitProgress,
		FrequencyDays: 1,
		StartedAt:     time.Now(),
	})
	require.NoError(s.T(), err)

	_, err = db.CreateProgress(ctx, &domain.ActivityPoint{
		ActivityID: activityID,
		UserID:     userID,
		Value:      2,
		Note:       "Gym session was great",
		ProgressAt: time.Now(),
	})
	require.NoError(s.T(), err)

	_, output, err := progress.SearchProgressNotes(ctx, nil, progress.SearchProgressNotesInput{
		QueryVariants: []string{"gym"},
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), output.Results, 1)
	assert.Equal(s.T(), "Gym session was great", output.Results[0].Note)
}

func (s *IntegrationTestSuite) TestSearchProgressNotes_FilterByActivity() {
	ctx := s.Context()
	db := s.Repo()
	userID := s.UserID()

	activityID1, err := db.CreateActivity(ctx, &domain.Activity{
		UserID:        userID,
		Name:          "Activity One",
		ProgressType:  domain.ProgressTypeMood,
		FrequencyDays: 1,
		StartedAt:     time.Now(),
	})
	require.NoError(s.T(), err)

	_, err = db.CreateProgress(ctx, &domain.ActivityPoint{
		ActivityID: activityID1,
		UserID:     userID,
		Value:      1,
		Note:       "gym for activity one",
		ProgressAt: time.Now(),
	})
	require.NoError(s.T(), err)

	activityID2, err := db.CreateActivity(ctx, &domain.Activity{
		UserID:        userID,
		Name:          "Activity Two",
		ProgressType:  domain.ProgressTypeMood,
		FrequencyDays: 1,
		StartedAt:     time.Now(),
	})
	require.NoError(s.T(), err)

	_, err = db.CreateProgress(ctx, &domain.ActivityPoint{
		ActivityID: activityID2,
		UserID:     userID,
		Value:      2,
		Note:       "gym for activity two",
		ProgressAt: time.Now(),
	})
	require.NoError(s.T(), err)

	_, output, err := progress.SearchProgressNotes(ctx, nil, progress.SearchProgressNotesInput{
		QueryVariants: []string{"gym"},
		ActivityID:    activityID1,
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), output.Results, 1)
	assert.Equal(s.T(), "Activity One", output.Results[0].ActivityName)
	assert.Equal(s.T(), "gym for activity one", output.Results[0].Note)
}

func (s *IntegrationTestSuite) TestSearchProgressNotes_FilterByDateRange() {
	ctx := s.Context()
	db := s.Repo()
	userID := s.UserID()

	activityID, err := db.CreateActivity(ctx, &domain.Activity{
		UserID:        userID,
		Name:          "Project",
		ProgressType:  domain.ProgressTypeProjectProgress,
		FrequencyDays: 1,
		StartedAt:     time.Now().AddDate(0, 0, -40),
	})
	require.NoError(s.T(), err)

	now := time.Now()

	// Point 1: 30 days ago (outside range)
	_, err = db.CreateProgress(ctx, &domain.ActivityPoint{
		ActivityID: activityID,
		UserID:     userID,
		Value:      -1,
		Note:       "progress note old",
		ProgressAt: now.AddDate(0, 0, -30),
	})
	require.NoError(s.T(), err)

	// Point 2: 10 days ago (inside range: from=-15, to=-5)
	_, err = db.CreateProgress(ctx, &domain.ActivityPoint{
		ActivityID: activityID,
		UserID:     userID,
		Value:      0,
		Note:       "progress note mid",
		ProgressAt: now.AddDate(0, 0, -10),
	})
	require.NoError(s.T(), err)

	// Point 3: today (outside range)
	_, err = db.CreateProgress(ctx, &domain.ActivityPoint{
		ActivityID: activityID,
		UserID:     userID,
		Value:      1,
		Note:       "progress note recent",
		ProgressAt: now,
	})
	require.NoError(s.T(), err)

	_, output, err := progress.SearchProgressNotes(ctx, nil, progress.SearchProgressNotesInput{
		QueryVariants: []string{"progress"},
		From:          now.AddDate(0, 0, -15).Format(time.RFC3339),
		To:            now.AddDate(0, 0, -5).Format(time.RFC3339),
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), output.Results, 1)
	assert.Equal(s.T(), "progress note mid", output.Results[0].Note)
}

func (s *IntegrationTestSuite) TestSearchProgressNotes_FilterByValue() {
	ctx := s.Context()
	db := s.Repo()
	userID := s.UserID()

	activityID, err := db.CreateActivity(ctx, &domain.Activity{
		UserID:        userID,
		Name:          "Stress",
		ProgressType:  domain.ProgressTypeMood,
		FrequencyDays: 1,
		StartedAt:     time.Now(),
	})
	require.NoError(s.T(), err)

	_, err = db.CreateProgress(ctx, &domain.ActivityPoint{
		ActivityID: activityID,
		UserID:     userID,
		Value:      -1,
		Note:       "stress was bad",
		ProgressAt: time.Now(),
	})
	require.NoError(s.T(), err)

	_, err = db.CreateProgress(ctx, &domain.ActivityPoint{
		ActivityID: activityID,
		UserID:     userID,
		Value:      0,
		Note:       "stress was neutral",
		ProgressAt: time.Now(),
	})
	require.NoError(s.T(), err)

	_, err = db.CreateProgress(ctx, &domain.ActivityPoint{
		ActivityID: activityID,
		UserID:     userID,
		Value:      1,
		Note:       "stress was manageable",
		ProgressAt: time.Now(),
	})
	require.NoError(s.T(), err)

	valueMin := 1
	_, output, err := progress.SearchProgressNotes(ctx, nil, progress.SearchProgressNotesInput{
		QueryVariants: []string{"stress"},
		ValueMin:      &valueMin,
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), output.Results, 1)
	assert.Equal(s.T(), "stress was manageable", output.Results[0].Note)
	assert.Equal(s.T(), 1, output.Results[0].Value)
}

func (s *IntegrationTestSuite) TestSearchProgressNotes_NoResults() {
	ctx := s.Context()

	_, output, err := progress.SearchProgressNotes(ctx, nil, progress.SearchProgressNotesInput{
		QueryVariants: []string{"zzznomatch"},
	})
	require.NoError(s.T(), err)
	assert.Empty(s.T(), output.Error)
	assert.Empty(s.T(), output.Results)
}

func (s *IntegrationTestSuite) TestSearchProgressNotes_EmptyVariants() {
	ctx := s.Context()

	_, output, err := progress.SearchProgressNotes(ctx, nil, progress.SearchProgressNotesInput{
		QueryVariants: []string{},
	})
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), output.Error)
}

func (s *IntegrationTestSuite) TestSearchProgressNotes_TooManyVariants() {
	ctx := s.Context()

	_, output, err := progress.SearchProgressNotes(ctx, nil, progress.SearchProgressNotesInput{
		QueryVariants: []string{"a", "b", "c", "d", "e", "f"},
	})
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), output.Error)
}

func (s *IntegrationTestSuite) TestSearchProgressNotes_UserIsolation() {
	ctx := s.Context()
	db := s.Repo()
	user1ID := s.UserID()
	user2ID := s.UserID() + 1

	activityID1, err := db.CreateActivity(ctx, &domain.Activity{
		UserID:        user1ID,
		Name:          "User1 Activity",
		ProgressType:  domain.ProgressTypeMood,
		FrequencyDays: 1,
		StartedAt:     time.Now(),
	})
	require.NoError(s.T(), err)

	_, err = db.CreateProgress(ctx, &domain.ActivityPoint{
		ActivityID: activityID1,
		UserID:     user1ID,
		Value:      1,
		Note:       "gym from user1",
		ProgressAt: time.Now(),
	})
	require.NoError(s.T(), err)

	activityID2, err := db.CreateActivity(ctx, &domain.Activity{
		UserID:        user2ID,
		Name:          "User2 Activity",
		ProgressType:  domain.ProgressTypeMood,
		FrequencyDays: 1,
		StartedAt:     time.Now(),
	})
	require.NoError(s.T(), err)

	_, err = db.CreateProgress(ctx, &domain.ActivityPoint{
		ActivityID: activityID2,
		UserID:     user2ID,
		Value:      1,
		Note:       "gym from user2",
		ProgressAt: time.Now(),
	})
	require.NoError(s.T(), err)

	// ctx has user1ID; should only see user1's note
	_, output, err := progress.SearchProgressNotes(ctx, nil, progress.SearchProgressNotesInput{
		QueryVariants: []string{"gym"},
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), output.Results, 1)
	assert.Equal(s.T(), "gym from user1", output.Results[0].Note)
}
