package ideas

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var UpdateIdeaMCPDefinition = mcp.Tool{
	Name: "update_idea",
	Annotations: &mcp.ToolAnnotations{
		Title: "Update an idea",
	},
	Description: `Append text to an idea's body and/or change its open status.

Use this tool when:
- Weekly review keeps an inbox idea: status someday
- An idea is chosen for a spike this week: status spike
- A spike gave no answer or wasn't done: status someday
- Spike outcome (4 sections) or new thoughts on the idea: append_body with the user's own words

Allowed status changes: inbox → someday | spike, someday → spike, spike → someday. Setting the current status again is a no-op.
To resolve an idea use resolve_idea. A resolved idea can't be updated.

Inputs:
- idea_id: required
- append_body: (optional) added to body after a blank line; body is never overwritten
- status: (optional) someday or spike`,
}

type UpdateIdeaInput struct {
	IdeaID     int64   `json:"idea_id" jsonschema:"Idea ID"`
	AppendBody *string `json:"append_body,omitempty" jsonschema:"Text appended to body after a blank line"`
	Status     *string `json:"status,omitempty" jsonschema:"New open status: someday or spike"`
}

type UpdateIdeaOutput struct {
	Idea IdeaResult `json:"idea" jsonschema:"Updated idea"`
}

// appendSeparator goes between the existing body and appended text.
const appendSeparator = "\n\n"

// allowedTransitions lists the open-status changes update_idea accepts.
var allowedTransitions = map[domain.IdeaStatus][]domain.IdeaStatus{
	domain.IdeaStatusInbox:   {domain.IdeaStatusSomeday, domain.IdeaStatusSpike},
	domain.IdeaStatusSomeday: {domain.IdeaStatusSpike},
	domain.IdeaStatusSpike:   {domain.IdeaStatusSomeday},
}

func UpdateIdea(ctx context.Context, _ *mcp.CallToolRequest, input UpdateIdeaInput) (*mcp.CallToolResult, UpdateIdeaOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, UpdateIdeaOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, UpdateIdeaOutput{}, fmt.Errorf("user_id not available in context")
	}

	if input.AppendBody == nil && input.Status == nil {
		return nil, UpdateIdeaOutput{}, fmt.Errorf("at least one of append_body or status is required")
	}

	idea, err := db.GetIdea(ctx, input.IdeaID, userID)
	if err != nil {
		return nil, UpdateIdeaOutput{}, fmt.Errorf("database error: %w", err)
	}
	if idea == nil {
		return nil, UpdateIdeaOutput{}, fmt.Errorf("idea not found")
	}
	if idea.Status == domain.IdeaStatusResolved {
		return nil, UpdateIdeaOutput{}, fmt.Errorf("idea is resolved and can't be updated")
	}

	changed := false

	if input.AppendBody != nil {
		if strings.TrimSpace(*input.AppendBody) == "" {
			return nil, UpdateIdeaOutput{}, fmt.Errorf("append_body cannot be empty")
		}
		idea.Body += appendSeparator + *input.AppendBody
		changed = true
	}

	if input.Status != nil {
		status := domain.IdeaStatus(*input.Status)
		if status != idea.Status {
			if !transitionAllowed(idea.Status, status) {
				return nil, UpdateIdeaOutput{}, fmt.Errorf("status can't change from %s to %s", idea.Status, status)
			}
			idea.Status = status
			changed = true
		}
	}

	if changed {
		if err := db.UpdateIdea(ctx, idea); err != nil {
			return nil, UpdateIdeaOutput{}, fmt.Errorf("failed to update idea: %w", err)
		}
		idea, err = db.GetIdea(ctx, input.IdeaID, userID)
		if err != nil {
			return nil, UpdateIdeaOutput{}, fmt.Errorf("failed to fetch updated idea: %w", err)
		}
	}

	return nil, UpdateIdeaOutput{Idea: ideaToResult(*idea)}, nil
}

func transitionAllowed(from, to domain.IdeaStatus) bool {
	for _, allowed := range allowedTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}
