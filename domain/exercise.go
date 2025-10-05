package domain

import "time"

type Exercise struct {
	ID            int64         `json:"id"`
	UserID        int64         `json:"user_id"`
	Name          string        `json:"name"`
	EquipmentType EquipmentType `json:"equipment_type"`
	CreatedAt     time.Time     `json:"created_at"`
	LastUsedAt    *time.Time    `json:"last_used_at,omitempty"` // Computed from sets
}

type EquipmentType string

const (
	EquipmentMachine    EquipmentType = "machine"
	EquipmentBarbell    EquipmentType = "barbell"
	EquipmentDumbbells  EquipmentType = "dumbbells"
	EquipmentBodyweight EquipmentType = "bodyweight"
)

// IsValid checks if the equipment type is valid
func (e EquipmentType) IsValid() bool {
	switch e {
	case EquipmentMachine, EquipmentBarbell, EquipmentDumbbells, EquipmentBodyweight:
		return true
	default:
		return false
	}
}
