package service

import (
	"context"
	"time"

	"borrowhub/services/lending/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type LendingRepository interface {
	RunInTx(ctx context.Context, fn func(tx pgx.Tx) error) error

	CreateLoanRequest(ctx context.Context, tx pgx.Tx, req *domain.LoanRequest) error
	GetLoanRequest(ctx context.Context, id uuid.UUID) (*domain.LoanRequest, error)
	GetLoanRequestByID(ctx context.Context, id uuid.UUID) (*domain.LoanRequest, error)
	ListLoanRequests(ctx context.Context, orgID uuid.UUID, requesterID *uuid.UUID, status *domain.RequestStatus, limit, offset int) ([]domain.LoanRequest, error)
	UpdateLoanRequest(ctx context.Context, tx pgx.Tx, req *domain.LoanRequest) error

	CreateReservation(ctx context.Context, tx pgx.Tx, res *domain.Reservation) error
	GetReservation(ctx context.Context, id uuid.UUID) (*domain.Reservation, error)
	GetReservationByLoanRequestID(ctx context.Context, loanRequestID uuid.UUID) (*domain.Reservation, error)
	UpdateReservationStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status domain.ReservationStatus) error
	FindOverlappingUnitIDs(ctx context.Context, unitIDs []uuid.UUID, startTime, endTime time.Time) ([]uuid.UUID, error)

	CreateLoan(ctx context.Context, tx pgx.Tx, loan *domain.Loan) error
	GetLoan(ctx context.Context, id uuid.UUID) (*domain.Loan, error)
	GetLoanByID(ctx context.Context, id uuid.UUID) (*domain.Loan, error)
	GetLoanByLoanRequestID(ctx context.Context, loanRequestID uuid.UUID) (*domain.Loan, error)
	ListLoans(ctx context.Context, orgID uuid.UUID, borrowerUserID *uuid.UUID, status *domain.LoanStatus, limit, offset int) ([]domain.Loan, error)
	UpdateLoan(ctx context.Context, tx pgx.Tx, loan *domain.Loan) error
	ListOverdueLoans(ctx context.Context, orgID uuid.UUID) ([]domain.Loan, error)
	FindOverdueCheckedOutLoans(ctx context.Context, now time.Time) ([]domain.Loan, error)
	TransitionLoanToOverdue(ctx context.Context, loanID uuid.UUID, now time.Time) (bool, error)
	CreateCheckoutToken(ctx context.Context, tx pgx.Tx, token *domain.CheckoutToken) error
	GetCheckoutTokenByHash(ctx context.Context, tokenHash string) (*domain.CheckoutToken, error)
	MarkCheckoutTokenUsed(ctx context.Context, tx pgx.Tx, id uuid.UUID, usedAt time.Time) error

	CreateStatusHistory(ctx context.Context, tx pgx.Tx, history *domain.StatusHistory) error
	GetStatusHistory(ctx context.Context, entityType string, entityID uuid.UUID) ([]domain.StatusHistory, error)

	CreateOutboxEvent(ctx context.Context, tx pgx.Tx, event *domain.OutboxEvent) error
	ClaimPendingOutboxEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error)
	GetPendingOutboxEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error)
	MarkOutboxEventPublished(ctx context.Context, id int64) error
	MarkOutboxEventFailed(ctx context.Context, id int64, errStr string) error

	GetIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*domain.IdempotencyRecord, error)
	SaveIdempotencyKey(ctx context.Context, record *domain.IdempotencyRecord) error
}
