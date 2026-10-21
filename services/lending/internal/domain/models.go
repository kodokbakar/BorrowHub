package domain

import (
	"time"

	"github.com/google/uuid"
)

type RequestStatus string

const (
	RequestStatusPending   RequestStatus = "pending"
	RequestStatusApproved  RequestStatus = "approved"
	RequestStatusRejected  RequestStatus = "rejected"
	RequestStatusCancelled RequestStatus = "cancelled"
)

type LoanStatus string

const (
	LoanStatusPendingCheckout LoanStatus = "pending_checkout"
	LoanStatusCheckedOut      LoanStatus = "checked_out"
	LoanStatusReturned        LoanStatus = "returned"
	LoanStatusOverdue         LoanStatus = "overdue"
	LoanStatusCancelled       LoanStatus = "cancelled"
)

type ReservationStatus string

const (
	ReservationStatusActive    ReservationStatus = "active"
	ReservationStatusFulfilled ReservationStatus = "fulfilled"
	ReservationStatusCancelled ReservationStatus = "cancelled"
)

type ReturnCondition string

const (
	ReturnConditionNew     ReturnCondition = "new"
	ReturnConditionGood    ReturnCondition = "good"
	ReturnConditionFair    ReturnCondition = "fair"
	ReturnConditionPoor    ReturnCondition = "poor"
	ReturnConditionDamaged ReturnCondition = "damaged"
)

func (c ReturnCondition) IsValid() bool {
	switch c {
	case ReturnConditionNew, ReturnConditionGood, ReturnConditionFair, ReturnConditionPoor, ReturnConditionDamaged:
		return true
	default:
		return false
	}
}

func (c ReturnCondition) RequiresMaintenance() bool {
	return c == ReturnConditionDamaged
}

type LoanRequest struct {
	ID              uuid.UUID     `json:"id"`
	OrganizationID  uuid.UUID     `json:"organization_id"`
	RequesterUserID uuid.UUID     `json:"requester_user_id"`
	ItemID          uuid.UUID     `json:"item_id"`
	Quantity        int           `json:"quantity"`
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
	Purpose         string        `json:"purpose"`
	Notes           *string       `json:"notes,omitempty"`
	Status          RequestStatus `json:"status"`
	ReviewerUserID  *uuid.UUID    `json:"reviewer_user_id,omitempty"`
	ReviewNotes     *string       `json:"review_notes,omitempty"`
	ReviewedAt      *time.Time    `json:"reviewed_at,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type Reservation struct {
	ID             uuid.UUID         `json:"id"`
	OrganizationID uuid.UUID         `json:"organization_id"`
	LoanRequestID  uuid.UUID         `json:"loan_request_id"`
	UnitID         uuid.UUID         `json:"unit_id"`
	StartTime      time.Time         `json:"start_time"`
	EndTime        time.Time         `json:"end_time"`
	Status         ReservationStatus `json:"status"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type Loan struct {
	ID                 uuid.UUID        `json:"id"`
	OrganizationID     uuid.UUID        `json:"organization_id"`
	LoanRequestID      uuid.UUID        `json:"loan_request_id"`
	ReservationID      uuid.UUID        `json:"reservation_id"`
	UnitID             uuid.UUID        `json:"unit_id"`
	BorrowerUserID     uuid.UUID        `json:"borrower_user_id"`
	Status             LoanStatus       `json:"status"`
	CheckoutTime       *time.Time       `json:"checkout_time,omitempty"`
	ExpectedReturnTime time.Time        `json:"expected_return_time"`
	ActualReturnTime   *time.Time       `json:"actual_return_time,omitempty"`
	ReturnCondition    *ReturnCondition `json:"return_condition,omitempty"`
	ReturnNotes        *string          `json:"return_notes,omitempty"`
	ProcessedByUserID  *uuid.UUID       `json:"processed_by_user_id,omitempty"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
}

type CheckoutToken struct {
	ID        uuid.UUID  `json:"id"`
	LoanID    uuid.UUID  `json:"loan_id"`
	TokenHash string     `json:"token_hash"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type StatusHistory struct {
	ID              uuid.UUID `json:"id"`
	EntityType      string    `json:"entity_type"`
	EntityID        uuid.UUID `json:"entity_id"`
	FromStatus      string    `json:"from_status"`
	ToStatus        string    `json:"to_status"`
	ChangedByUserID uuid.UUID `json:"changed_by_user_id"`
	Reason          *string   `json:"reason,omitempty"`
	CorrelationID   string    `json:"correlation_id"`
	CreatedAt       time.Time `json:"created_at"`
}

type OutboxEvent struct {
	ID             int64          `json:"id"`
	EventID        uuid.UUID      `json:"event_id"`
	EventType      string         `json:"event_type"`
	EventVersion   int            `json:"event_version"`
	Source         string         `json:"source"`
	Subject        string         `json:"subject"`
	OrganizationID uuid.UUID      `json:"organization_id"`
	OccurredAt     time.Time      `json:"occurred_at"`
	CorrelationID  string         `json:"correlation_id"`
	CausationID    string         `json:"causation_id"`
	Data           map[string]any `json:"data"`
	Status         string         `json:"status"`
	RetryCount     int            `json:"retry_count"`
	LastError      *string        `json:"last_error,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	PublishedAt    *time.Time     `json:"published_at,omitempty"`
}

type IdempotencyRecord struct {
	Key            string    `json:"key"`
	OrganizationID uuid.UUID `json:"organization_id"`
	UserID         uuid.UUID `json:"user_id"`
	RequestPath    string    `json:"request_path"`
	RequestHash    string    `json:"request_hash"`
	ResponseCode   int       `json:"response_code"`
	ResponseBody   []byte    `json:"response_body"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type ItemAvailability struct {
	ItemID         uuid.UUID   `json:"item_id"`
	StartTime      time.Time   `json:"start_time"`
	EndTime        time.Time   `json:"end_time"`
	TotalUnits     int         `json:"total_units"`
	AvailableUnits int         `json:"available_units"`
	IsAvailable    bool        `json:"is_available"`
	EligibleUnits  []uuid.UUID `json:"eligible_unit_ids,omitempty"`
}
