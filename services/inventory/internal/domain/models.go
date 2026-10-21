package domain

import (
	"time"

	"github.com/google/uuid"
)

type UnitStatus string

const (
	StatusAvailable   UnitStatus = "available"
	StatusReserved    UnitStatus = "reserved"
	StatusBorrowed    UnitStatus = "borrowed"
	StatusMaintenance UnitStatus = "maintenance"
	StatusRetired     UnitStatus = "retired"
)

func (s UnitStatus) IsValid() bool {
	switch s {
	case StatusAvailable, StatusReserved, StatusBorrowed, StatusMaintenance, StatusRetired:
		return true
	default:
		return false
	}
}

type UnitCondition string

const (
	ConditionNew     UnitCondition = "new"
	ConditionGood    UnitCondition = "good"
	ConditionFair    UnitCondition = "fair"
	ConditionPoor    UnitCondition = "poor"
	ConditionDamaged UnitCondition = "damaged"
)

func (c UnitCondition) IsValid() bool {
	switch c {
	case ConditionNew, ConditionGood, ConditionFair, ConditionPoor, ConditionDamaged:
		return true
	default:
		return false
	}
}

type Category struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	Description    *string   `json:"description,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Item struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	CategoryID     uuid.UUID `json:"category_id"`
	Name           string    `json:"name"`
	Description    *string   `json:"description,omitempty"`
	IsLendable     bool      `json:"is_lendable"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Unit struct {
	ID             uuid.UUID     `json:"id"`
	OrganizationID uuid.UUID     `json:"organization_id"`
	ItemID         uuid.UUID     `json:"item_id"`
	Barcode        string        `json:"barcode"`
	SerialNumber   *string       `json:"serial_number,omitempty"`
	Status         UnitStatus    `json:"status"`
	Condition      UnitCondition `json:"condition"`
	Notes          *string       `json:"notes,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

func (u *Unit) CanLend(item *Item) bool {
	if item != nil && !item.IsLendable {
		return false
	}
	return u.Status == StatusAvailable
}

func (u *Unit) IsActive() bool {
	return u.Status != StatusRetired
}

type ConditionLog struct {
	ID                uuid.UUID      `json:"id"`
	UnitID            uuid.UUID      `json:"unit_id"`
	PreviousCondition *UnitCondition `json:"previous_condition,omitempty"`
	NewCondition      UnitCondition  `json:"new_condition"`
	Reason            *string        `json:"reason,omitempty"`
	ChangedBy         uuid.UUID      `json:"changed_by"`
	CreatedAt         time.Time      `json:"created_at"`
}
