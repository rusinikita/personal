package ideas

import (
	"time"

	"personal/domain"
)

// IdeaResult is one idea as returned by every ideas MCP tool.
type IdeaResult struct {
	ID                      int64  `json:"id" jsonschema:"Idea ID"`
	Body                    string `json:"body" jsonschema:"The user's own words; spike outcomes are appended"`
	Status                  string `json:"status" jsonschema:"inbox, someday, spike or resolved"`
	Resolution              string `json:"resolution,omitempty" jsonschema:"dropped, merged, expired, promoted or blocked; set only when status is resolved"`
	MergedIntoID            *int64 `json:"merged_into_id,omitempty" jsonschema:"Older idea this one was merged into (resolution merged)"`
	ResolvedProgressPointID *int64 `json:"resolved_progress_point_id,omitempty" jsonschema:"Progress point the idea was promoted into (resolution promoted)"`
	ResolvedAt              string `json:"resolved_at,omitempty" jsonschema:"When the idea was resolved (ISO8601)"`
	CreatedAt               string `json:"created_at" jsonschema:"When the idea was captured (ISO8601)"`
	UpdatedAt               string `json:"updated_at" jsonschema:"Last status change or appended text (ISO8601)"`
	SurfaceCount            int    `json:"surface_count" jsonschema:"1 + number of duplicates merged into this idea; 3+ is a spike trigger"`
	LastSurfacedAt          string `json:"last_surfaced_at" jsonschema:"Latest created_at of this idea and its merged duplicates (ISO8601)"`
}

func ideaToResult(idea domain.Idea) IdeaResult {
	r := IdeaResult{
		ID:                      idea.ID,
		Body:                    idea.Body,
		Status:                  string(idea.Status),
		MergedIntoID:            idea.MergedIntoID,
		ResolvedProgressPointID: idea.ResolvedProgressPointID,
		CreatedAt:               idea.CreatedAt.Format(time.RFC3339),
		UpdatedAt:               idea.UpdatedAt.Format(time.RFC3339),
		SurfaceCount:            idea.SurfaceCount,
		LastSurfacedAt:          idea.LastSurfacedAt.Format(time.RFC3339),
	}
	if idea.Resolution != nil {
		r.Resolution = string(*idea.Resolution)
	}
	if idea.ResolvedAt != nil {
		r.ResolvedAt = idea.ResolvedAt.Format(time.RFC3339)
	}
	return r
}

func ideasToResults(ideas []domain.Idea) []IdeaResult {
	results := make([]IdeaResult, len(ideas))
	for i, idea := range ideas {
		results[i] = ideaToResult(idea)
	}
	return results
}
