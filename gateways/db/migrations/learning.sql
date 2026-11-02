CREATE TABLE IF NOT EXISTS learning_plans (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'completed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS skills (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    learning_plan_id BIGINT NOT NULL REFERENCES learning_plans(id),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS learning_exercises (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    skill_id BIGINT NOT NULL REFERENCES skills(id),
    kind VARCHAR(100),
    content TEXT NOT NULL,
    planned_for TIMESTAMPTZ NOT NULL,
    answer TEXT,
    answered_at TIMESTAMPTZ,
    score SMALLINT,
    note TEXT,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS vocabulary (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    learning_plan_id BIGINT NOT NULL REFERENCES learning_plans(id),
    skill_id BIGINT REFERENCES skills(id),
    term VARCHAR(255) NOT NULL,
    translation VARCHAR(255) NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS vocabulary_reviews (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    vocabulary_id BIGINT NOT NULL REFERENCES vocabulary(id) ON DELETE CASCADE,
    planned_for TIMESTAMPTZ NOT NULL,
    score SMALLINT,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
