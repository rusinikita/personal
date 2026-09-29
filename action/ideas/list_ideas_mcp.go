package ideas

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var ListIdeasMCPDefinition = mcp.Tool{
	Name: "list_ideas",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint:   true,
		IdempotentHint: true,
		Title:          "List ideas",
	},
	Description: `List the user's ideas with surface_count and last_surfaced_at, oldest first.

Use this tool when:
- Weekly review: statuses [inbox] (step 1), [spike] (step 2), [inbox, someday] (spike choice, step 9)
- Monthly review: statuses [someday]; resolutions [blocked] for the portfolio check
- Quarterly review: statuses [resolved] with resolved_from / resolved_to

Parameters (all optional):
- statuses: inbox, someday, spike, resolved; default every open status (resolved hidden)
- resolutions: dropped, merged, expired, promoted, blocked; only resolved ideas with these resolutions
- resolved_from / resolved_to: ISO8601 window on resolved_at (from inclusive, to exclusive)`,
}

type ListIdeasInput struct {
	Statuses     []string `json:"statuses,omitempty" jsonschema:"inbox, someday, spike, resolved; default every open status"`
	Resolutions  []string `json:"resolutions,omitempty" jsonschema:"dropped, merged, expired, promoted, blocked; only resolved ideas with these resolutions"`
	ResolvedFrom string   `json:"resolved_from,omitempty" jsonschema:"ISO8601, resolved_at >= resolved_from"`
	ResolvedTo   string   `json:"resolved_to,omitempty" jsonschema:"ISO8601, resolved_at < resolved_to"`
}

type ListIdeasOutput struct {
	Ideas []IdeaResult `json:"ideas" jsonschema:"Matching ideas, oldest first"`
}

var validStatuses = map[domain.IdeaStatus]bool{
	domain.IdeaStatusInbox:    true,
	domain.IdeaStatusSomeday:  true,
	domain.IdeaStatusSpike:    true,
	domain.IdeaStatusResolved: true,
}

var validResolutions = map[domain.IdeaResolution]bool{
	domain.IdeaResolutionDropped:  true,
	domain.IdeaResolutionMerged:   true,
	domain.IdeaResolutionExpired:  true,
	domain.IdeaResolutionPromoted: true,
	domain.IdeaResolutionBlocked:  true,
}

func ListIdeas(ctx context.Context, _ *mcp.CallToolRequest, input ListIdeasInput) (*mcp.CallToolResult, ListIdeasOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, ListIdeasOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, ListIdeasOutput{}, fmt.Errorf("user_id not available in context")
	}

	statuses, err := parseStatuses(input.Statuses)
	if err != nil {
		return nil, ListIdeasOutput{}, err
	}

	filter := domain.IdeaFilter{UserID: userID, Statuses: statuses}

	for _, raw := range input.Resolutions {
		resolution := domain.IdeaResolution(raw)
		if !validResolutions[resolution] {
			return nil, ListIdeasOutput{}, fmt.Errorf("invalid resolution %q", raw)
		}
		filter.Resolutions = append(filter.Resolutions, resolution)
	}

	if input.ResolvedFrom != "" {
		from, err := time.Parse(time.RFC3339, input.ResolvedFrom)
		if err != nil {
			return nil, ListIdeasOutput{}, fmt.Errorf("invalid resolved_from format, expected RFC3339: %w", err)
		}
		filter.ResolvedFrom = &from
	}
	if input.ResolvedTo != "" {
		to, err := time.Parse(time.RFC3339, input.ResolvedTo)
		if err != nil {
			return nil, ListIdeasOutput{}, fmt.Errorf("invalid resolved_to format, expected RFC3339: %w", err)
		}
		filter.ResolvedTo = &to
	}

	ideas, err := db.ListIdeas(ctx, filter)
	if err != nil {
		return nil, ListIdeasOutput{}, fmt.Errorf("failed to list ideas: %w", err)
	}

	return nil, ListIdeasOutput{Ideas: ideasToResults(ideas)}, nil
}

func parseStatuses(raw []string) ([]domain.IdeaStatus, error) {
	var statuses []domain.IdeaStatus
	for _, s := range raw {
		status := domain.IdeaStatus(s)
		if !validStatuses[status] {
			return nil, fmt.Errorf("invalid status %q", s)
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}
