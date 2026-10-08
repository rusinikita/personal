CREATE TABLE IF NOT EXISTS docs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    title TEXT NOT NULL CHECK (title <> ''),
    description TEXT NOT NULL CHECK (description <> ''),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- stored: set by UpdatePersonalDoc and by every chunk write
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_docs_user_id ON docs(user_id);

CREATE TABLE IF NOT EXISTS doc_chunks (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    doc_id BIGINT NOT NULL,
    position INT NOT NULL CHECK (position >= 1),
    body TEXT NOT NULL CHECK (body <> ''),
    previous_body TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_doc_chunk_doc FOREIGN KEY (doc_id) REFERENCES docs(id),
    -- backstop for "no two chunks share a position"; checked at commit, so the shift
    -- and the insert or move inside one transaction may pass through duplicates.
    -- Its index also serves reading a document's chunks in order
    CONSTRAINT uq_doc_chunks_doc_position UNIQUE (doc_id, position) DEFERRABLE INITIALLY DEFERRED
);
