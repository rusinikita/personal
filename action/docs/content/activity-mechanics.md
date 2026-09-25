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
| **Goal** | Achievement: прогресс-бар на аспект activity, упражнения или денег | Activity / exercise / категория трат (0..1) |
| **Life part** | Область жизни | — |
| **Событие календаря** | Слот в Google Calendar | Activity через `activity_id:XX` в description |

```
Life part ──< Activity >── Progress point
                 │   └──< Step
                 └── Goal (0..N за жизнь)
Google Calendar event ··· activity_id:XX
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
- `delete_activity` — безвозвратное удаление activity и всех её чекинов. Падает, если на activity ссылается goal.
- **Legacy-паузы:** часть старых activity «поставлена на паузу» сдвигом `started_at` в будущее — до появления `status=paused` и `deferred_until`. Такие activity не видны в `get_activity_list(active_only=true)`. Их разбор — в миграции (`rituals.md`, «Разово»).

### 2.4 Служебные activity

Инфраструктура процесса, вне WIP-лимита (`rituals.md`, 2.6):

| Activity | Назначение |
|---|---|
| «Inbox» | Сырые мысли: одна мысль — один progress point с `value=0`. Спайки — steps на ней. *Ещё не создана* |
| 70 «Еженедельное планирование» | Итоги ритуалов: один progress point `value=1` на сессию |
| 67 «Дневник и заметки» | Только личная рефлексия; трудные дни — `value=-1` |

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
- `delete_progress_point(progress_id)` — **жёсткое** удаление, возвращает удалённое для подтверждения. Soft delete пока не реализован (раздел 9).

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
| `completed_by_progress_point_id` | int | Чекин, которым шаг закрыт |

### 4.2 Поведение

- Статуса `dropped` у step нет: ненужный step удаляется `delete_step`.
- `get_step_list` отдаёт steps **только активных activity**. У paused / finished / dropped activity steps не всплывают автоматически.
- `create_step` через MCP не связывает step с чекином. Веб-форма чекина связывает созданный вместе с ней step с этой точкой.

---

## 5. Goal

> **Goal = achievement.** Не цель и не финишная черта activity, а прогресс-бар на отдельный аспект — для мотивации на втором экране. Название сущности историческое.

### 5.1 Типы

| `goal_type` | Что считает | Нужно указать | `current_value` |
|---|---|---|---|
| `activity_occurrence_count` | Число чекинов activity за период | `activity_id` | Считается |
| `activity_streak_count` | Серия чекинов activity подряд | `activity_id` | Считается |
| `exercise_max_weight` | Максимальный вес в упражнении | `exercise_id` | Считается |
| `exercise_total_volume` | Суммарный объём по упражнению | `exercise_id` | Считается |
| `money_saving` | Накоплено | — | Считается |
| `money_spend` | Потрачено в категории (лимит) | `category` | Считается |
| `manual` | Что угодно | `unit` | Вручную: `log_goal_progress(delta)` |

### 5.2 Поля и поведение

- Общие поля: `name`, `target_value`, `starts_at`, `ends_at` (срок), `unit`.
- `current_value` — кэш. Для шести вычисляемых типов обновляется `refresh_goals` (все или один); `get_goal_progress` его не пересчитывает.
- `get_goal_progress(date?)` — все goal, активные на дату, с `remaining_value`.
- `update_goal` — `name`, `target_value`, `ends_at` / `clear_ends_at`, `unit`, `category` (только `money_spend`), `current_value` (только `manual`).
- Удалить activity, на которую ссылается goal, нельзя, пока goal не удалён или не перепривязан.

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

Не часть трекинга activity, но на них опираются goal и месячный финансовый срез.

| Модуль | Что есть | Где используется |
|---|---|---|
| Финансы | Транзакции (expense / income / transfer), `get_balance`, `get_spending_by_category`, `compare_periods`, `get_top_merchants`; суммы в EUR | Goal `money_*`; месячный срез (`rituals.md`, 3.3) |
| Тренировки | Упражнения, подходы, `get_personal_records`, `get_exercise_history` | Goal `exercise_*` |
| Питание | Продукты, логирование, `get_nutrition_stats` | — |
| Telegram | `send_telegram_message` — сообщение мне вне сессии | — |

---

## 9. Расхождения механики и процесса

Что `rituals.md` уже предполагает, а механика пока не поддерживает.

| Процесс требует | Сейчас в MCP | Что сделать |
|---|---|---|
| Soft delete заметок inbox с решением (`dropped`, `→ step`, `merged`, `expired`…), удалённое скрыто от выдачи | Только жёсткий `delete_progress_point` | Реализовать soft delete: `deleted_at` + `resolution` (+ ссылка на ID), фильтр в `get_activity_stats` и `search_progress_notes` |
| Счётчик всплываний `+1` | Нет поля | Пока — в тексте заметки; решить, нужно ли поле |
| Служебная activity «Inbox» | Не создана | Создать при миграции; выбрать `progress_type` и life part |
| Лимит 6 goal, WIP 6 / 3 на тип | Не проверяется | Считает агент на ревью; возможно — проверка в `create_*` |
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
| Чекин | `create_progress_point(activity_id, value, note?, progress_at?, hours_left?)` |
| Поправить чекин | `edit_progress_point(progress_id, …)` |
| Удалить чекин | `delete_progress_point(progress_id)` |
| Поиск по заметкам | `search_progress_notes(query_variants, …)` |
| Метафоры шкалы | `get_progress_type_examples` |
| Steps: список / создать / изменить / удалить | `get_step_list` / `create_step` / `edit_step` / `delete_step` |
| Goals: список / создать / изменить / +delta / пересчитать | `get_goal_progress` / `create_goal` / `update_goal` / `log_goal_progress` / `refresh_goals` |
| Life parts | `list_life_parts` |