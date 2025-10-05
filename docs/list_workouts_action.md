# List Workouts Action

## Requirements

### Domain Entities

**Workout (Тренировка)**
- ID
- User ID (владелец тренировки)
- Started at timestamp
- Completed at timestamp (NULL если тренировка активна)
- Total sets count (вычисляется через COUNT)

**Set (Подход)**
- ID
- User ID
- Workout ID
- Exercise ID
- Reps (количество повторений)
- Duration seconds (длительность в секундах)
- Weight kg (вес в кг)
- Created at timestamp

**Exercise (Упражнение)**
- ID
- Name (название)
- Equipment Type (тип оборудования)

### MCP Tool

**list_workouts** - вывод последних N тренировок с их подходами

### User Story

Пользователь запрашивает список своих тренировок. Получает список тренировок (последние первыми) с информацией о каждом подходе и упражнении в тренировке.

## E2E Tests

### Test: List workouts with sets
```go
// Arrange
// Create 2 exercises via WorkoutRepository.CreateExercise
// Create workout 1 via WorkoutRepository.CreateWorkout with started_at=now-2h, completed_at=now-1h
// Create 3 sets for workout 1, exercise 1 via WorkoutRepository.CreateSet
// Create workout 2 via WorkoutRepository.CreateWorkout with started_at=now-1h, completed_at=NULL (active)
// Create 2 sets for workout 2, exercise 2 via WorkoutRepository.CreateSet

// Act
// Call MCP tool list_workouts

// Assert
// Verify 2 workouts returned, sorted by started_at DESC (workout 2 first, workout 1 second)
// Verify workout 1: 3 sets total, all sets have exercise 1 details (name, equipment_type)
// Verify workout 2: 2 sets total, all sets have exercise 2 details (name, equipment_type), completed_at=null
```

## Implementation

### Domain structure

All domain structures already exist in the codebase:

```go
// domain/workout.go - ALREADY EXISTS
type Workout struct {
    ID          int64      `json:"id"`
    UserID      int64      `json:"user_id"`
    StartedAt   time.Time  `json:"started_at"`
    CompletedAt *time.Time `json:"completed_at"` // NULL means active
}

// domain/set.go - ALREADY EXISTS
type Set struct {
    ID              int64     `json:"id"`
    UserID          int64     `json:"user_id"`
    WorkoutID       int64     `json:"workout_id"`
    ExerciseID      int64     `json:"exercise_id"`
    Reps            int64     `json:"reps,omitempty"`             // 0 for static exercises
    DurationSeconds int64     `json:"duration_seconds,omitempty"` // 0 for rep-based exercises
    WeightKg        float64   `json:"weight_kg,omitempty"`        // 0 for bodyweight
    CreatedAt       time.Time `json:"created_at"`
}

// domain/exercise.go - ALREADY EXISTS
type Exercise struct {
    ID            int64         `json:"id"`
    UserID        int64         `json:"user_id"`
    Name          string        `json:"name"`
    EquipmentType EquipmentType `json:"equipment_type"`
    CreatedAt     time.Time     `json:"created_at"`
    LastUsedAt    *time.Time    `json:"last_used_at,omitempty"`
}
```

### Database

```go
// gateways/interfaces.go - ADD new methods to DB interface
ListSets(ctx context.Context, userID int64, from time.Time, to time.Time) ([]domain.Set, error)
GetExercisesByIDs(ctx context.Context, userID int64, exerciseIDs []int64) ([]domain.Exercise, error)
GetWorkoutsByIDs(ctx context.Context, userID int64, workoutIDs []int64) ([]domain.Workout, error)
```

```mermaid
erDiagram
    exercises {
        serial id PK
        int user_id FK
        varchar name
        varchar equipment_type
        timestamp created_at
    }

    workouts {
        serial id PK
        int user_id FK
        timestamp started_at
        timestamp completed_at
    }

    sets {
        serial id PK
        int user_id FK
        int workout_id FK
        int exercise_id FK
        int reps
        int duration_seconds
        decimal weight_kg
        timestamp created_at
    }

    workouts ||--o{ sets : "contains"
    exercises ||--o{ sets : "used_in"
```

### MCP Tool

#### list_workouts

**Input:**
```go
{
    "limit": int  // optional, default 10
}
```

**Output:**
```go
{
    "workouts": [
        {
            "id": int64,
            "user_id": int64,
            "started_at": string (ISO8601),
            "completed_at": string (ISO8601) | null,
            "exercises": [
                {
                    "exercise_id": int64,
                    "exercise_name": string,
                    "equipment_type": string,
                    "sets": [
                        {
                            "id": int64,
                            "reps": int | null,
                            "duration_seconds": int | null,
                            "weight_kg": float | null,
                            "created_at": string (ISO8601)
                        }
                    ]
                }
            ]
        }
    ]
}
```

**Logic:**
- Use default user_id from context
- Call DB.ListSets(user_id, from=NOW()-30days, to=NOW()) - returns all sets sorted by created_at DESC
- Extract unique workout_ids and exercise_ids from sets
- Call DB.GetWorkoutsByIDs(user_id, workout_ids) to get workout details
- Call DB.GetExercisesByIDs(user_id, exercise_ids) to get exercise details
- Group sets by workout_id, then by exercise_id
- Merge workout, exercise, and set data
- Sort workouts by started_at DESC, limit to first N workouts (default 10)
- Return workouts with grouped exercises and sets as JSON
