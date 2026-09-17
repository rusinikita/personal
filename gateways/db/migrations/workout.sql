-- =====================================================
-- Система учета тренировок - SQL скрипт создания БД
-- =====================================================

-- =====================================================
-- EXERCISES - Таблица упражнений
-- =====================================================
CREATE TABLE IF NOT EXISTS exercises (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    equipment_type VARCHAR(20) NOT NULL CHECK (equipment_type IN ('machine', 'barbell', 'dumbbells', 'bodyweight')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_exercises_user_id ON exercises(user_id);

-- =====================================================
-- WORKOUTS - Таблица тренировок
-- =====================================================
CREATE TABLE IF NOT EXISTS workouts (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_workouts_user_started ON workouts(user_id, started_at DESC);

-- =====================================================
-- SETS - Таблица подходов
-- =====================================================
CREATE TABLE IF NOT EXISTS sets (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    workout_id BIGINT NOT NULL REFERENCES workouts(id),
    exercise_id BIGINT NOT NULL REFERENCES exercises(id),
    reps BIGINT NULL,
    duration_seconds BIGINT NULL,
    weight_kg DECIMAL(5, 2) NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sets_user_created ON sets(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_sets_exercise_user ON sets(exercise_id, user_id);
