# Search Exercises Action

## Requirements

### User Story

`list_exercises` returns only the 20 most recently used exercises. Rarely-used exercises become invisible, causing the assistant to create duplicates. `search_exercises` searches ALL exercises by name variants (same pattern as `resolve_food_id_by_name`), returning results ranked by how many variants matched.

### MCP Tool

**search_exercises** — search exercises by 1-5 name variants, case-insensitive

### Input

- `name_variants` ([]string, required) — 1 to 5 partial name strings, case-insensitive

### Output

- `exercises` — array of matches, each with: `exercise_id`, `name`, `equipment_type`, `last_used_at` (nullable ISO8601), `match_count`
- `error` (string, omitempty) — validation error if any

Results sorted by `match_count` DESC, then `exercise_id` ASC for stability.

## E2E Tests

### Test: Returns matching exercises for partial name

```go
// Create exercises: "Bench Press", "Incline Bench Press", "Squat"
// Call search_exercises with name_variants=["bench"]
// Verify 2 results returned: "Bench Press" and "Incline Bench Press"
// Verify "Squat" not in results
```

### Test: Multiple variants increase match_count

```go
// Create exercise "Overhead Press"
// Call search_exercises with name_variants=["overhead", "press"]
// Verify exercise returned with match_count=2
```

### Test: Case-insensitive search

```go
// Create exercise "Deadlift"
// Call search_exercises with name_variants=["DEADLIFT"]
// Verify exercise returned
```

### Test: Returns empty array when no matches

```go
// Call search_exercises with name_variants=["zzznomatch"]
// Verify empty exercises array (no error)
```

### Test: Returns last_used_at when exercise has been used

```go
// Create exercise "Pull-up"
// Log a set for it
// Call search_exercises with name_variants=["pull"]
// Verify result has non-null last_used_at
```

### Test: Validation error for empty name_variants

```go
// Call search_exercises with name_variants=[]
// Verify error returned in response
```

## Implementation

### Domain structure

Reuse existing `domain.Exercise` (no changes needed).

### Database

```go
// gateways/interfaces.go — add to DB interface
SearchExercises(ctx context.Context, userID int64, query string) ([]domain.Exercise, error)
```

SQL: `SELECT exercises.*, MAX(sets.created_at) as last_used_at FROM exercises LEFT JOIN sets ON sets.exercise_id = exercises.id AND sets.user_id = exercises.user_id WHERE exercises.user_id = $1 AND exercises.name ILIKE $2 GROUP BY exercises.id ORDER BY last_used_at DESC NULLS LAST, exercises.name ASC`

### MCP Tool

#### search_exercises

**Input:**
```json
{ "name_variants": ["bench", "press"] }
```

**Output:**
```json
{
  "exercises": [
    { "exercise_id": 1, "name": "Bench Press", "equipment_type": "barbell", "last_used_at": "2026-03-10T10:00:00Z", "match_count": 2 },
    { "exercise_id": 2, "name": "Incline Bench Press", "equipment_type": "barbell", "last_used_at": null, "match_count": 1 }
  ]
}
```

**Logic:**
- Validate `name_variants` is non-empty (1-5 items, no blank strings)
- For each variant call `DB.SearchExercises(userID, variant)` → ILIKE `%variant%`
- Deduplicate by exercise ID, track `match_count` per exercise
- Sort by `match_count` DESC, `exercise_id` ASC
- Return matches
