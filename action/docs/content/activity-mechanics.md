# Механика системы

> Что есть в системе трекинга и как это устроено: сущности, поля, статусы, инструменты. Процесс — когда, зачем и что остаётся после — описан в `activity-rituals`. Здесь нормативных правил нет, только механика; если правило из `activity-rituals` опирается на поле или инструмент, оно ссылается сюда.
>
> Источник правды — схемы инструментов Progress MCP (снято 23.09.2026). При расхождении документа и MCP верен MCP, документ обновляется.

---

## 1. Обзор

| Сущность | Что это | Привязка |
|---|---|---|
| **Activity** | Всё, что трекается: привычка, проект, обещание, настроение | Life parts (1..N) |
| **Progress point** | Чекин: значение −2..+2 и заметка | Activity (1) |
| **Step** | Next-action: конкретное следующее действие | Activity (1) |
| **Idea** | Сырая мысль вне activity: inbox → someday → spike → решение | Progress point (0..1, при `promoted`) |
| **Achievement** | Прогресс-бар на аспект activity, упражнения или денег | Activity / exercise / категория трат (0..1) |
| **Life part** | Область жизни | — |
| **Событие календаря** | Слот в Google Calendar | Activity через `activity_id:XX` в description |

```
Life part ──< Activity >── Progress point
                 │   └──< Step
                 └── Achievement (0..N за жизнь)
Google Calendar event ··· activity_id:XX
Idea ··· Progress point (promoted)
```

---

## 2. Activity

### 2.1 Поля

| Поле | Тип | Смысл |
|---|---|---|
| `id` | int | Всегда упоминается вместе с названием: «17 «Греческий»» |
| `name` | string | Короткое название |
| `description` | string | Коротко: «Цель: … Средство: …». Без next-actions — они живут в steps |
| `progress_type` | enum | `mood` / `habit_progress` / `project_progress` / `promise_state` (2.2) |
| `frequency_days` | int | Мера внимания: как часто activity всплывает на ревью (1 — ежедневно, 7 — еженедельно, 30 — ежемесячно) |
| `status` | enum | `active` / `paused` / `finished` / `dropped` (2.3) |
| `deferred_until` | date | Дата «проверить» для паузы. Имеет смысл только при `status=paused` |
| `started_at` | datetime | Начало трекинга |
| `ended_at` | datetime | Дата завершения для `finished` / `dropped`. Пустая строка — переоткрыть |
| `life_part_ids` | int[] | Области жизни; при редактировании заменяются целиком |

### 2.2 Типы прогресса

| Тип | Для чего | В WIP-лимите считается как |
|---|---|---|
| `habit_progress` | Регулярное поведение (тренировки, язык, сон) | habit |
| `project_progress` | Работа с результатом (жильё, конференция, автоматизация) | project |
| `promise_state` | Обещание кому-то или себе (написать, связаться) | promise |
| `mood` | Состояние, а не действие | mood |

Смена типа через `edit_activity` не пересчитывает старые чекины — их значения остаются в семантике прежнего типа.

### 2.3 Статусы

```
            ┌──────── edit_activity(status=paused, deferred_until) ───────┐
            │                                                             ▼
  create → active ◄──── edit_activity(status=active, deferred_until="") ─ paused
            │
            ├── edit_activity(status=finished, ended_at) → finished
            └── edit_activity(status=dropped,  ended_at) → dropped
                                 ▲
             edit_activity(ended_at="", status=active) — переоткрыть
```

| Статус | Видимость | История |
|---|---|---|
| `active` | `get_activity_list(active_only=true)` → `activities`, по срочности чекина | Сохраняется |
| `paused` | Там же, отдельным списком `paused_activities` с `deferred_until` | Сохраняется |
| `finished` / `dropped` | `get_activity_list(active_only=false)`, по `ended_at` | Сохраняется |
| удалена | Нигде | **Уничтожается** вместе с чекинами |

- Отдельного `finish_activity` нет — завершение через `edit_activity(status, ended_at)`.
- `delete_activity` — безвозвратное удаление activity и всех её чекинов. Падает, если на activity ссылается achievement.
- **Legacy-паузы:** часть старых activity «поставлена на паузу» сдвигом `started_at` в будущее — до появления `status=paused` и `deferred_until`. Такие activity не видны в `get_activity_list(active_only=true)`. Их разбор — в миграции (`rituals.md`, «Разово»).

### 2.4 Служебные activity

Инфраструктура процесса, вне WIP-лимита (`rituals.md`, 2.6):

| Activity | Назначение |
|---|---|
| 70 «Еженедельное планирование» | Итоги ритуалов: один progress point `value=1` на сессию |
| 67 «Дневник и заметки» | Только личная рефлексия; трудные дни — `value=-1` |

### 2.5 Inbox: идеи

Inbox — отдельная сущность `ideas`, не activity (`rituals.md`, 2.2–2.5).

| Поле | Тип | Смысл |
|---|---|---|
| `id` | int | `idea_id` |
| `body` | string | Мои слова; итог спайка и новые мысли дописываются через пустую строку, не перезаписываются |
| `status` | enum | `inbox` / `someday` / `spike` / `resolved` |
| `resolution` | enum | Только при `resolved`: `dropped` / `merged` / `expired` / `promoted` / `blocked` |
| `merged_into_id` | int | Только `merged`: старая идея |
| `resolved_progress_point_id` | int | Только `promoted`: точка, которой идея стала; NULL, если точку удалили |
| `resolved_at`, `created_at`, `updated_at` | datetime | `updated_at` — последняя смена статуса или дописанный текст |
| `surface_count` | int | Только чтение: 1 + число идей, слитых в эту |
| `last_surfaced_at` | datetime | Только чтение: последний `created_at` среди идеи и её дублей |

- Переходы `update_idea`: `inbox → someday | spike`, `someday → spike`, `spike → someday`. В `inbox` ничего не возвращается; повтор текущего статуса — no-op.
- В `resolved` — только `resolve_idea`. `blocked` — только из `spike`. Пересмотреть можно только `blocked`: в `promoted` / `dropped` / `expired`, `resolved_at` перезаписывается.
- `merged` в свою же, чужую или решённую идею не проходит; дубли слитой идеи переходят к цели.
- `promoted` требует свою точку. Activity и steps идеи видны через точку: её `activity_id` и steps с `created_by_progress_point_id`.
- Физического удаления идей нет.
- Web: `/web/ideas` — форма захвата и открытые идеи по статусам плюс `blocked`.

---

## 3. Progress point

### 3.1 Поля

| Поле | Тип | Смысл |
|---|---|---|
| `id` | int | `progress_id` для правки и удаления |
| `activity_id` | int | Чья точка |
| `value` | int −2..+2 | Оценка (3.2) |
| `note` | string | Что я сказал, без интерпретаций |
| `progress_at` | datetime | Когда было; по умолчанию сейчас, можно задним числом |
| `hours_left` | number | Только для проектов: оценка оставшихся часов |
| `executed_step_id` | int | Repeatable step этой activity, выполненный в этом чекине; максимум один |

### 3.2 Шкала по типам

| value | habit_progress | project_progress | promise_state | mood |
|---|---|---|---|---|
| +2 | crushing it 💪 / blooming 🌸 | breakthrough 🚀 / sprinting 🏃 | — | sunny ☀️ / bright ✨ |
| +1 | mostly doing 👍 / growing 🌱 | moving forward ➡️ / walking 🚶 | did something ✅ / burning 🔥 | partly cloudy ⛅ / light 💡 |
| 0 | trying 🤔 / planted 🌰 | stuck ⏸️ / resting 🧘 | remember 💭 / lit 🕯️ | overcast ☁️ / dim 🕯️ |
| −1 | rarely 😔 / wilting 🥀 | setback ↩️ / backtracking 🔙 | forgot 🤷 / extinguished 💨 | rainy 🌧️ / dark 🌑 |
| −2 | not doing ❌ / withered 🍂 | changed plans 🔄 / lost 🗺️ | — | stormy ⛈️ / pitch black ⚫ |

- У `promise_state` шкала фактически −1..+1.
- Полный список метафор — `get_progress_type_examples`.

### 3.3 Правка и удаление

- `edit_progress_point(progress_id, …)` — поправить `value`, `note`, `progress_at`, `hours_left`. Предпочтительнее, чем добавлять корректирующую точку.
- `delete_progress_point(progress_id)` — **жёсткое** удаление, возвращает удалённое для подтверждения. Soft delete нет: решения по мыслям inbox живут в идеях (2.5).

### 3.4 Чтение

- `get_activity_stats(activity_id)` — последние 3 точки + тренды: за всё время, месяц, неделю (count, среднее, 80-й перцентиль). Только по одной activity; пакетного вызова нет.
- `search_progress_notes(query_variants[1..5], activity_id?, from?, to?, value_min?, value_max?)` — поиск по заметкам, ранжирование по числу совпавших вариантов.

---

## 4. Step

### 4.1 Поля

| Поле | Тип | Смысл |
|---|---|---|
| `id` | int | `step_id` |
| `activity_id` | int | Чей шаг |
| `name` | string | Физическое действие, близко к моим словам |
| `type` | enum | `one_time` — сделал и закрыл; `repeatable` — повторяется, пока жива activity |
| `status` | enum | `active` / `finished` (закрытие ставит `closed_at`) |
| `created_by_progress_point_id` | int | Чекин, которым шаг создан; `create_step` — необязательный параметр |
| `completed_by_progress_point_id` | int | Чекин, которым шаг закрыт |
| `last_executed_at` | datetime | Только чтение: `progress_at` последнего чекина с этим `executed_step_id` |
| `executions_last_30_days` | int | Только чтение: сколько чекинов выполнили step за 30 дней |

### 4.2 Поведение

- Статуса `dropped` у step нет: ненужный step удаляется `delete_step`.
- `get_step_list` отдаёт steps **только активных activity**. У paused / finished / dropped activity steps не всплывают автоматически.
- `create_step` связывает step с чекином, только если передан `created_by_progress_point_id` (своя точка той же activity) — например, для идеи, ставшей step. Веб-форма чекина связывает созданный вместе с ней step с этой точкой.
- Отметка в чекине: one_time закрывается; repeatable остаётся `active`, а чекин получает `executed_step_id` (MCP — параметр `create_progress_point`, веб — radio-группа). Step должен быть active repeatable той же activity. Закрыть repeatable можно только явно — `edit_step status=finished`.
- Удаление step не удаляет чекины, только обнуляет их `executed_step_id`.

---

## 5. Achievement

> **Achievement** — не цель и не финишная черта activity, а прогресс-бар на отдельный аспект — для мотивации на втором экране.

### 5.1 Типы

| `achievement_type` | Что считает | Нужно указать | `current_value` |
|---|---|---|---|
| `activity_occurrence_count` | Число чекинов activity за период | `activity_id` | Считается |
| `activity_streak_count` | Серия чекинов activity подряд | `activity_id` | Считается |
| `exercise_max_weight` | Максимальный вес в упражнении | `exercise_id` | Считается |
| `exercise_total_volume` | Суммарный объём по упражнению | `exercise_id` | Считается |
| `money_saving` | Накоплено | — | Считается |
| `money_spend` | Потрачено в категории (лимит) | `category` | Считается |
| `manual` | Что угодно | `unit` | Вручную: `log_achievement_progress(delta)` |

### 5.2 Поля и поведение

- Общие поля: `name`, `target_value`, `starts_at`, `ends_at` (срок), `unit`.
- `current_value` — кэш. Для шести вычисляемых типов обновляется `refresh_achievements` (все или один); `get_achievement_progress` его не пересчитывает.
- `get_achievement_progress(date?)` — все achievement, активные на дату, с `remaining_value`.
- `update_achievement` — `name`, `target_value`, `ends_at` / `clear_ends_at`, `unit`, `category` (только `money_spend`), `current_value` (только `manual`).
- Удалить activity, на которую ссылается achievement, нельзя, пока achievement не удалён или не перепривязан.

---

## 6. Life part

| id | Название | Цель области |
|---|---|---|
| 1 | Тело | Сохранять и развивать здоровье, подтянутость |
| 3 | Эмоции | Делать то, что развивает, развлекает, приносит хорошие эмоции; бонус — связи или бизнес |
| 4 | Дисциплина | Навыки постановки и достижения целей для поддержки других областей |
| 5 | Карьера | Обрастать профессиональными связями и рекомендациями |
| 6 | Семья | Не терять связь с родными, развивать отношения и взаимопомощь |

- `list_life_parts` — только чтение. Создаются и переименовываются вручную в БД, поэтому ID сверять в начале сессии, а не держать в голове.
- У activity может быть несколько life parts.

---

## 7. Календарь

- Google Calendar, часовой пояс `Asia/Nicosia`.
- Связь с activity — строка `activity_id:XX` в `description` события. Других связей нет: MCP о событиях не знает, сопоставление делает агент.
- Исключения повторяющихся событий: `[eventId]_[YYYYMMDDTHHMMSSZ]` — работает ненадёжно, ручные удаления повторов делаю сам.
- inDrive фокус-слот живёт в рабочем календаре и заводится автоматически.

---

## 8. Смежные модули MCP

Не часть трекинга activity, но на них опираются achievement и месячный финансовый срез.

| Модуль | Что есть | Где используется |
|---|---|---|
| Финансы | Транзакции (expense / income / transfer), `get_balance`, `get_spending_by_category`, `compare_periods`, `get_top_merchants`; суммы в EUR | Achievement `money_*`; месячный срез (`rituals.md`, 3.3) |
| Тренировки | Упражнения, подходы, `get_personal_records`, `get_exercise_history` | Achievement `exercise_*` |
| Питание | Продукты, логирование, `get_nutrition_stats` | — |
| Telegram | `send_telegram_message` — сообщение мне вне сессии | — |

---

## 9. Расхождения механики и процесса

Что `rituals.md` уже предполагает, а механика пока не поддерживает.

| Процесс требует | Сейчас в MCP | Что сделать |
|---|---|---|
| Лимит 6 achievement, WIP 6 / 3 на тип | Не проверяется | Считает агент на ревью; возможно — проверка в `create_*` |
| Паузы только через `status=paused` + `deferred_until` | Остались legacy-паузы через `started_at` | Миграция |

---

## 10. Шпаргалка: операция → инструмент

| Операция | Инструмент |
|---|---|
| Список activity (активные + paused / завершённые) | `get_activity_list(active_only)` |
| Статистика activity | `get_activity_stats(activity_id)` |
| Создать activity | `create_activity(name, progress_type, frequency_days, description?, life_part_ids?, started_at?)` |
| Изменить / пауза / завершить / переоткрыть | `edit_activity(activity_id, …)` |
| Удалить activity навсегда | `delete_activity` — только по явной просьбе |
| Чекин | `create_progress_point(activity_id, value, note?, progress_at?, hours_left?, executed_step_id?)` |
| Поправить чекин | `edit_progress_point(progress_id, …)` |
| Удалить чекин | `delete_progress_point(progress_id)` |
| Поиск по заметкам | `search_progress_notes(query_variants, …)` |
| Метафоры шкалы | `get_progress_type_examples` |
| Steps: список / создать / изменить / удалить | `get_step_list` / `create_step(…, created_by_progress_point_id?)` / `edit_step` / `delete_step` |
| Записать идею | `search_ideas(query_variants)` по дублям, затем `create_idea(body)` |
| Идеи по статусу / решённые за период | `list_ideas(statuses?, resolutions?, resolved_from?, resolved_to?)` |
| Поиск похожих идей | `search_ideas(query_variants[1..5], statuses?)` — по умолчанию все статусы, включая решённые |
| Дописать / сменить статус идеи | `update_idea(idea_id, append_body?, status?)` |
| Решение по идее | `resolve_idea(idea_id, resolution, merged_into_id?, resolved_progress_point_id?)` |
| Achievements: список / создать / изменить / +delta / пересчитать | `get_achievement_progress` / `create_achievement` / `update_achievement` / `log_achievement_progress` / `refresh_achievements` |
| Life parts | `list_life_parts` |