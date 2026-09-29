package tests

// Covers the MCP tools from docs/functions/ideas-spec.md: create_idea,
// list_ideas, search_ideas, update_idea, resolve_idea.

import (
	"context"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/ideas"
	"personal/domain"
	"personal/gateways"
	"personal/util"
)

// createIdea seeds an idea through the repository, optionally with an
// explicit created_at so ordering is deterministic.
func (s *IntegrationTestSuite) createIdea(ctx context.Context, body string, createdAt time.Time) int64 {
	id, err := s.Repo().CreateIdea(ctx, &domain.Idea{
		UserID:    gateways.UserIDFromContext(ctx),
		Body:      body,
		CreatedAt: createdAt,
	})
	require.NoError(s.T(), err)
	return id
}

// setIdeaStatus moves a seeded idea to an open status via the repository.
func (s *IntegrationTestSuite) setIdeaStatus(ctx context.Context, ideaID int64, status domain.IdeaStatus) {
	idea := s.getIdea(ctx, ideaID)
	idea.Status = status
	require.NoError(s.T(), s.Repo().UpdateIdea(ctx, idea))
}

func (s *IntegrationTestSuite) resolveIdea(ctx context.Context, ideaID int64, resolution domain.IdeaResolution, resolvedAt time.Time) {
	require.NoError(s.T(), s.Repo().ResolveIdea(ctx, domain.IdeaResolve{
		IdeaID:     ideaID,
		UserID:     gateways.UserIDFromContext(ctx),
		Resolution: resolution,
		ResolvedAt: resolvedAt,
	}))
}

func (s *IntegrationTestSuite) getIdea(ctx context.Context, ideaID int64) *domain.Idea {
	idea, err := s.Repo().GetIdea(ctx, ideaID, gateways.UserIDFromContext(ctx))
	require.NoError(s.T(), err)
	require.NotNil(s.T(), idea)
	return idea
}

// otherUserContext returns a context for a second user whose data is
// removed after the test.
func (s *IntegrationTestSuite) otherUserContext(ctx context.Context) context.Context {
	otherID := gateways.UserIDFromContext(ctx) + 1
	s.T().Cleanup(func() {
		require.NoError(s.T(), s.dbMaintainer.TruncateUserData(context.Background(), otherID))
	})
	return gateways.WithUserID(ctx, otherID)
}

func (s *IntegrationTestSuite) createPoint(ctx context.Context, activityID int64) int64 {
	id, err := s.Repo().CreateProgress(ctx, &domain.ActivityPoint{
		ActivityID: activityID,
		UserID:     gateways.UserIDFromContext(ctx),
		Value:      1,
		Note:       "check-in",
		ProgressAt: time.Now(),
	})
	require.NoError(s.T(), err)
	return id
}

func ideaIDs(results []ideas.IdeaResult) []int64 {
	ids := make([]int64, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}
	return ids
}

// --- create_idea ----------------------------------------------------------

func (s *IntegrationTestSuite) TestCreateIdea() {
	ctx := s.Context()

	tests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{name: "created in inbox", body: "learn to sail"},
		{name: "empty body fails", body: "   ", wantErr: "body is required"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, out, err := ideas.CreateIdea(ctx, nil, ideas.CreateIdeaInput{Body: tt.body})
			if tt.wantErr != "" {
				require.Error(s.T(), err)
				assert.Contains(s.T(), err.Error(), tt.wantErr)
				return
			}
			require.NoError(s.T(), err)

			idea := s.getIdea(ctx, out.Idea.ID)
			assert.Equal(s.T(), tt.body, idea.Body)
			assert.Equal(s.T(), domain.IdeaStatusInbox, idea.Status)
			assert.Nil(s.T(), idea.Resolution)
			assert.Equal(s.T(), 1, idea.SurfaceCount)
		})
	}
}

// --- list_ideas -----------------------------------------------------------

func (s *IntegrationTestSuite) TestListIdeas() {
	ctx := s.Context()
	now := time.Now()

	inbox := s.createIdea(ctx, "inbox idea", now.Add(-5*time.Hour))
	someday := s.createIdea(ctx, "someday idea", now.Add(-4*time.Hour))
	s.setIdeaStatus(ctx, someday, domain.IdeaStatusSomeday)
	spike := s.createIdea(ctx, "spike idea", now.Add(-3*time.Hour))
	s.setIdeaStatus(ctx, spike, domain.IdeaStatusSpike)
	dropped := s.createIdea(ctx, "dropped idea", now.Add(-2*time.Hour))
	s.resolveIdea(ctx, dropped, domain.IdeaResolutionDropped, now.AddDate(0, 0, -40))
	blocked := s.createIdea(ctx, "blocked idea", now.Add(-time.Hour))
	s.setIdeaStatus(ctx, blocked, domain.IdeaStatusSpike)
	s.resolveIdea(ctx, blocked, domain.IdeaResolutionBlocked, now)

	otherCtx := s.otherUserContext(ctx)
	s.createIdea(otherCtx, "other user's idea", now)

	tests := []struct {
		name  string
		input ideas.ListIdeasInput
		want  []int64
	}{
		{name: "default hides resolved", input: ideas.ListIdeasInput{}, want: []int64{inbox, someday, spike}},
		{name: "statuses filter", input: ideas.ListIdeasInput{Statuses: []string{"someday", "spike"}}, want: []int64{someday, spike}},
		{name: "resolved", input: ideas.ListIdeasInput{Statuses: []string{"resolved"}}, want: []int64{dropped, blocked}},
		{name: "blocked only", input: ideas.ListIdeasInput{Resolutions: []string{"blocked"}}, want: []int64{blocked}},
		{
			name: "resolved window",
			input: ideas.ListIdeasInput{
				Statuses:     []string{"resolved"},
				ResolvedFrom: now.AddDate(0, 0, -50).Format(time.RFC3339),
				ResolvedTo:   now.AddDate(0, 0, -30).Format(time.RFC3339),
			},
			want: []int64{dropped},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, out, err := ideas.ListIdeas(ctx, nil, tt.input)
			require.NoError(s.T(), err)
			assert.Equal(s.T(), tt.want, ideaIDs(out.Ideas))
		})
	}
}

// --- search_ideas ---------------------------------------------------------

func (s *IntegrationTestSuite) TestSearchIdeas() {
	ctx := s.Context()
	now := time.Now()

	oldSail := s.createIdea(ctx, "Learn to SAIL on a yacht", now.Add(-3*time.Hour))
	newSail := s.createIdea(ctx, "sailing trip in Greece", now.Add(-2*time.Hour))
	boat := s.createIdea(ctx, "buy a small boat", now.Add(-time.Hour))
	s.createIdea(ctx, "unrelated: bake bread", now)

	droppedYacht := s.createIdea(ctx, "rent a yacht for summer", now.Add(-4*time.Hour))
	s.resolveIdea(ctx, droppedYacht, domain.IdeaResolutionDropped, now)

	_, merged, err := ideas.ResolveIdea(ctx, nil, ideas.ResolveIdeaInput{
		IdeaID: newSail, Resolution: "merged", MergedIntoID: util.Ptr(oldSail),
	})
	require.NoError(s.T(), err)
	require.Equal(s.T(), oldSail, *merged.Idea.MergedIntoID)

	otherCtx := s.otherUserContext(ctx)
	s.createIdea(otherCtx, "sail around the world", now)

	tests := []struct {
		name      string
		input     ideas.SearchIdeasInput
		want      []int64
		wantMatch []int
		wantErr   string
	}{
		{
			name:      "case-insensitive substring",
			input:     ideas.SearchIdeasInput{QueryVariants: []string{"sail"}},
			want:      []int64{newSail, oldSail},
			wantMatch: []int{1, 1},
		},
		{
			name:      "ranked by match_count, then newest first; each idea once",
			input:     ideas.SearchIdeasInput{QueryVariants: []string{"sail", "yacht", "boat"}},
			want:      []int64{oldSail, boat, newSail, droppedYacht},
			wantMatch: []int{2, 1, 1, 1},
		},
		{
			name:      "default includes resolved",
			input:     ideas.SearchIdeasInput{QueryVariants: []string{"rent a yacht"}},
			want:      []int64{droppedYacht},
			wantMatch: []int{1},
		},
		{
			name:      "statuses filter",
			input:     ideas.SearchIdeasInput{QueryVariants: []string{"yacht"}, Statuses: []string{"inbox"}},
			want:      []int64{oldSail},
			wantMatch: []int{1},
		},
		{name: "no variants fails", input: ideas.SearchIdeasInput{}, wantErr: "cannot be empty"},
		{name: "more than 5 variants fails", input: ideas.SearchIdeasInput{QueryVariants: []string{"a", "b", "c", "d", "e", "f"}}, wantErr: "maximum 5"},
		{name: "empty variant fails", input: ideas.SearchIdeasInput{QueryVariants: []string{"sail", ""}}, wantErr: "empty strings"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, out, err := ideas.SearchIdeas(ctx, nil, tt.input)
			require.NoError(s.T(), err)
			if tt.wantErr != "" {
				assert.Contains(s.T(), out.Error, tt.wantErr)
				return
			}
			assert.Empty(s.T(), out.Error)

			gotIDs := make([]int64, len(out.Results))
			gotMatch := make([]int, len(out.Results))
			for i, r := range out.Results {
				gotIDs[i] = r.ID
				gotMatch[i] = r.MatchCount
			}
			assert.Equal(s.T(), tt.want, gotIDs)
			assert.Equal(s.T(), tt.wantMatch, gotMatch)
		})
	}

	s.Run("merged idea returned with merged_into_id", func() {
		_, out, err := ideas.SearchIdeas(ctx, nil, ideas.SearchIdeasInput{QueryVariants: []string{"Greece"}})
		require.NoError(s.T(), err)
		require.Len(s.T(), out.Results, 1)
		assert.Equal(s.T(), "merged", out.Results[0].Resolution)
		require.NotNil(s.T(), out.Results[0].MergedIntoID)
		assert.Equal(s.T(), oldSail, *out.Results[0].MergedIntoID)
	})
}

// --- update_idea ----------------------------------------------------------

func (s *IntegrationTestSuite) TestUpdateIdea() {
	ctx := s.Context()

	s.Run("append adds a blank line and the text", func() {
		id := s.createIdea(ctx, "learn to sail", time.Time{})
		_, out, err := ideas.UpdateIdea(ctx, nil, ideas.UpdateIdeaInput{IdeaID: id, AppendBody: util.Ptr("spike: 2 schools nearby")})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), "learn to sail\n\nspike: 2 schools nearby", out.Idea.Body)
		assert.Equal(s.T(), "learn to sail\n\nspike: 2 schools nearby", s.getIdea(ctx, id).Body)
	})

	transitions := []struct {
		from domain.IdeaStatus
		to   string
	}{
		{domain.IdeaStatusInbox, "someday"},
		{domain.IdeaStatusInbox, "spike"},
		{domain.IdeaStatusSomeday, "spike"},
		{domain.IdeaStatusSpike, "someday"},
	}
	for _, tt := range transitions {
		s.Run("transition "+string(tt.from)+" → "+tt.to, func() {
			id := s.createIdea(ctx, "idea", time.Time{})
			if tt.from != domain.IdeaStatusInbox {
				s.setIdeaStatus(ctx, id, tt.from)
			}
			_, _, err := ideas.UpdateIdea(ctx, nil, ideas.UpdateIdeaInput{IdeaID: id, Status: util.Ptr(tt.to)})
			require.NoError(s.T(), err)
			assert.Equal(s.T(), domain.IdeaStatus(tt.to), s.getIdea(ctx, id).Status)
		})
	}

	s.Run("setting the current status is a no-op", func() {
		id := s.createIdea(ctx, "idea", time.Time{})
		s.setIdeaStatus(ctx, id, domain.IdeaStatusSomeday)
		before := s.getIdea(ctx, id)

		_, _, err := ideas.UpdateIdea(ctx, nil, ideas.UpdateIdeaInput{IdeaID: id, Status: util.Ptr("someday")})
		require.NoError(s.T(), err)
		after := s.getIdea(ctx, id)
		assert.Equal(s.T(), domain.IdeaStatusSomeday, after.Status)
		assert.True(s.T(), before.UpdatedAt.Equal(after.UpdatedAt))
	})

	invalid := []struct {
		from domain.IdeaStatus
		to   string
	}{
		{domain.IdeaStatusSomeday, "inbox"},
		{domain.IdeaStatusSpike, "inbox"},
		{domain.IdeaStatusInbox, "resolved"},
	}
	for _, tt := range invalid {
		s.Run("transition "+string(tt.from)+" → "+tt.to+" fails", func() {
			id := s.createIdea(ctx, "idea", time.Time{})
			s.setIdeaStatus(ctx, id, tt.from)
			_, _, err := ideas.UpdateIdea(ctx, nil, ideas.UpdateIdeaInput{IdeaID: id, Status: util.Ptr(tt.to)})
			require.Error(s.T(), err)
			assert.Equal(s.T(), tt.from, s.getIdea(ctx, id).Status)
		})
	}

	s.Run("any change on a resolved idea fails", func() {
		id := s.createIdea(ctx, "idea", time.Time{})
		s.resolveIdea(ctx, id, domain.IdeaResolutionDropped, time.Now())

		_, _, err := ideas.UpdateIdea(ctx, nil, ideas.UpdateIdeaInput{IdeaID: id, AppendBody: util.Ptr("more")})
		require.Error(s.T(), err)
		_, _, err = ideas.UpdateIdea(ctx, nil, ideas.UpdateIdeaInput{IdeaID: id, Status: util.Ptr("someday")})
		require.Error(s.T(), err)
		assert.Equal(s.T(), "idea", s.getIdea(ctx, id).Body)
	})
}

// --- resolve_idea ---------------------------------------------------------

func (s *IntegrationTestSuite) TestResolveIdea() {
	ctx := s.Context()
	now := time.Now()

	for _, resolution := range []string{"dropped", "expired"} {
		s.Run(resolution+" resolves the idea", func() {
			id := s.createIdea(ctx, "idea", time.Time{})
			_, _, err := ideas.ResolveIdea(ctx, nil, ideas.ResolveIdeaInput{IdeaID: id, Resolution: resolution})
			require.NoError(s.T(), err)

			idea := s.getIdea(ctx, id)
			assert.Equal(s.T(), domain.IdeaStatusResolved, idea.Status)
			require.NotNil(s.T(), idea.Resolution)
			assert.Equal(s.T(), domain.IdeaResolution(resolution), *idea.Resolution)
			assert.NotNil(s.T(), idea.ResolvedAt)
		})
	}

	s.Run("merged into an older idea counts as a surfacing", func() {
		older := s.createIdea(ctx, "older", now.Add(-2*time.Hour))
		duplicate := s.createIdea(ctx, "duplicate", now.Add(-time.Hour))

		_, _, err := ideas.ResolveIdea(ctx, nil, ideas.ResolveIdeaInput{IdeaID: duplicate, Resolution: "merged", MergedIntoID: util.Ptr(older)})
		require.NoError(s.T(), err)

		o := s.getIdea(ctx, older)
		assert.Equal(s.T(), 2, o.SurfaceCount)
		assert.True(s.T(), s.getIdea(ctx, duplicate).CreatedAt.Equal(o.LastSurfacedAt))
	})

	s.Run("merging B with its own duplicate into A re-points the duplicate", func() {
		a := s.createIdea(ctx, "A", now.Add(-3*time.Hour))
		b := s.createIdea(ctx, "B", now.Add(-2*time.Hour))
		c := s.createIdea(ctx, "C", now.Add(-time.Hour))

		_, _, err := ideas.ResolveIdea(ctx, nil, ideas.ResolveIdeaInput{IdeaID: c, Resolution: "merged", MergedIntoID: util.Ptr(b)})
		require.NoError(s.T(), err)
		_, _, err = ideas.ResolveIdea(ctx, nil, ideas.ResolveIdeaInput{IdeaID: b, Resolution: "merged", MergedIntoID: util.Ptr(a)})
		require.NoError(s.T(), err)

		assert.Equal(s.T(), 3, s.getIdea(ctx, a).SurfaceCount)
		assert.Equal(s.T(), a, *s.getIdea(ctx, c).MergedIntoID)
		assert.Equal(s.T(), 1, s.getIdea(ctx, b).SurfaceCount)
	})

	s.Run("invalid merges fail", func() {
		id := s.createIdea(ctx, "idea", time.Time{})
		resolved := s.createIdea(ctx, "resolved", time.Time{})
		s.resolveIdea(ctx, resolved, domain.IdeaResolutionDropped, now)
		foreign := s.createIdea(s.otherUserContext(ctx), "foreign", time.Time{})

		for name, target := range map[string]*int64{
			"into itself":            util.Ptr(id),
			"into a resolved idea":   util.Ptr(resolved),
			"into another user's":    util.Ptr(foreign),
			"without merged_into_id": nil,
		} {
			_, _, err := ideas.ResolveIdea(ctx, nil, ideas.ResolveIdeaInput{IdeaID: id, Resolution: "merged", MergedIntoID: target})
			assert.Error(s.T(), err, name)
		}
		assert.Equal(s.T(), domain.IdeaStatusInbox, s.getIdea(ctx, id).Status)
	})

	s.Run("promoted with own point, invalid points fail", func() {
		activityID := s.createActivity(ctx, "Sailing", domain.ProgressTypeProjectProgress, now.AddDate(0, 0, -1))
		pointID := s.createPoint(ctx, activityID)

		otherCtx := s.otherUserContext(ctx)
		otherActivity := s.createActivity(otherCtx, "Other", domain.ProgressTypeProjectProgress, now.AddDate(0, 0, -1))
		foreignPoint := s.createPoint(otherCtx, otherActivity)

		id := s.createIdea(ctx, "learn to sail", time.Time{})
		for name, point := range map[string]*int64{
			"missing point id":   nil,
			"other user's point": util.Ptr(foreignPoint),
			"non-existent point": util.Ptr(int64(999999999)),
		} {
			_, _, err := ideas.ResolveIdea(ctx, nil, ideas.ResolveIdeaInput{IdeaID: id, Resolution: "promoted", ResolvedProgressPointID: point})
			assert.Error(s.T(), err, name)
		}

		_, _, err := ideas.ResolveIdea(ctx, nil, ideas.ResolveIdeaInput{IdeaID: id, Resolution: "promoted", ResolvedProgressPointID: util.Ptr(pointID)})
		require.NoError(s.T(), err)
		idea := s.getIdea(ctx, id)
		assert.Equal(s.T(), domain.IdeaResolutionPromoted, *idea.Resolution)
		assert.Equal(s.T(), pointID, *idea.ResolvedProgressPointID)

		require.NoError(s.T(), s.Repo().DeleteProgress(ctx, pointID, gateways.UserIDFromContext(ctx)))
		idea = s.getIdea(ctx, id)
		assert.Equal(s.T(), domain.IdeaResolutionPromoted, *idea.Resolution)
		assert.Nil(s.T(), idea.ResolvedProgressPointID)
	})

	s.Run("blocked only from spike", func() {
		spike := s.createIdea(ctx, "spike", time.Time{})
		s.setIdeaStatus(ctx, spike, domain.IdeaStatusSpike)
		_, _, err := ideas.ResolveIdea(ctx, nil, ideas.ResolveIdeaInput{IdeaID: spike, Resolution: "blocked"})
		require.NoError(s.T(), err)

		for _, status := range []domain.IdeaStatus{domain.IdeaStatusInbox, domain.IdeaStatusSomeday} {
			id := s.createIdea(ctx, "idea", time.Time{})
			s.setIdeaStatus(ctx, id, status)
			_, _, err := ideas.ResolveIdea(ctx, nil, ideas.ResolveIdeaInput{IdeaID: id, Resolution: "blocked"})
			assert.Error(s.T(), err, string(status))
		}
	})

	s.Run("blocked idea can be re-resolved", func() {
		activityID := s.createActivity(ctx, "Freed slot", domain.ProgressTypeProjectProgress, now.AddDate(0, 0, -1))

		for _, resolution := range []string{"promoted", "dropped", "expired"} {
			id := s.createIdea(ctx, "idea", time.Time{})
			s.setIdeaStatus(ctx, id, domain.IdeaStatusSpike)
			s.resolveIdea(ctx, id, domain.IdeaResolutionBlocked, now.AddDate(0, -1, 0))

			input := ideas.ResolveIdeaInput{IdeaID: id, Resolution: resolution}
			if resolution == "promoted" {
				input.ResolvedProgressPointID = util.Ptr(s.createPoint(ctx, activityID))
			}
			_, _, err := ideas.ResolveIdea(ctx, nil, input)
			require.NoError(s.T(), err, resolution)

			idea := s.getIdea(ctx, id)
			assert.Equal(s.T(), domain.IdeaResolution(resolution), *idea.Resolution)
			assert.True(s.T(), idea.ResolvedAt.After(now.AddDate(0, 0, -1)), "resolved_at is overwritten")
		}
	})

	s.Run("re-resolving any other resolved idea fails", func() {
		id := s.createIdea(ctx, "idea", time.Time{})
		s.resolveIdea(ctx, id, domain.IdeaResolutionDropped, now)
		_, _, err := ideas.ResolveIdea(ctx, nil, ideas.ResolveIdeaInput{IdeaID: id, Resolution: "expired"})
		require.Error(s.T(), err)
		assert.Equal(s.T(), domain.IdeaResolutionDropped, *s.getIdea(ctx, id).Resolution)
	})
}
