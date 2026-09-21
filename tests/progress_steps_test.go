package tests

// Covers the "Steps" feature from docs/functions/progress-spec.md (backlog:
// steps — concrete, short-horizon next-actions owned by an activity): the
// create_step/edit_step/delete_step/get_step_list MCP tools, plus their web
// surfacing in the browse list, drill-down, and "log a point" form.

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/progress"
	"personal/domain"
	"personal/gateways"
)

// --- create_step --------------------------------------------------------

func (s *IntegrationTestSuite) TestCreateStep_Success() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	activityID := s.createActivity(ctx, "Move apartment", domain.ProgressTypeProjectProgress, time.Now().AddDate(0, 0, -5))

	_, output, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{
		ActivityID: activityID,
		Name:       "Book moving truck",
		Type:       "one_time",
	})
	require.NoError(s.T(), err)
	assert.Greater(s.T(), output.Step.ID, int64(0))
	assert.Equal(s.T(), activityID, output.Step.ActivityID)
	assert.Equal(s.T(), "Book moving truck", output.Step.Name)
	assert.Equal(s.T(), "one_time", output.Step.Type)
	assert.Equal(s.T(), "active", output.Step.Status)
	assert.Empty(s.T(), output.Step.ClosedAt)

	steps, err := s.Repo().ListSteps(ctx, domain.StepFilter{UserID: userID, ActivityID: activityID})
	require.NoError(s.T(), err)
	require.Len(s.T(), steps, 1)
	assert.Nil(s.T(), steps[0].CreatedByProgressPointID, "create_step never sets created_by_progress_point_id — only the web form does")
}

func (s *IntegrationTestSuite) TestCreateStep_InvalidType() {
	ctx := s.Context()
	activityID := s.createActivity(ctx, "Move apartment", domain.ProgressTypeProjectProgress, time.Now().AddDate(0, 0, -5))

	_, _, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: activityID, Name: "x", Type: "bogus"})
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "invalid type")
}

func (s *IntegrationTestSuite) TestCreateStep_EmptyName() {
	ctx := s.Context()
	activityID := s.createActivity(ctx, "Move apartment", domain.ProgressTypeProjectProgress, time.Now().AddDate(0, 0, -5))

	_, _, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: activityID, Name: "   ", Type: "one_time"})
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "name is required")
}

func (s *IntegrationTestSuite) TestCreateStep_ActivityNotFound() {
	ctx := s.Context()

	_, _, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: 999999999, Name: "x", Type: "one_time"})
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "activity not found")
}

// --- edit_step -----------------------------------------------------------

func (s *IntegrationTestSuite) TestEditStep_Rename() {
	ctx := s.Context()
	activityID := s.createActivity(ctx, "Move apartment", domain.ProgressTypeProjectProgress, time.Now().AddDate(0, 0, -5))
	_, created, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: activityID, Name: "Old name", Type: "one_time"})
	require.NoError(s.T(), err)

	newName := "New name"
	_, output, err := progress.EditStep(ctx, nil, progress.EditStepInput{StepID: created.Step.ID, Name: &newName})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "New name", output.Step.Name)
}

func (s *IntegrationTestSuite) TestEditStep_CloseSetsClosedAt() {
	ctx := s.Context()
	activityID := s.createActivity(ctx, "Move apartment", domain.ProgressTypeProjectProgress, time.Now().AddDate(0, 0, -5))
	_, created, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: activityID, Name: "Book truck", Type: "one_time"})
	require.NoError(s.T(), err)

	status := "finished"
	_, output, err := progress.EditStep(ctx, nil, progress.EditStepInput{StepID: created.Step.ID, Status: &status})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "finished", output.Step.Status)
	assert.NotEmpty(s.T(), output.Step.ClosedAt)
}

func (s *IntegrationTestSuite) TestEditStep_ReopenClearsClosedAt() {
	ctx := s.Context()
	activityID := s.createActivity(ctx, "Move apartment", domain.ProgressTypeProjectProgress, time.Now().AddDate(0, 0, -5))
	_, created, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: activityID, Name: "Book truck", Type: "one_time"})
	require.NoError(s.T(), err)

	finished := "finished"
	_, _, err = progress.EditStep(ctx, nil, progress.EditStepInput{StepID: created.Step.ID, Status: &finished})
	require.NoError(s.T(), err)

	active := "active"
	_, output, err := progress.EditStep(ctx, nil, progress.EditStepInput{StepID: created.Step.ID, Status: &active})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "active", output.Step.Status)
	assert.Empty(s.T(), output.Step.ClosedAt)
}

func (s *IntegrationTestSuite) TestEditStep_LinksCompletedByProgressPoint() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	activityID := s.createActivity(ctx, "Move apartment", domain.ProgressTypeProjectProgress, time.Now().AddDate(0, 0, -5))
	_, created, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: activityID, Name: "Book truck", Type: "one_time"})
	require.NoError(s.T(), err)

	pointID, err := s.Repo().CreateProgress(ctx, &domain.ActivityPoint{ActivityID: activityID, UserID: userID, Value: 1, ProgressAt: time.Now()})
	require.NoError(s.T(), err)

	status := "finished"
	_, output, err := progress.EditStep(ctx, nil, progress.EditStepInput{StepID: created.Step.ID, Status: &status, CompletedByProgressPointID: &pointID})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "finished", output.Step.Status)

	step, err := s.Repo().GetStep(ctx, created.Step.ID, userID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), step.CompletedByProgressPointID)
	assert.Equal(s.T(), pointID, *step.CompletedByProgressPointID)
}

func (s *IntegrationTestSuite) TestEditStep_NotFound() {
	ctx := s.Context()
	name := "x"
	_, _, err := progress.EditStep(ctx, nil, progress.EditStepInput{StepID: 999999999, Name: &name})
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "step not found")
}

func (s *IntegrationTestSuite) TestEditStep_NoFieldsProvided() {
	ctx := s.Context()
	activityID := s.createActivity(ctx, "Move apartment", domain.ProgressTypeProjectProgress, time.Now().AddDate(0, 0, -5))
	_, created, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: activityID, Name: "Book truck", Type: "one_time"})
	require.NoError(s.T(), err)

	_, _, err = progress.EditStep(ctx, nil, progress.EditStepInput{StepID: created.Step.ID})
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "at least one field")
}

// --- delete_step -----------------------------------------------------------

func (s *IntegrationTestSuite) TestDeleteStep_Success() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	activityID := s.createActivity(ctx, "Move apartment", domain.ProgressTypeProjectProgress, time.Now().AddDate(0, 0, -5))
	_, created, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: activityID, Name: "Book truck", Type: "one_time"})
	require.NoError(s.T(), err)

	_, output, err := progress.DeleteStep(ctx, nil, progress.DeleteStepInput{StepID: created.Step.ID})
	require.NoError(s.T(), err)
	assert.True(s.T(), output.Success)

	step, err := s.Repo().GetStep(ctx, created.Step.ID, userID)
	require.NoError(s.T(), err)
	assert.Nil(s.T(), step)
}

func (s *IntegrationTestSuite) TestDeleteStep_NotFound() {
	ctx := s.Context()
	_, _, err := progress.DeleteStep(ctx, nil, progress.DeleteStepInput{StepID: 999999999})
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "step not found")
}

// --- get_step_list -----------------------------------------------------------

func (s *IntegrationTestSuite) TestGetStepList_DefaultsToActive() {
	ctx := s.Context()
	activityID := s.createActivity(ctx, "Move apartment", domain.ProgressTypeProjectProgress, time.Now().AddDate(0, 0, -5))
	_, activeStep, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: activityID, Name: "Book truck", Type: "one_time"})
	require.NoError(s.T(), err)
	_, doneStep, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: activityID, Name: "Pack boxes", Type: "repeatable"})
	require.NoError(s.T(), err)

	finished := "finished"
	_, _, err = progress.EditStep(ctx, nil, progress.EditStepInput{StepID: doneStep.Step.ID, Status: &finished})
	require.NoError(s.T(), err)

	_, output, err := progress.GetStepList(ctx, nil, progress.GetStepListInput{})
	require.NoError(s.T(), err)
	require.Len(s.T(), output.Steps, 1)
	assert.Equal(s.T(), activeStep.Step.ID, output.Steps[0].ID)
	assert.Equal(s.T(), "Move apartment", output.Steps[0].ActivityName)
}

func (s *IntegrationTestSuite) TestGetStepList_FilterByActivity() {
	ctx := s.Context()
	activity1 := s.createActivity(ctx, "Move apartment", domain.ProgressTypeProjectProgress, time.Now().AddDate(0, 0, -5))
	activity2 := s.createActivity(ctx, "Learn Spanish", domain.ProgressTypeHabitProgress, time.Now().AddDate(0, 0, -5))
	_, _, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: activity1, Name: "Book truck", Type: "one_time"})
	require.NoError(s.T(), err)
	_, _, err = progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: activity2, Name: "Practice", Type: "repeatable"})
	require.NoError(s.T(), err)

	_, output, err := progress.GetStepList(ctx, nil, progress.GetStepListInput{ActivityID: activity1})
	require.NoError(s.T(), err)
	require.Len(s.T(), output.Steps, 1)
	assert.Equal(s.T(), "Book truck", output.Steps[0].Name)
}

func (s *IntegrationTestSuite) TestGetStepList_ExcludesStepsOfInactiveActivity() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	activityID := s.createActivity(ctx, "Move apartment", domain.ProgressTypeProjectProgress, time.Now().AddDate(0, 0, -5))
	_, _, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: activityID, Name: "Book truck", Type: "one_time"})
	require.NoError(s.T(), err)

	s.finishActivity(ctx, activityID, userID, time.Now())

	_, output, err := progress.GetStepList(ctx, nil, progress.GetStepListInput{ActivityID: activityID})
	require.NoError(s.T(), err)
	assert.Empty(s.T(), output.Steps, "a step belonging to a finished activity must never be returned")
}

func (s *IntegrationTestSuite) TestGetStepList_FilterByStatusFinished() {
	ctx := s.Context()
	activityID := s.createActivity(ctx, "Move apartment", domain.ProgressTypeProjectProgress, time.Now().AddDate(0, 0, -5))
	_, created, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: activityID, Name: "Book truck", Type: "one_time"})
	require.NoError(s.T(), err)
	finished := "finished"
	_, _, err = progress.EditStep(ctx, nil, progress.EditStepInput{StepID: created.Step.ID, Status: &finished})
	require.NoError(s.T(), err)

	_, output, err := progress.GetStepList(ctx, nil, progress.GetStepListInput{Status: "finished"})
	require.NoError(s.T(), err)
	require.Len(s.T(), output.Steps, 1)
	assert.Equal(s.T(), "finished", output.Steps[0].Status)
}

// --- Web: browse list, drill-down, and "log a point" form -----------------

func (s *IntegrationTestSuite) TestBrowse_ShowsOpenStepsCompactUnderActiveActivity() {
	ctx := s.Context()
	activityID := s.createActivity(ctx, "Move apartment", domain.ProgressTypeProjectProgress, time.Now().AddDate(0, 0, -5))
	_, _, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: activityID, Name: "Book moving truck", Type: "one_time"})
	require.NoError(s.T(), err)

	r := s.browseRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/progress/browse", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), w.Body.String(), "Book moving truck")
}

func (s *IntegrationTestSuite) TestBrowseDetail_ShowsOpenStepsForActiveActivity() {
	ctx := s.Context()
	activityID := s.createActivity(ctx, "Move apartment", domain.ProgressTypeProjectProgress, time.Now().AddDate(0, 0, -5))
	_, _, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: activityID, Name: "Book moving truck", Type: "one_time"})
	require.NoError(s.T(), err)

	r := s.browseRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/web/progress/browse/%d", activityID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), w.Body.String(), "Book moving truck")
}

func (s *IntegrationTestSuite) TestBrowseDetail_HidesStepsForFinishedActivity() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	activityID := s.createActivity(ctx, "Move apartment", domain.ProgressTypeProjectProgress, time.Now().AddDate(0, 0, -20))
	_, _, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: activityID, Name: "Book moving truck", Type: "one_time"})
	require.NoError(s.T(), err)

	s.finishActivity(ctx, activityID, userID, time.Now())

	r := s.browseRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/web/progress/browse/%d", activityID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.NotContains(s.T(), w.Body.String(), "Book moving truck", "a finished activity's drill-down must not show its steps")
}

func (s *IntegrationTestSuite) TestBrowseNewPoint_ShowsOpenStepCheckboxesAndNewStepFields() {
	ctx := s.Context()
	activityID := s.createActivity(ctx, "Move apartment", domain.ProgressTypeProjectProgress, time.Now().AddDate(0, 0, -5))
	_, created, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: activityID, Name: "Book moving truck", Type: "one_time"})
	require.NoError(s.T(), err)

	r := s.browseRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/web/progress/browse/%d/points/new", activityID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, "Book moving truck")
	assert.Contains(s.T(), body, fmt.Sprintf(`name="close_step_ids" value="%d"`, created.Step.ID))
	assert.Contains(s.T(), body, `name="new_one_time_steps"`)
	assert.Contains(s.T(), body, `name="new_repeatable_steps"`)
}

func (s *IntegrationTestSuite) TestBrowseCreatePoint_ClosesCheckedSteps() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	activityID := s.createActivity(ctx, "Move apartment", domain.ProgressTypeProjectProgress, time.Now().AddDate(0, 0, -5))
	_, created, err := progress.CreateStep(ctx, nil, progress.CreateStepInput{ActivityID: activityID, Name: "Book moving truck", Type: "one_time"})
	require.NoError(s.T(), err)

	r := s.browseRouter(ctx)
	form := url.Values{"value": {"1"}, "close_step_ids": {fmt.Sprint(created.Step.ID)}}
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/web/progress/browse/%d/points", activityID), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusFound, w.Code)

	step, err := s.Repo().GetStep(ctx, created.Step.ID, userID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), step)
	assert.Equal(s.T(), domain.StepStatusFinished, step.Status)
	assert.NotNil(s.T(), step.ClosedAt)
	require.NotNil(s.T(), step.CompletedByProgressPointID)

	points, err := s.Repo().ListProgress(ctx, domain.ProgressFilter{UserID: userID, ActivityID: activityID})
	require.NoError(s.T(), err)
	require.Len(s.T(), points, 1)
	assert.Equal(s.T(), points[0].ID, *step.CompletedByProgressPointID)
}

func (s *IntegrationTestSuite) TestBrowseCreatePoint_QueuesNewStepsFromSemicolonSeparatedFields() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	activityID := s.createActivity(ctx, "Move apartment", domain.ProgressTypeProjectProgress, time.Now().AddDate(0, 0, -5))

	r := s.browseRouter(ctx)
	form := url.Values{
		"value":                {"1"},
		"new_one_time_steps":   {"Book moving truck ; Cancel internet"},
		"new_repeatable_steps": {"Pack a box"},
	}
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/web/progress/browse/%d/points", activityID), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusFound, w.Code)

	steps, err := s.Repo().ListSteps(ctx, domain.StepFilter{UserID: userID, ActivityID: activityID})
	require.NoError(s.T(), err)
	require.Len(s.T(), steps, 3)

	byName := make(map[string]domain.Step, len(steps))
	for _, st := range steps {
		byName[st.Name] = st
	}
	require.Contains(s.T(), byName, "Book moving truck")
	require.Contains(s.T(), byName, "Cancel internet")
	require.Contains(s.T(), byName, "Pack a box")
	assert.Equal(s.T(), domain.StepTypeOneTime, byName["Book moving truck"].Type)
	assert.Equal(s.T(), domain.StepTypeOneTime, byName["Cancel internet"].Type)
	assert.Equal(s.T(), domain.StepTypeRepeatable, byName["Pack a box"].Type)

	points, err := s.Repo().ListProgress(ctx, domain.ProgressFilter{UserID: userID, ActivityID: activityID})
	require.NoError(s.T(), err)
	require.Len(s.T(), points, 1)
	for _, st := range steps {
		require.NotNil(s.T(), st.CreatedByProgressPointID)
		assert.Equal(s.T(), points[0].ID, *st.CreatedByProgressPointID)
	}
}
