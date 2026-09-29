-- Fixture data for cmd/webui-preview, applied once on top of fresh
-- migrations and then snapshotted. Everything belongs to user_id = 1
-- (webui.DefaultUserID); timestamps are relative to now() so date-bound pages
-- always have data.

-- =====================================================
-- Progress
-- =====================================================
INSERT INTO life_parts (id, user_id, name, description) VALUES
    (1, 1, 'Career', 'Work, side projects, professional growth'),
    (2, 1, 'Health', 'Physical fitness, sleep, nutrition'),
    (3, 1, 'Family', 'Parents, siblings, extended family');

INSERT INTO activities (id, user_id, life_part_ids, name, description, progress_type, frequency_days, started_at, ended_at, status) VALUES
    (1, 1, '{1}', 'Ship personal tracker', '**Refactor** web transport to the shared `action/webui` design system, replace the ad-hoc screenshot dashboards one subdomain at a time, and retire the old per-page CSS/JS once every view has a browse/detail equivalent built on the new components.', 'project_progress', 1, now() - interval '14 days', NULL, 'active'),
    (2, 1, '{2}', 'Gym', 'Push/pull/legs', 'habit_progress', 2, now() - interval '60 days', NULL, 'active'),
    (3, 1, '{}', 'Mood check-in', '', 'mood', 1, now() - interval '90 days', NULL, 'active'),
    (4, 1, '{2,3}', 'Call mom', '', 'promise_state', 7, now() - interval '30 days', NULL, 'active'),
    (5, 1, '{1}', 'Read "Designing Data-Intensive Applications"', '', 'project_progress', 3, now() - interval '120 days', now() - interval '20 days', 'finished'),
    (6, 1, '{2}', 'Morning run', '', 'habit_progress', 2, now() - interval '45 days', NULL, 'paused');

INSERT INTO steps (id, user_id, activity_id, name, type, status, created_at) VALUES
    (1, 1, 1, 'Wire up steps UI', 'one_time', 'active', now() - interval '2 days'),
    (2, 1, 1, 'Write E2E tests', 'one_time', 'active', now() - interval '1 day'),
    (3, 1, 2, 'Buy protein powder', 'repeatable', 'active', now() - interval '20 days');

INSERT INTO activity_progress (activity_id, user_id, value, note, progress_at, executed_step_id) VALUES
    (1, 1, 1, 'Wired up the transport/web package', now() - interval '20 hours', NULL),
    (1, 1, 2, 'Shipped cookie-based session login', now() - interval '3 days', NULL),
    (2, 1, 1, 'Leg day, felt strong', now() - interval '20 hours', NULL),
    (2, 1, 1, 'Push day', now() - interval '4 days', 3),
    (2, 1, 0, 'Short pull session', now() - interval '9 days', 3),
    (2, 1, 2, 'New squat PR', now() - interval '15 days', 3),
    (3, 1, 2, 'Bright and productive morning', now(), NULL),
    (3, 1, -1, 'Tired, slept badly', now() - interval '2 days', NULL),
    (4, 1, 1, 'Left a voice message', now() - interval '20 hours', NULL),
    (5, 1, 2, 'Finished the last chapter', now() - interval '20 days', NULL),
    (6, 1, 1, '5k easy pace', now() - interval '25 days', NULL);

UPDATE activities a
SET last_point_at = p.last_point_at
FROM (SELECT activity_id, max(progress_at) AS last_point_at FROM activity_progress GROUP BY activity_id) p
WHERE a.id = p.activity_id;

-- =====================================================
-- Workouts
-- =====================================================
INSERT INTO exercises (id, user_id, name, equipment_type) VALUES
    (1, 1, 'Bench Press', 'barbell'),
    (2, 1, 'Squat', 'barbell'),
    (3, 1, 'Pull-up', 'bodyweight');

INSERT INTO workouts (id, user_id, started_at, completed_at) VALUES
    (1, 1, now() - interval '28 days', now() - interval '28 days' + interval '1 hour'),
    (2, 1, now() - interval '21 days', now() - interval '21 days' + interval '1 hour'),
    (3, 1, now() - interval '14 days', now() - interval '14 days' + interval '1 hour'),
    (4, 1, now() - interval '7 days', now() - interval '7 days' + interval '1 hour');

INSERT INTO sets (user_id, workout_id, exercise_id, reps, weight_kg, created_at) VALUES
    (1, 1, 1, 10, 60, now() - interval '28 days'),
    (1, 1, 2, 10, 80, now() - interval '28 days' + interval '20 minutes'),
    (1, 2, 1, 8, 65, now() - interval '21 days'),
    (1, 2, 2, 8, 85, now() - interval '21 days' + interval '20 minutes'),
    (1, 2, 3, 8, NULL, now() - interval '21 days' + interval '40 minutes'),
    (1, 3, 1, 6, 70, now() - interval '14 days'),
    (1, 3, 2, 6, 90, now() - interval '14 days' + interval '20 minutes'),
    (1, 3, 3, 10, NULL, now() - interval '14 days' + interval '40 minutes'),
    (1, 4, 1, 5, 82.5, now() - interval '7 days'),
    (1, 4, 2, 5, 100, now() - interval '7 days' + interval '20 minutes'),
    (1, 4, 3, 12, NULL, now() - interval '7 days' + interval '40 minutes');

-- =====================================================
-- Money
-- =====================================================
INSERT INTO transactions (user_id, type, amount_original, currency, amount_eur, account, category, merchant, transacted_at, created_at) VALUES
    (1, 'income', 3500, 'EUR', 3500, 'Revolut', 'salary', 'Employer', now() - interval '95 days', now() - interval '95 days'),
    (1, 'expense', 700, 'EUR', 700, 'Revolut', 'rent', 'Landlord', now() - interval '90 days', now() - interval '90 days'),
    (1, 'income', 3500, 'EUR', 3500, 'Revolut', 'salary', 'Employer', now() - interval '65 days', now() - interval '65 days'),
    (1, 'expense', 700, 'EUR', 700, 'Revolut', 'rent', 'Landlord', now() - interval '60 days', now() - interval '60 days'),
    (1, 'expense', 320, 'EUR', 320, 'Revolut', 'groceries', 'Lidl', now() - interval '58 days', now() - interval '58 days'),
    (1, 'expense', 45, 'EUR', 45, 'Revolut', 'transport', 'Bolt', now() - interval '40 days', now() - interval '40 days'),
    (1, 'income', 3500, 'EUR', 3500, 'Revolut', 'salary', 'Employer', now() - interval '35 days', now() - interval '35 days'),
    (1, 'expense', 700, 'EUR', 700, 'Revolut', 'rent', 'Landlord', now() - interval '30 days', now() - interval '30 days'),
    (1, 'expense', 280, 'EUR', 280, 'Revolut', 'groceries', 'Lidl', now() - interval '15 days', now() - interval '15 days'),
    (1, 'expense', 64.5, 'EUR', 64.5, 'Revolut', 'food/restaurants', 'Meze House', now() - interval '8 days', now() - interval '8 days'),
    (1, 'income', 3500, 'EUR', 3500, 'Revolut', 'salary', 'Employer', now() - interval '5 days', now() - interval '5 days'),
    (1, 'expense', 32, 'EUR', 32, 'Revolut', 'food/cafe', 'Costa Coffee', now() - interval '3 days', now() - interval '1 day'),
    (1, 'expense', 18, 'EUR', 18, 'Revolut', 'transport', 'Bolt', now() - interval '1 day', now() - interval '1 day');

-- =====================================================
-- Achievements
-- =====================================================
INSERT INTO achievements (user_id, name, achievement_type, exercise_id, activity_id, target_value, current_value, details, starts_at, ends_at) VALUES
    (1, 'Emergency Fund', 'money_saving', NULL, NULL, 5000, 2100, '{"baseline_balance_eur": 1000}', now() - interval '2 months', now() + interval '3 months'),
    (1, 'Food Budget', 'money_spend', NULL, NULL, 400, 96.5, '{"category": "food"}', now() - interval '20 days', NULL),
    (1, 'Bench Press 100kg', 'exercise_max_weight', 1, NULL, 100, 82.5, '{"unit": "kg"}', now() - interval '1 month', NULL),
    (1, 'Gym 30 Times', 'activity_occurrence_count', NULL, 2, 30, 12, '{"unit": "sessions"}', now() - interval '14 days', NULL),
    (1, 'Read 12 Books', 'manual', NULL, NULL, 12, 5, '{"unit": "books"}', now() - interval '6 months', now() + interval '6 months');

-- =====================================================
-- Ideas
-- =====================================================
INSERT INTO ideas (id, user_id, body, status, resolution, merged_into_id, resolved_at, created_at, updated_at) VALUES
    (1, 1, 'Learn to sail', 'spike', NULL, NULL, NULL, now() - interval '2 months', now() - interval '3 days'),
    (2, 1, 'Build a bookshelf for the hallway', 'inbox', NULL, NULL, NULL, now() - interval '1 day', now() - interval '1 day'),
    (3, 1, 'Write a blog post about the personal tracker', 'someday', NULL, NULL, NULL, now() - interval '1 month', now() - interval '10 days'),
    (4, 1, 'Start a running club', 'resolved', 'blocked', NULL, now() - interval '80 days', now() - interval '3 months', now() - interval '80 days'),
    -- Merged duplicates: raise the surface count of ideas 1 and 3.
    (5, 1, 'Take a sailing course', 'resolved', 'merged', 1, now() - interval '20 days', now() - interval '20 days', now() - interval '20 days'),
    (6, 1, 'Sailing weekend in Limassol', 'resolved', 'merged', 1, now() - interval '3 days', now() - interval '3 days', now() - interval '3 days'),
    (7, 1, 'Blog: how the tracker is built', 'resolved', 'merged', 3, now() - interval '10 days', now() - interval '10 days', now() - interval '10 days');

-- Explicit ids above don't advance the serial sequences; bump them so rows
-- created from the preview's forms don't collide.
SELECT setval(pg_get_serial_sequence('life_parts', 'id'), (SELECT max(id) FROM life_parts));
SELECT setval(pg_get_serial_sequence('activities', 'id'), (SELECT max(id) FROM activities));
SELECT setval(pg_get_serial_sequence('steps', 'id'), (SELECT max(id) FROM steps));
SELECT setval(pg_get_serial_sequence('exercises', 'id'), (SELECT max(id) FROM exercises));
SELECT setval(pg_get_serial_sequence('workouts', 'id'), (SELECT max(id) FROM workouts));
SELECT setval(pg_get_serial_sequence('ideas', 'id'), (SELECT max(id) FROM ideas));
