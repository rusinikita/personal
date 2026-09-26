-- Named z_achievements.sql (not achievements.sql) so ApplyMigrations, which
-- runs every *.sql file in alphabetical order on every startup, applies this
-- file after workout.sql and progress.sql — achievements.exercise_id/activity_id
-- are real FK references to exercises(id)/activities(id), which must already exist.

-- Rename preamble for databases created before the goals → achievements rename
-- (see achievements-spec.md). Every step is a no-op once done or on a fresh DB.
ALTER TABLE IF EXISTS goals RENAME TO achievements;

DO $$
DECLARE
    tbl regclass := to_regclass('achievements');
    old_name text;
    new_name text;
BEGIN
    IF tbl IS NULL THEN
        RETURN;
    END IF;

    IF EXISTS (SELECT 1 FROM pg_attribute
               WHERE attrelid = tbl AND attname = 'goal_type' AND NOT attisdropped) THEN
        ALTER TABLE achievements RENAME COLUMN goal_type TO achievement_type;
    END IF;

    FOREACH old_name IN ARRAY ARRAY[
        'check_goal_type', 'check_goal_period',
        'goals_pkey', 'goals_exercise_id_fkey', 'goals_activity_id_fkey'
    ] LOOP
        new_name := replace(replace(old_name, 'goals_', 'achievements_'), 'goal_', 'achievement_');
        IF EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = tbl AND conname = old_name) THEN
            EXECUTE format('ALTER TABLE achievements RENAME CONSTRAINT %I TO %I', old_name, new_name);
        END IF;
    END LOOP;
END $$;

ALTER INDEX IF EXISTS idx_goals_user_period RENAME TO idx_achievements_user_period;
ALTER INDEX IF EXISTS idx_goals_user_type RENAME TO idx_achievements_user_type;
ALTER SEQUENCE IF EXISTS goals_id_seq RENAME TO achievements_id_seq;

CREATE TABLE IF NOT EXISTS achievements (
    id                    BIGSERIAL PRIMARY KEY,
    user_id               BIGINT NOT NULL,
    name                  VARCHAR(255) NOT NULL,
    achievement_type      VARCHAR(25) NOT NULL,            -- 'money_saving', 'money_spend', 'exercise_max_weight', 'exercise_total_volume', 'activity_occurrence_count', 'activity_streak_count', 'manual'
    exercise_id           BIGINT REFERENCES exercises(id), -- exercise_max_weight|exercise_total_volume only
    activity_id           BIGINT REFERENCES activities(id), -- activity_occurrence_count|activity_streak_count only
    target_value          DECIMAL(12,2) NOT NULL,
    current_value         DECIMAL(12,2) NOT NULL DEFAULT 0, -- cached for every achievement_type — see achievements-spec.md Best Practices for who writes it and when
    details               JSONB NOT NULL DEFAULT '{}',     -- remaining type-specific scalars, shape varies by achievement_type — see achievements-spec.md Best Practices:
                                                            --   money_spend: {"category": "food/cafe"}
                                                            --   money_saving: {"baseline_balance_eur": 1234.56}
                                                            --   exercise_max_weight / exercise_total_volume: {"unit"?: "kg"}
                                                            --   activity_occurrence_count / activity_streak_count: {"unit"?: "sessions"}
                                                            --   manual: {"unit"?: "books"}
    starts_at             TIMESTAMPTZ NOT NULL,
    ends_at               TIMESTAMPTZ,                     -- nullable — null means no deadline
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- bumped by create_achievement (insert) and every UpdateAchievement call (update_achievement, log_achievement_progress, refresh_achievements)

    CONSTRAINT check_achievement_type CHECK (achievement_type IN (
        'money_saving', 'money_spend', 'exercise_max_weight', 'exercise_total_volume',
        'activity_occurrence_count', 'activity_streak_count', 'manual'
    )),
    CONSTRAINT check_spend_category CHECK (
        (achievement_type = 'money_spend' AND details ? 'category' AND details->>'category' <> '')
        OR (achievement_type <> 'money_spend' AND NOT (details ? 'category'))
    ),
    CONSTRAINT check_saving_baseline CHECK (
        (achievement_type = 'money_saving' AND details ? 'baseline_balance_eur')
        OR (achievement_type <> 'money_saving' AND NOT (details ? 'baseline_balance_eur'))
    ),
    CONSTRAINT check_exercise_link CHECK (
        (achievement_type IN ('exercise_max_weight', 'exercise_total_volume') AND exercise_id IS NOT NULL)
        OR (achievement_type NOT IN ('exercise_max_weight', 'exercise_total_volume') AND exercise_id IS NULL)
    ),
    CONSTRAINT check_activity_link CHECK (
        (achievement_type IN ('activity_occurrence_count', 'activity_streak_count') AND activity_id IS NOT NULL)
        OR (achievement_type NOT IN ('activity_occurrence_count', 'activity_streak_count') AND activity_id IS NULL)
    ),
    CONSTRAINT check_unit_scope CHECK (
        NOT (details ? 'unit') OR achievement_type IN (
            'exercise_max_weight', 'exercise_total_volume', 'activity_occurrence_count', 'activity_streak_count', 'manual'
        )
    ),
    CONSTRAINT check_target_value CHECK (target_value > 0),
    CONSTRAINT check_achievement_period CHECK (ends_at IS NULL OR ends_at > starts_at)
);

CREATE INDEX IF NOT EXISTS idx_achievements_user_period ON achievements(user_id, starts_at, ends_at);
CREATE INDEX IF NOT EXISTS idx_achievements_user_type   ON achievements(user_id, achievement_type);
