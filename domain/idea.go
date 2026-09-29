package domain

import "time"

// IdeaStatus is the queue an idea waits in; each open status is reviewed at
// its own ritual (see docs/functions/ideas-spec.md).
type IdeaStatus string

const (
	IdeaStatusInbox    IdeaStatus = "inbox"
	IdeaStatusSomeday  IdeaStatus = "someday"
	IdeaStatusSpike    IdeaStatus = "spike"
	IdeaStatusResolved IdeaStatus = "resolved"
)

// IdeaStatuses are every status, resolved included.
var IdeaStatuses = []IdeaStatus{IdeaStatusInbox, IdeaStatusSomeday, IdeaStatusSpike, IdeaStatusResolved}

// IdeaResolution is the decision made on a resolved idea.
type IdeaResolution string

const (
	IdeaResolutionDropped  IdeaResolution = "dropped"
	IdeaResolutionMerged   IdeaResolution = "merged"
	IdeaResolutionExpired  IdeaResolution = "expired"
	IdeaResolutionPromoted IdeaResolution = "promoted"
	IdeaResolutionBlocked  IdeaResolution = "blocked"
)

// Idea is a raw thought captured outside any existing activity.
type Idea struct {
	ID                      int64           `json:"id" db:"id"`
	UserID                  int64           `json:"-" db:"user_id"`
	Body                    string          `json:"body" db:"body" jsonschema:"The user's own words; spike outcomes are appended"`
	Status                  IdeaStatus      `json:"status" db:"status" jsonschema:"inbox, someday, spike or resolved"`
	Resolution              *IdeaResolution `json:"resolution,omitempty" db:"resolution" jsonschema:"Set only when status is resolved"`
	MergedIntoID            *int64          `json:"merged_into_id,omitempty" db:"merged_into_id" jsonschema:"Older idea this one was merged into (resolution merged)"`
	ResolvedProgressPointID *int64          `json:"resolved_progress_point_id,omitempty" db:"resolved_progress_point_id" jsonschema:"Progress point the idea was promoted into (resolution promoted); its activity and created steps show what it became"`
	ResolvedAt              *time.Time      `json:"resolved_at,omitempty" db:"resolved_at"`
	CreatedAt               time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time       `json:"updated_at" db:"updated_at"`
	// Read-only, computed by ListIdeas/GetIdea/SearchIdeas from ideas merged into this one
	SurfaceCount   int       `json:"surface_count" db:"surface_count" jsonschema:"1 + number of duplicates merged into this idea; 3+ is a spike trigger"`
	LastSurfacedAt time.Time `json:"last_surfaced_at" db:"last_surfaced_at" jsonschema:"Latest created_at of this idea and its merged duplicates"`
}

// IdeaFilter defines query parameters for listing ideas
type IdeaFilter struct {
	UserID       int64
	Statuses     []IdeaStatus     // empty = every open status (all but resolved)
	Resolutions  []IdeaResolution // only resolved ideas with these resolutions, e.g. [blocked] for the portfolio check
	ResolvedFrom *time.Time       // resolved_at >= ResolvedFrom
	ResolvedTo   *time.Time       // resolved_at < ResolvedTo
	Limit        int              // > 0: only the Limit newest ideas, created_at DESC; 0 = all, created_at ASC
}

// IdeaSearchFilter defines parameters for a single-variant body search
type IdeaSearchFilter struct {
	UserID   int64
	Query    string       // required, ILIKE substring on body
	Statuses []IdeaStatus // empty = every status, resolved included
}

// IdeaResolve is the input of ResolveIdea
type IdeaResolve struct {
	IdeaID                  int64
	UserID                  int64
	Resolution              IdeaResolution
	MergedIntoID            *int64
	ResolvedProgressPointID *int64
	ResolvedAt              time.Time
}
