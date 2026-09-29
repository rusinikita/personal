-- Named z_ideas.sql (not ideas.sql) so ApplyMigrations, which runs every
-- *.sql file in alphabetical order, applies it after progress.sql —
-- ideas.resolved_progress_point_id is a real FK to activity_progress(id).
CREATE TABLE IF NOT EXISTS ideas (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    body TEXT NOT NULL CHECK (body <> ''),
    status VARCHAR(20) NOT NULL DEFAULT 'inbox' CHECK (status IN ('inbox', 'someday', 'spike', 'resolved')),
    resolution VARCHAR(20) CHECK (resolution IN ('dropped', 'merged', 'expired', 'promoted', 'blocked')),
    merged_into_id BIGINT,
    resolved_progress_point_id BIGINT,
    resolved_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_idea_merged_into FOREIGN KEY (merged_into_id) REFERENCES ideas(id),
    CONSTRAINT fk_idea_resolved_point FOREIGN KEY (resolved_progress_point_id) REFERENCES activity_progress(id) ON DELETE SET NULL,
    -- resolution and resolved_at are set iff status = resolved
    CONSTRAINT check_idea_resolution CHECK ((status = 'resolved') = (resolution IS NOT NULL)),
    CONSTRAINT check_idea_resolved_at CHECK ((status = 'resolved') = (resolved_at IS NOT NULL)),
    -- merged_into_id iff merged; point link only for promoted (NULL after the point is deleted)
    CONSTRAINT check_idea_merged_into CHECK (COALESCE(resolution = 'merged', FALSE) = (merged_into_id IS NOT NULL)),
    CONSTRAINT check_idea_resolved_point CHECK (resolved_progress_point_id IS NULL OR resolution = 'promoted')
);

CREATE INDEX IF NOT EXISTS idx_ideas_user_status ON ideas(user_id, status);
CREATE INDEX IF NOT EXISTS idx_ideas_merged_into_id ON ideas(merged_into_id) WHERE merged_into_id IS NOT NULL;
