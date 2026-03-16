# Merge Exercises Action

## Requirements

### User Story

Duplicate exercises exist because `list_exercises` didn't show rarely-used ones, causing the assistant to create new ones. `merge_exercises` moves all sets from a source exercise into a target exercise, then deletes the source. No historical data is lost.

### MCP Tool

**merge_exercises** — move all sets from source exercise to target, delete source

### Input

- `source_exercise_id` (int, required) — exercise to merge FROM (will be deleted)
- `target_exercise_id` (int, required) — exercise to merge INTO (kept)

### Output

- `sets_moved` (int) — number of sets reassigned
- `deleted_exercise_name` (string) — name of the deleted source exercise

## E2E Tests

### Test: Merges sets and deletes source

```go
// Create source exercise "Bench Press Duplicate" and target "Bench Press"
// Create workout, log 3 sets for source exercise
// Call merge_exercises with source_id, target_id
// Verify sets_moved=3, deleted_exercise_name="Bench Press Duplicate"
// Verify source exercise no longer findable via search
// Verify sets now belong to target exercise via get_exercise_history
```

### Test: Works with zero sets on source

```go
// Create source (no sets), target
// Call merge_exercises
// Verify sets_moved=0, source deleted
```

### Test: Error when source not found

```go
// Call merge_exercises with non-existent source_exercise_id
// Verify error returned
```

### Test: Error when same IDs provided

```go
// Create exercise
// Call merge_exercises with source_id == target_id
// Verify validation error returned
```

## Implementation

### Database

```go
// gateways/interfaces.go — add to DB interface
MoveSetsBetweenExercises(ctx context.Context, sourceID, targetID, userID int64) (int64, error)
DeleteExercise(ctx context.Context, exerciseID int64, userID int64) error
```

SQL move: `UPDATE sets SET exercise_id = $2 WHERE exercise_id = $1 AND user_id = $3` → returns RowsAffected

SQL delete: `DELETE FROM exercises WHERE id = $1 AND user_id = $2`

### MCP Tool

**Logic:**
- Validate source_id != target_id, both non-zero
- Fetch source exercise (verify exists, capture name)
- Fetch target exercise (verify exists)
- Call `MoveSetsBetweenExercises(sourceID, targetID, userID)` → count
- Call `DeleteExercise(sourceID, userID)`
- Return `sets_moved`, `deleted_exercise_name`
