package domain

import "time"

// PersonalDoc is a user-owned markdown document made of ordered chunks (see
// docs/functions/personaldocs-spec.md).
type PersonalDoc struct {
	ID          int64     `json:"id" db:"id"`
	UserID      int64     `json:"-" db:"user_id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description" jsonschema:"One-line summary of what the document is for"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at" jsonschema:"Time of the last write to the document or any of its chunks"`
}

// PersonalDocChunk is one section of a document.
type PersonalDocChunk struct {
	ID           int64     `json:"id" db:"id"`
	UserID       int64     `json:"-" db:"user_id"`
	DocID        int64     `json:"-" db:"doc_id"`
	Position     int       `json:"position" db:"position" jsonschema:"1-based order inside the document"`
	Body         string    `json:"body" db:"body" jsonschema:"Markdown text"`
	PreviousBody *string   `json:"-" db:"previous_body"` // body before the last edit; nil if never edited
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at" jsonschema:"Time of the last body edit"`
}
