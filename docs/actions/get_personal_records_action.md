# Get Personal Records Action

## Requirements

### User Story

Currently the assistant pulls `list_workouts` with a high limit and manually scans for bests — slow and limited to ~30 days. `get_personal_records` returns pre-computed bests for a given exercise.

### MCP Tool

**get_personal_records** — return best results for an exercise across all time

### Input

- `exercise_id` (int, required)

### Output

- `max_weight` — heaviest single set: `{ weight_kg, reps, date }`
- `max_reps` — most reps in a single set: `{ weight_kg, reps, date }`
- `max_volume` — highest total volume in one workout (sum of weight×reps): `{ volume, date }`
- `estimated_1rm` — Epley formula from max_weight set: `weight × (1 + reps/30)`

All fields are nullable (null if no sets exist). Only sets with `reps > 0` and `weight_kg > 0` count toward weight/volume metrics.

## E2E Tests

### Test: Returns correct records across multiple workouts

```go
// Create exercise
// Workout 1: sets [5×80, 5×80, 5×80] → volume=1200
// Workout 2: sets [3×100, 8×80] → max_weight=100, max_reps=8, volume=940
// Verify max_weight={100, 3}, max_reps={80, 8}, max_volume={1200, workout1 date}
// Verify estimated_1rm = 100 * (1 + 3/30) = 110.0
```

### Test: Returns nil fields for exercise with no sets

```go
// Create exercise, no sets logged
// Verify max_weight=nil, max_reps=nil, max_volume=nil, estimated_1rm=0
```

### Test: Estimated 1RM uses Epley formula

```go
// Create exercise, log one set: 5×100kg
// Verify estimated_1rm = 100 * (1 + 5/30) ≈ 116.67
```

## Implementation

### Domain structure

```go
// domain/set.go — add
type SetRecord struct {
    WeightKg  float64
    Reps      int64
    CreatedAt time.Time
}

type VolumeRecord struct {
    Volume    float64
    StartedAt time.Time
}

type PersonalRecords struct {
    MaxWeight *SetRecord
    MaxReps   *SetRecord
    MaxVolume *VolumeRecord
}
```

### Database

```go
// gateways/interfaces.go — add
GetPersonalRecords(ctx context.Context, userID int64, exerciseID int64) (*domain.PersonalRecords, error)
```

Three SQL queries:
1. `SELECT weight_kg, reps, created_at FROM sets WHERE exercise_id=$1 AND user_id=$2 AND reps>0 AND weight_kg>0 ORDER BY weight_kg DESC, reps DESC LIMIT 1`
2. `SELECT weight_kg, reps, created_at FROM sets WHERE exercise_id=$1 AND user_id=$2 AND reps>0 ORDER BY reps DESC, weight_kg DESC LIMIT 1`
3. `SELECT SUM(s.weight_kg*s.reps) as vol, w.started_at FROM sets s JOIN workouts w ON s.workout_id=w.id WHERE s.exercise_id=$1 AND s.user_id=$2 AND s.reps>0 AND s.weight_kg>0 GROUP BY s.workout_id, w.started_at ORDER BY vol DESC LIMIT 1`

### MCP Tool

**Logic:**
- Validate exercise_id non-zero
- Call `DB.GetPersonalRecords(userID, exerciseID)`
- Compute `estimated_1rm = weight_kg * (1 + float64(reps)/30)` from MaxWeight set (if non-nil)
- Return formatted output
