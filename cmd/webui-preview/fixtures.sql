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

-- =====================================================
-- Docs
-- =====================================================
INSERT INTO docs (id, user_id, title, description, created_at, updated_at) VALUES
    (1, 1, 'Development plan', 'Where I am going this year and what I do about it each quarter', now() - interval '40 days', now() - interval '2 days'),
    (2, 1, 'Personal instructions', 'How the agent should work with me: tone, defaults, things to avoid', now() - interval '25 days', now() - interval '6 hours');

INSERT INTO doc_chunks (user_id, doc_id, position, body, previous_body, created_at, updated_at) VALUES
    (1, 1, 1, '# Direction

Become a **staff engineer** who owns a product area end to end, without giving up health and family time.', NULL, now() - interval '40 days', now() - interval '40 days'),
    (1, 1, 2, '## This quarter

- Ship the personal tracker web UI
- Read *Designing Data-Intensive Applications* to the end
- Give one internal talk on what I learned', '## This quarter

- Ship the personal tracker web UI
- Read *Designing Data-Intensive Applications*', now() - interval '40 days', now() - interval '2 days'),
    (1, 1, 3, '## Review rhythm

1. Weekly: look at the steps of every active activity
2. Monthly: update this plan
3. Quarterly: rewrite the "This quarter" section

```mermaid
graph LR
    Weekly[Weekly: steps] --> Monthly[Monthly: plan]
    Monthly --> Quarterly[Quarterly: rewrite]
    Quarterly --> Weekly
```

> A plan that is not reviewed is a wish.', NULL, now() - interval '38 days', now() - interval '38 days'),
    (1, 2, 1, '# Tone

Short and direct. No praise, no filler. Answer in the language I write in.', NULL, now() - interval '25 days', now() - interval '25 days'),
    (1, 2, 2, '## Defaults

- Times are in `Asia/Nicosia`
- Money is in EUR
- When logging food, ask for the weight if I did not give one', '## Defaults

- Times are in `Asia/Nicosia`
- Money is in EUR', now() - interval '25 days', now() - interval '6 hours');

-- =====================================================
-- Learning
-- =====================================================
-- Planned days are stored as midnight Asia/Nicosia; pg_temp.learning_day(n)
-- is that midnight n days from today.
CREATE FUNCTION pg_temp.learning_day(n int) RETURNS timestamptz AS $$
    SELECT (date_trunc('day', now() AT TIME ZONE 'Asia/Nicosia') + make_interval(days => n)) AT TIME ZONE 'Asia/Nicosia'
$$ LANGUAGE sql;

INSERT INTO learning_plans (id, user_id, name, description, status, created_at, updated_at) VALUES
    (1, 1, 'Greek A1', 'Order food, ask for directions and small talk in Greek by summer', 'active', now() - interval '70 days', now() - interval '2 days'),
    (2, 1, 'Kubernetes fundamentals', 'Deploy and debug a small app on a local cluster', 'active', now() - interval '20 days', now() - interval '20 days'),
    (3, 1, 'Greek A2', NULL, 'paused', now() - interval '10 days', now() - interval '10 days'),
    (4, 1, 'Spanish basics', 'Holiday phrases', 'completed', now() - interval '300 days', now() - interval '200 days');

INSERT INTO skills (id, user_id, learning_plan_id, name, created_at, updated_at) VALUES
    (1, 1, 1, 'Present tense: είμαι, έχω', now() - interval '70 days', now() - interval '70 days'),
    (2, 1, 1, 'Definite articles', now() - interval '69 days', now() - interval '69 days'),
    (3, 1, 1, 'Food & drinks', now() - interval '30 days', now() - interval '30 days'),
    (4, 1, 1, 'Past tense (aorist)', now() - interval '3 days', now() - interval '3 days'),
    (5, 1, 2, 'Pods & deployments', now() - interval '20 days', now() - interval '20 days'),
    (6, 1, 2, 'Services & networking', now() - interval '20 days', now() - interval '20 days');

INSERT INTO learning_exercises (user_id, skill_id, kind, content, planned_for, answer, answered_at, score, note, reviewed_at, created_at) VALUES
    -- Present tense: four perfect results this month → 80%, practiced recently
    (1, 1, 'put into the correct form', E'1. Εγώ ___ (είμαι) φοιτητής.\n2. Εσύ ___ (έχω) σκύλο;\n3. Αυτή ___ (είμαι) στο σπίτι.', pg_temp.learning_day(-10), E'1. Εγώ είμαι φοιτητής.\n2. Εσύ έχεις σκύλο;\n3. Αυτή είναι στο σπίτι.', pg_temp.learning_day(-10) + interval '19 hours', 5, NULL, pg_temp.learning_day(-3) + interval '10 hours', now() - interval '12 days'),
    (1, 1, 'put into the correct form', E'1. Εμείς ___ (έχω) χρόνο.\n2. Εσείς ___ (είμαι) εδώ;', pg_temp.learning_day(-8), E'1. Εμείς έχουμε χρόνο.\n2. Εσείς είστε εδώ;', pg_temp.learning_day(-8) + interval '20 hours', 5, NULL, pg_temp.learning_day(-3) + interval '10 hours', now() - interval '12 days'),
    (1, 1, 'translate RU→EL', E'1. У меня есть кот.\n2. Они дома.', pg_temp.learning_day(-6), E'1. Έχω γάτα.\n2. Είναι στο σπίτι.', pg_temp.learning_day(-6) + interval '18 hours', 5, NULL, pg_temp.learning_day(-3) + interval '10 hours', now() - interval '12 days'),
    (1, 1, 'translate RU→EL', E'1. Ты студент?\n2. У нас нет машины.', pg_temp.learning_day(-4), E'1. Είσαι φοιτητής;\n2. Δεν έχουμε αυτοκινητο.', pg_temp.learning_day(-4) + interval '21 hours', 5, 'Accent missing: αυτοκίνητο', pg_temp.learning_day(-3) + interval '10 hours', now() - interval '12 days'),
    (1, 1, 'choose the ending', E'Εμείς είμ___ (-αστε / -αι) καλά.', pg_temp.learning_day(1), NULL, NULL, NULL, NULL, NULL, now() - interval '2 days'),
    -- Definite articles: mastered long ago, now slipping (−20% debuff)
    (1, 2, 'choose the article', E'___ σπίτι, ___ γάτα, ___ σκύλος', pg_temp.learning_day(-60), E'το σπίτι, η γάτα, ο σκύλος', now() - interval '57 days', 5, NULL, now() - interval '55 days', now() - interval '62 days'),
    (1, 2, 'choose the article', E'___ νερό, ___ μητέρα, ___ δρόμος', pg_temp.learning_day(-58), E'το νερό, η μητέρα, ο δρόμος', now() - interval '55 days', 5, NULL, now() - interval '54 days', now() - interval '62 days'),
    (1, 2, 'plural articles', E'___ σπίτια, ___ γάτες', pg_temp.learning_day(-56), E'τα σπίτια, οι γάτες', now() - interval '53 days', 5, NULL, now() - interval '52 days', now() - interval '62 days'),
    (1, 2, 'plural articles', E'___ σκύλοι, ___ μητέρες', pg_temp.learning_day(-54), E'οι σκύλοι, οι μητέρες', now() - interval '51 days', 5, NULL, now() - interval '51 days', now() - interval '62 days'),
    -- Food & drinks: mixed results, one answer waiting for the weekly analysis
    (1, 3, 'translate RU→EL', 'Я хочу кофе и воду, пожалуйста.', pg_temp.learning_day(-9), 'Θέλω καφές και νερό, παρακαλώ.', pg_temp.learning_day(-9) + interval '19 hours', 4, 'Accusative: καφέ, not καφές', pg_temp.learning_day(-3) + interval '10 hours', now() - interval '12 days'),
    (1, 3, 'dialogue', E'Закажи в таверне:\n- салат\n- хлеб\n- бокал вина', pg_temp.learning_day(-7), E'Ένα σαλάτα, ψωμί και ένα ποτήρι κρασί.', pg_temp.learning_day(-7) + interval '21 hours', 3, E'Μία σαλάτα (feminine).\nMissing «θα ήθελα» — too blunt for a waiter.', pg_temp.learning_day(-3) + interval '10 hours', now() - interval '12 days'),
    (1, 3, 'translate EL→RU', 'Το τυρί είναι πολύ νόστιμο.', pg_temp.learning_day(-5), 'Сыр очень вкусный.', pg_temp.learning_day(-5) + interval '18 hours', 3, NULL, pg_temp.learning_day(-3) + interval '10 hours', now() - interval '12 days'),
    (1, 3, 'translate RU→EL', E'1. Счёт, пожалуйста.\n2. Можно меню?', pg_temp.learning_day(-1), E'1. Τον λογαριασμό, παρακαλώ.\n2. Μπορώ το μενού;', pg_temp.learning_day(-1) + interval '20 hours', NULL, NULL, NULL, now() - interval '2 days'),
    -- Past tense: new skill, only planned (one missed, one due today, one ahead)
    (1, 4, 'put into the correct form', E'Χθες ___ (πηγαίνω) στη θάλασσα.', pg_temp.learning_day(-1), NULL, NULL, NULL, NULL, NULL, now() - interval '2 days'),
    (1, 4, 'translate RU→EL', 'Вчера я ел рыбу.', pg_temp.learning_day(0), NULL, NULL, NULL, NULL, NULL, now() - interval '2 days'),
    (1, 4, 'put into the correct form', E'Πέρσι ___ (μένω) στην Αθήνα.', pg_temp.learning_day(2), NULL, NULL, NULL, NULL, NULL, now() - interval '2 days'),
    -- Kubernetes: practice-heavy, no vocabulary
    (1, 5, 'kubectl', 'Create a deployment nginx with 3 replicas and check that every pod is Running.', pg_temp.learning_day(-6), E'kubectl create deployment nginx --image=nginx --replicas=3\nkubectl get pods -l app=nginx', pg_temp.learning_day(-6) + interval '21 hours', 5, NULL, pg_temp.learning_day(-3) + interval '11 hours', now() - interval '8 days'),
    (1, 5, 'debugging', 'A pod is in CrashLoopBackOff. Find out why.', pg_temp.learning_day(-4), E'kubectl describe pod web-1\nkubectl logs web-1', pg_temp.learning_day(-4) + interval '22 hours', 4, 'Add --previous to see the logs of the crashed container', pg_temp.learning_day(-3) + interval '11 hours', now() - interval '8 days'),
    (1, 5, 'kubectl', 'Roll back the nginx deployment to the previous revision.', pg_temp.learning_day(0), NULL, NULL, NULL, NULL, NULL, now() - interval '2 days'),
    (1, 6, 'kubectl', 'Expose the nginx deployment on port 80 inside the cluster and curl it from another pod.', pg_temp.learning_day(3), NULL, NULL, NULL, NULL, NULL, now() - interval '2 days');

INSERT INTO vocabulary (id, user_id, learning_plan_id, skill_id, term, translation, notes, created_at, updated_at) VALUES
    (1, 1, 1, 3, 'νερό', 'вода', 'το νερό, neuter', now() - interval '60 days', now() - interval '60 days'),
    (2, 1, 1, 3, 'ψωμί', 'хлеб', 'το ψωμί', now() - interval '60 days', now() - interval '60 days'),
    (3, 1, 1, 3, 'γάλα', 'молоко', E'το γάλα\nplural: τα γάλατα', now() - interval '40 days', now() - interval '40 days'),
    (4, 1, 1, 3, 'καφές', 'кофе', 'ο καφές, acc. τον καφέ', now() - interval '30 days', now() - interval '30 days'),
    (5, 1, 1, 3, 'τυρί', 'сыр', NULL, now() - interval '3 days', now() - interval '3 days'),
    (6, 1, 1, 3, 'κρασί', 'вино', NULL, now() - interval '2 days', now() - interval '2 days'),
    (7, 1, 1, NULL, 'σπίτι', 'дом', 'το σπίτι', now() - interval '50 days', now() - interval '50 days'),
    (8, 1, 1, NULL, 'καλημέρα', 'доброе утро', NULL, now() - interval '65 days', now() - interval '65 days'),
    (9, 1, 1, NULL, 'ευχαριστώ', 'спасибо', NULL, now() - interval '65 days', now() - interval '65 days'),
    (10, 1, 1, NULL, 'παρακαλώ', 'пожалуйста', 'also «you''re welcome»', now() - interval '65 days', now() - interval '65 days'),
    (11, 1, 1, 1, 'φοιτητής', 'студент', 'η φοιτήτρια — female', now() - interval '12 days', now() - interval '12 days'),
    (12, 1, 1, NULL, 'λογαριασμός', 'счёт', 'τον λογαριασμό, παρακαλώ', now() - interval '1 day', now() - interval '1 day'),
    (13, 1, 3, NULL, 'ταξίδι', 'путешествие', NULL, now() - interval '10 days', now() - interval '10 days'),
    (14, 1, 3, NULL, 'αεροδρόμιο', 'аэропорт', NULL, now() - interval '10 days', now() - interval '10 days');

INSERT INTO vocabulary_reviews (user_id, vocabulary_id, planned_for, score, reviewed_at, created_at) VALUES
    -- νερό: mastered and fresh, next review in a few days
    (1, 1, pg_temp.learning_day(-14), 5, pg_temp.learning_day(-14) + interval '8 hours', now() - interval '15 days'),
    (1, 1, pg_temp.learning_day(-10), 5, pg_temp.learning_day(-10) + interval '8 hours', now() - interval '15 days'),
    (1, 1, pg_temp.learning_day(-6), 5, pg_temp.learning_day(-6) + interval '8 hours', now() - interval '15 days'),
    (1, 1, pg_temp.learning_day(-2), 5, pg_temp.learning_day(-2) + interval '8 hours', now() - interval '15 days'),
    (1, 1, pg_temp.learning_day(4), NULL, NULL, now() - interval '2 days'),
    -- ψωμί: mastered 51 days ago, slipping; due today
    (1, 2, pg_temp.learning_day(-57), 5, now() - interval '57 days', now() - interval '58 days'),
    (1, 2, pg_temp.learning_day(-55), 5, now() - interval '55 days', now() - interval '58 days'),
    (1, 2, pg_temp.learning_day(-53), 5, now() - interval '53 days', now() - interval '58 days'),
    (1, 2, pg_temp.learning_day(-51), 5, now() - interval '51 days', now() - interval '58 days'),
    (1, 2, pg_temp.learning_day(0), NULL, NULL, now() - interval '2 days'),
    -- γάλα: struggling; a missed review and today's one, so it is shown twice
    (1, 3, pg_temp.learning_day(-12), 5, pg_temp.learning_day(-12) + interval '9 hours', now() - interval '13 days'),
    (1, 3, pg_temp.learning_day(-8), 3, pg_temp.learning_day(-8) + interval '9 hours', now() - interval '13 days'),
    (1, 3, pg_temp.learning_day(-4), 2, pg_temp.learning_day(-4) + interval '9 hours', now() - interval '13 days'),
    (1, 3, pg_temp.learning_day(-2), NULL, NULL, now() - interval '6 days'),
    (1, 3, pg_temp.learning_day(0), NULL, NULL, now() - interval '6 days'),
    -- καφές: flashcard plus an indirect review from an exercise answer, next review tomorrow
    (1, 4, pg_temp.learning_day(-7), 5, pg_temp.learning_day(-7) + interval '9 hours', now() - interval '8 days'),
    (1, 4, pg_temp.learning_day(-3), 4, pg_temp.learning_day(-3) + interval '10 hours', now() - interval '3 days'),
    (1, 4, pg_temp.learning_day(1), NULL, NULL, now() - interval '3 days'),
    -- τυρί: first contact today
    (1, 5, pg_temp.learning_day(0), NULL, NULL, now() - interval '3 days'),
    -- κρασί: unscheduled, never reviewed
    -- σπίτι: good and recent, next review in a week
    (1, 7, pg_temp.learning_day(-20), 5, pg_temp.learning_day(-20) + interval '8 hours', now() - interval '21 days'),
    (1, 7, pg_temp.learning_day(-12), 5, pg_temp.learning_day(-12) + interval '8 hours', now() - interval '21 days'),
    (1, 7, pg_temp.learning_day(-5), 5, pg_temp.learning_day(-5) + interval '8 hours', now() - interval '21 days'),
    (1, 7, pg_temp.learning_day(7), NULL, NULL, now() - interval '5 days'),
    -- καλημέρα, ευχαριστώ: fully mastered long ago, unscheduled
    (1, 8, pg_temp.learning_day(-64), 5, now() - interval '64 days', now() - interval '65 days'),
    (1, 8, pg_temp.learning_day(-60), 5, now() - interval '60 days', now() - interval '65 days'),
    (1, 8, pg_temp.learning_day(-50), 5, now() - interval '50 days', now() - interval '65 days'),
    (1, 8, pg_temp.learning_day(-30), 5, now() - interval '30 days', now() - interval '65 days'),
    (1, 8, pg_temp.learning_day(-10), 5, now() - interval '10 days', now() - interval '65 days'),
    (1, 9, pg_temp.learning_day(-64), 5, now() - interval '64 days', now() - interval '65 days'),
    (1, 9, pg_temp.learning_day(-60), 4, now() - interval '60 days', now() - interval '65 days'),
    (1, 9, pg_temp.learning_day(-40), 5, now() - interval '40 days', now() - interval '65 days'),
    (1, 9, pg_temp.learning_day(-20), 5, now() - interval '20 days', now() - interval '65 days'),
    -- παρακαλώ: guessed wrong once, due tomorrow
    (1, 10, pg_temp.learning_day(-15), 5, pg_temp.learning_day(-15) + interval '8 hours', now() - interval '16 days'),
    (1, 10, pg_temp.learning_day(-6), 1, pg_temp.learning_day(-6) + interval '8 hours', now() - interval '16 days'),
    (1, 10, pg_temp.learning_day(1), NULL, NULL, now() - interval '6 days'),
    -- φοιτητής: missed review two days ago
    (1, 11, pg_temp.learning_day(-2), NULL, NULL, now() - interval '12 days'),
    -- Greek A2 (paused)
    (1, 13, pg_temp.learning_day(-9), 3, pg_temp.learning_day(-9) + interval '9 hours', now() - interval '10 days');


-- created from the preview's forms don't collide.
SELECT setval(pg_get_serial_sequence('life_parts', 'id'), (SELECT max(id) FROM life_parts));
SELECT setval(pg_get_serial_sequence('activities', 'id'), (SELECT max(id) FROM activities));
SELECT setval(pg_get_serial_sequence('steps', 'id'), (SELECT max(id) FROM steps));
SELECT setval(pg_get_serial_sequence('exercises', 'id'), (SELECT max(id) FROM exercises));
SELECT setval(pg_get_serial_sequence('workouts', 'id'), (SELECT max(id) FROM workouts));
SELECT setval(pg_get_serial_sequence('ideas', 'id'), (SELECT max(id) FROM ideas));
SELECT setval(pg_get_serial_sequence('docs', 'id'), (SELECT max(id) FROM docs));
SELECT setval(pg_get_serial_sequence('learning_plans', 'id'), (SELECT max(id) FROM learning_plans));
SELECT setval(pg_get_serial_sequence('skills', 'id'), (SELECT max(id) FROM skills));
SELECT setval(pg_get_serial_sequence('vocabulary', 'id'), (SELECT max(id) FROM vocabulary));
