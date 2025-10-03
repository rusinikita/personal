# Workout Tracking System - Complete Specification

## Overview

System for tracking workout exercises, sets, and training history with MCP (Model Context Protocol) interface. Supports different equipment types (machine, barbell, dumbbells, bodyweight) and tracks both rep-based and time-based exercises.

## Best Practices Applied

- **Multi-user Support**: All tables have user_id for data isolation
- **Single Active Workout Pattern**: Only one workout per user can be active at a time (completed_at IS NULL)
- **Auto-workout Creation**: First log_set call automatically creates active workout if none exists
- **Flexible Metrics**: Support both reps (dynamic) and duration (static/isometric exercises)
- **Denormalized Reads**: Calculate last_used_at via JOIN instead of storing redundantly
- **Progress Tracking**: Weight and time metrics for monitoring improvements
- **Nullable Fields**: reps, duration_seconds, weight_kg are nullable but at least one must be set
- **User Context**: user_id extracted from authentication context (JWT/session), not passed explicitly

## Architecture Diagrams

### Entity Relation Diagram

```mermaid
erDiagram
    EXERCISES ||--o{ SETS : "used_in"
    WORKOUTS ||--o{ SETS : "contains"
    
    EXERCISES {
        int id PK
        int user_id FK
        string name
        string equipment_type "machine|barbell|dumbbells|bodyweight"
        timestamp created_at
    }
    
    WORKOUTS {
        int id PK
        int user_id FK
        timestamp started_at
        timestamp completed_at "NULL if active"
    }
    
    SETS {
        int id PK
        int user_id FK
        int workout_id FK
        int exercise_id FK
        int reps "NULL for static exercises"
        int duration_seconds "NULL for rep-based exercises"
        float weight_kg "NULL for bodyweight"
        timestamp created_at
    }
```

### C4 Context Diagram

```mermaid
graph TB
    User[User/Claude MCP Client]
    
    subgraph "Workout Tracking System"
        MCP[MCP Server]
        DB[(PostgreSQL Database)]
        
        MCP -->|SQL queries| DB
    end
    
    User -->|create_exercise| MCP
    User -->|list_exercises| MCP
    User -->|log_set| MCP
    User -->|list_workouts| MCP
    
    DB -.->|exercises table| DB
    DB -.->|workouts table| DB
    DB -.->|sets table| DB
    
    style User fill:#e1f5ff
    style MCP fill:#ffe1e1
    style DB fill:#e1ffe1
```

### Sequence Diagram: Log Set

```mermaid
sequenceDiagram
    participant User
    participant MCP
    participant Auth
    participant DB
    
    User->>MCP: log_set(exercise_id, reps, weight_kg)
    MCP->>Auth: Get user_id from context
    Auth-->>MCP: user_id
    
    MCP->>DB: SELECT id FROM workouts<br/>WHERE user_id = ? AND completed_at IS NULL<br/>LIMIT 1
    
    alt No active workout
        MCP->>DB: INSERT INTO workouts<br/>(user_id, started_at) VALUES (?, NOW())<br/>RETURNING id
        DB-->>MCP: workout_id
    else Active workout exists
        DB-->>MCP: workout_id
    end
    
    MCP->>DB: INSERT INTO sets<br/>(user_id, workout_id, exercise_id, reps, weight_kg, created_at)<br/>VALUES (...)
    
    DB-->>MCP: set_id
    MCP-->>User: Success: {set_id, workout_id}
```

### Sequence Diagram: List Exercises

```mermaid
sequenceDiagram
    participant User
    participant MCP
    participant Auth
    participant DB
    
    User->>MCP: list_exercises()
    MCP->>Auth: Get user_id from context
    Auth-->>MCP: user_id
    
    MCP->>DB: SELECT e.*, MAX(s.created_at) as last_used_at<br/>FROM exercises e<br/>LEFT JOIN sets s ON e.id = s.exercise_id AND s.user_id = ?<br/>WHERE e.user_id = ?<br/>GROUP BY e.id<br/>ORDER BY last_used_at DESC NULLS LAST, e.name<br/>LIMIT 20
    
    DB-->>MCP: List of exercises with last_used_at
    
    MCP-->>User: [{id, name, equipment_type, last_used_at}, ...]
```

### Sequence Diagram: List Workouts

```mermaid
sequenceDiagram
    participant User
    participant MCP
    participant Auth
    participant DB
    
    User->>MCP: list_workouts(limit=10)
    MCP->>Auth: Get user_id from context
    Auth-->>MCP: user_id
    
    MCP->>DB: SELECT w.*, <br/>COUNT(s.id) as total_sets<br/>FROM workouts w<br/>LEFT JOIN sets s ON w.id = s.workout_id<br/>WHERE w.user_id = ?<br/>GROUP BY w.id<br/>ORDER BY w.started_at DESC<br/>LIMIT 10
    
    DB-->>MCP: Workouts list
    
    loop For each workout
        MCP->>DB: SELECT s.*, e.name, e.equipment_type<br/>FROM sets s<br/>JOIN exercises e ON s.exercise_id = e.id<br/>WHERE s.workout_id = ? AND s.user_id = ?<br/>ORDER BY s.created_at
        DB-->>MCP: Sets with exercise details
    end
    
    MCP-->>User: [{workout, sets: [{set, exercise}, ...]}, ...]
```

## Database Schema

### SQL DDL

```sql
-- Exercises table
CREATE TABLE exercises (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    equipment_type VARCHAR(50) NOT NULL CHECK (equipment_type IN ('machine', 'barbell', 'dumbbells', 'bodyweight')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_exercises_name ON exercises(name);

-- Workouts table
CREATE TABLE workouts (
    id SERIAL PRIMARY KEY,
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP NULL,  -- NULL means active workout
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workouts_completed_at ON workouts(completed_at) WHERE completed_at IS NULL;
CREATE INDEX idx_workouts_started_at ON workouts(started_at DESC);

-- Sets table
CREATE TABLE sets (
    id SERIAL PRIMARY KEY,
    workout_id INTEGER NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
    exercise_id INTEGER NOT NULL REFERENCES exercises(id) ON DELETE RESTRICT,
    reps INTEGER NULL,  -- For dynamic exercises
    duration_seconds INTEGER NULL,  -- For static/isometric exercises
    weight_kg DECIMAL(6,2) NULL,  -- NULL for bodyweight exercises
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_set_metrics CHECK (
        reps IS NOT NULL OR duration_seconds IS NOT NULL
    )
);

CREATE INDEX idx_sets_workout_id ON sets(workout_id);
CREATE INDEX idx_sets_exercise_id ON sets(exercise_id);
CREATE INDEX idx_sets_created_at ON sets(created_at DESC);
```

## Go Code Structure

### Domain Models

```go
package workout

import (
	"time"
)

// EquipmentType represents the type of equipment used for an exercise
type EquipmentType string

const (
	EquipmentMachine    EquipmentType = "machine"
	EquipmentBarbell    EquipmentType = "barbell"
	EquipmentDumbbells  EquipmentType = "dumbbells"
	EquipmentBodyweight EquipmentType = "bodyweight"
)

// Exercise represents a workout exercise
type Exercise struct {
	ID            int64           `json:"id"`
	UserID        int64           `json:"user_id"`
	Name          string        `json:"name"`
	EquipmentType EquipmentType `json:"equipment_type"`
	CreatedAt     time.Time     `json:"created_at"`
	LastUsedAt    *time.Time    `json:"last_used_at,omitempty"` // Computed from sets
}

// Workout represents a training session
type Workout struct {
	ID          int64        `json:"id"`
	UserID      int64        `json:"user_id"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"` // NULL means active
}

// Set represents a single set within a workout
type Set struct {
	ID              int64       `json:"id"`
	UserID          int64       `json:"user_id"`
	WorkoutID       int64       `json:"workout_id"`
	ExerciseID      int64       `json:"exercise_id"`
	Reps            *int64      `json:"reps"`             // NULL for static exercises
	DurationSeconds *int64      `json:"duration_seconds"` // NULL for rep-based exercises
	WeightKg        *float64  `json:"weight_kg"`        // NULL for bodyweight
	CreatedAt       time.Time `json:"created_at"`
}

type WorkoutSet struct {
	Workout
	Set
}

type ExerciseSearch struct {
	UserID  int
	Limit   int 
}



```

## DB Repository interface

```go
// gateways/workout_repository.go
type WorkoutRepository interface {
	// Exercise operations
	CreateExercise(ctx context.Context, exercise Exercise) (int64, error)
	ListExercises(ctx context.Context, params ExerciseSearch) ([]Exercise, error)

	// Workout operations
	CreateWorkout(ctx context.Context, workout Workout) (int64, error)
	CloseWorkout(ctx context.Context, workoutID int64) error
	ListWorkouts(ctx context.Context, params ExerciseSearch) ([]Workout, error)

	// Set operations
	CreateSet(ctx context.Context, set *Set) error
	ListSets(ctx context.Context, workoutID int64) ([]Set, error)
	GetLastSet(ctx context.Context, userID int64) (WorkoutSet, error)
}
```