package ideas

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var ResolveIdeaMCPDefinition = mcp.Tool{
	Name: "resolve_idea",
	Annotations: &mcp.ToolAnnotations{
		Title: "Resolve an idea",
	},
	Description: `Resolve an open idea with a decision, or re-resolve a blocked one.

Resolutions:
- dropped: decided not to do it (including "not needed" after a spike)
- merged: a duplicate; requires merged_into_id, the older idea (own, not resolved, not itself). The older idea gets +1 to surface_count and also takes over this idea's own duplicates
- expired: a someday or blocked idea with no movement for more than 3 months
- promoted: became progress; requires resolved_progress_point_id, the point it became (a check-in on an existing activity, or the initial point of a new activity after a go). Create the point (and steps with created_by_progress_point_id) first
- blocked: only from spike; needed, but weaker than every current activity while the WIP limit is full

A blocked idea is the only resolved one that can be resolved again, as promoted, dropped or expired (resolved_at is overwritten).`,
}

type ResolveIdeaInput struct {
	IdeaID                  int64  `json:"idea_id" jsonschema:"Idea ID"`
	Resolution              string `json:"resolution" jsonschema:"dropped, merged, expired, promoted or blocked"`
	MergedIntoID            *int64 `json:"merged_into_id,omitempty" jsonschema:"Older idea to merge into (only for merged)"`
	ResolvedProgressPointID *int64 `json:"resolved_progress_point_id,omitempty" jsonschema:"Progress point the idea became (only for promoted)"`
}

type ResolveIdeaOutput struct {
	Idea IdeaResult `json:"idea" jsonschema:"Resolved idea"`
}

// reResolvableFromBlocked are the resolutions a blocked idea can move to.
var reResolvableFromBlocked = map[domain.IdeaResolution]bool{
	domain.IdeaResolutionPromoted: true,
	domain.IdeaResolutionDropped:  true,
	domain.IdeaResolutionExpired:  true,
}

func ResolveIdea(ctx context.Context, _ *mcp.CallToolRequest, input ResolveIdeaInput) (*mcp.CallToolResult, ResolveIdeaOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, ResolveIdeaOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, ResolveIdeaOutput{}, fmt.Errorf("user_id not available in context")
	}

	resolution := domain.IdeaResolution(input.Resolution)
	if !validResolutions[resolution] {
		return nil, ResolveIdeaOutput{}, fmt.Errorf("invalid resolution %q", input.Resolution)
	}

	idea, err := db.GetIdea(ctx, input.IdeaID, userID)
	if err != nil {
		return nil, ResolveIdeaOutput{}, fmt.Errorf("database error: %w", err)
	}
	if idea == nil {
		return nil, ResolveIdeaOutput{}, fmt.Errorf("idea not found")
	}

	if idea.Status == domain.IdeaStatusResolved {
		if *idea.Resolution != domain.IdeaResolutionBlocked {
			return nil, ResolveIdeaOutput{}, fmt.Errorf("idea is already resolved as %s", *idea.Resolution)
		}
		if !reResolvableFromBlocked[resolution] {
			return nil, ResolveIdeaOutput{}, fmt.Errorf("a blocked idea can only be re-resolved as promoted, dropped or expired")
		}
	} else if resolution == domain.IdeaResolutionBlocked && idea.Status != domain.IdeaStatusSpike {
		return nil, ResolveIdeaOutput{}, fmt.Errorf("blocked is only allowed from spike")
	}

	if resolution != domain.IdeaResolutionMerged && input.MergedIntoID != nil {
		return nil, ResolveIdeaOutput{}, fmt.Errorf("merged_into_id is only allowed for merged")
	}
	if resolution != domain.IdeaResolutionPromoted && input.ResolvedProgressPointID != nil {
		return nil, ResolveIdeaOutput{}, fmt.Errorf("resolved_progress_point_id is only allowed for promoted")
	}

	switch resolution {
	case domain.IdeaResolutionMerged:
		if err := checkMergeTarget(ctx, db, userID, idea.ID, input.MergedIntoID); err != nil {
			return nil, ResolveIdeaOutput{}, err
		}
	case domain.IdeaResolutionPromoted:
		if input.ResolvedProgressPointID == nil {
			return nil, ResolveIdeaOutput{}, fmt.Errorf("resolved_progress_point_id is required for promoted")
		}
		point, err := db.GetProgress(ctx, *input.ResolvedProgressPointID, userID)
		if err != nil {
			return nil, ResolveIdeaOutput{}, fmt.Errorf("database error: %w", err)
		}
		if point == nil {
			return nil, ResolveIdeaOutput{}, fmt.Errorf("progress point not found")
		}
	}

	err = db.ResolveIdea(ctx, domain.IdeaResolve{
		IdeaID:                  idea.ID,
		UserID:                  userID,
		Resolution:              resolution,
		MergedIntoID:            input.MergedIntoID,
		ResolvedProgressPointID: input.ResolvedProgressPointID,
		ResolvedAt:              time.Now(),
	})
	if err != nil {
		return nil, ResolveIdeaOutput{}, fmt.Errorf("failed to resolve idea: %w", err)
	}

	resolved, err := db.GetIdea(ctx, idea.ID, userID)
	if err != nil {
		return nil, ResolveIdeaOutput{}, fmt.Errorf("failed to fetch resolved idea: %w", err)
	}

	return nil, ResolveIdeaOutput{Idea: ideaToResult(*resolved)}, nil
}

func checkMergeTarget(ctx context.Context, db gateways.DB, userID, ideaID int64, targetID *int64) error {
	if targetID == nil {
		return fmt.Errorf("merged_into_id is required for merged")
	}
	if *targetID == ideaID {
		return fmt.Errorf("an idea can't be merged into itself")
	}

	target, err := db.GetIdea(ctx, *targetID, userID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if target == nil {
		return fmt.Errorf("merge target idea not found")
	}
	if target.Status == domain.IdeaStatusResolved {
		return fmt.Errorf("merge target idea is resolved")
	}

	return nil
}
