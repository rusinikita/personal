-- Life parts table
CREATE TABLE IF NOT EXISTS life_parts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_life_parts_user_id ON life_parts(user_id);

-- Activities table
CREATE TABLE IF NOT EXISTS activities (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    life_part_ids BIGINT[] DEFAULT '{}',
    name VARCHAR(255) NOT NULL,
    description TEXT,
    progress_type VARCHAR(30) NOT NULL CHECK (progress_type IN ('mood', 'habit_progress', 'project_progress', 'promise_state')),
    frequency_days INT NOT NULL CHECK (frequency_days > 0),
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP, -- NULL unless status is finished or dropped
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_point_at TIMESTAMP, -- NULL means no points
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'finished', 'dropped')),
    deferred_until TIMESTAMP -- NULL unless paused with a resume date
);

CREATE INDEX IF NOT EXISTS idx_activities_user_id ON activities(user_id);
CREATE INDEX IF NOT EXISTS idx_activities_life_part_ids ON activities USING GIN(life_part_ids);

-- Activity progress table
CREATE TABLE IF NOT EXISTS activity_progress (
    id BIGSERIAL PRIMARY KEY,
    activity_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    value INT NOT NULL CHECK (value BETWEEN -2 AND 2),
    hours_left DECIMAL(8,2),
    note TEXT,
    progress_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_progress_activity FOREIGN KEY (activity_id) REFERENCES activities(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_progress_activity_progress_at ON activity_progress(activity_id, progress_at DESC);
CREATE INDEX IF NOT EXISTS idx_progress_user_progress_at ON activity_progress(user_id, progress_at DESC);
