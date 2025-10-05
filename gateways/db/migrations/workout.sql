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
