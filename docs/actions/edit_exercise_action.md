# Edit Exercise Action

## Requirements

### User Story

Exercises occasionally get wrong names or equipment types (e.g. "Behind the Neck Press" was mapped to "Overhead Cable Tricep Extension"). There is no way to fix this without manually touching the database. `edit_exercise` allows updating the name and/or equipment type of an existing exercise.

### MCP Tool

**edit_exercise** — update name and/or equipment type of an existing exercise

### Input

- `exercise_id` (int, required)
- `name` (string, optional) — new name
- `equipment_type` (string, optional) — one of: `machine`, `barbell`, `dumbbells`, `bodyweight`

At least one of `name` or `equipment_type` must be provided.

### Output

Updated exercise object:
- `id`, `user_id`, `name`, `equipment_type`, `created_at`, `last_used_at`

## E2E Tests

### Test: Successfully updates name

```go
// Create exercise "Old Name" with equipment_type="barbell"
// Call edit_exercise with exercise_id, name="New Name"
// Verify output has name="New Name", equipment_type="barbell" (unchanged)
// Fetch exercise via search_exercises to confirm persisted
```

### Test: Successfully updates equipment_type

```go
// Create exercise "Leg Press" with equipment_type="barbell"
// Call edit_exercise with exercise_id, equipment_type="machine"
// Verify output has equipment_type="machine", name unchanged
```

### Test: Successfully updates both fields

```go
// Create exercise "Wrong Name" with equipment_type="barbell"
// Call edit_exercise with exercise_id, name="Correct Name", equipment_type="dumbbells"
// Verify both fields updated in output
```

### Test: Returns error when exercise not found

```go
// Call edit_exercise with non-existent exercise_id
// Verify error returned
```

### Test: Returns error when neither name nor equipment_type provided

```go
// Create exercise
// Call edit_exercise with only exercise_id, no name or equipment_type
// Verify validation error returned
```

### Test: Returns error for invalid equipment_type

```go
// Create exercise
// Call edit_exercise with equipment_type="invalid"
// Verify validation error returned
```

## Implementation

### Domain structure

Reuse existing `domain.Exercise` (no changes needed).

### Database

```go
// gateways/interfaces.go — add to DB interface
UpdateExercise(ctx context.Context, exercise *domain.Exercise) error
GetExercise(ctx context.Context, exerciseID int64, userID int64) (*domain.Exercise, error)
```

SQL for update: `UPDATE exercises SET name = $1, equipment_type = $2 WHERE id = $3 AND user_id = $4`

SQL for get: `SELECT id, user_id, name, equipment_type, created_at FROM exercises WHERE id = $1 AND user_id = $2`

(`last_used_at` computed via MAX(sets.created_at) JOIN — same as ListExercises)

### MCP Tool

#### edit_exercise

**Input:**
```json
{ "exercise_id": 5, "name": "Correct Name", "equipment_type": "barbell" }
```

**Output:**
```json
{ "id": 5, "user_id": 1, "name": "Correct Name", "equipment_type": "barbell", "created_at": "...", "last_used_at": null }
```

**Logic:**
- Validate: at least one of name/equipment_type provided; equipment_type valid if provided
- Call `DB.GetExercise(exerciseID, userID)` — returns 404-style error if not found
- Apply updates: use existing value for any field not provided
- Call `DB.UpdateExercise(exercise)`
- Re-fetch via `DB.GetExerciseWithLastUsed(exerciseID, userID)` and return
