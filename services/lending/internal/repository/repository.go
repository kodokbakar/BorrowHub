package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"borrowhub/services/lending/internal/domain"
	"borrowhub/services/lending/internal/event"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Pool() *pgxpool.Pool {
	return r.pool
}

func (r *Repository) RunInTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Loan Requests
// ---------------------------------------------------------------------------

func (r *Repository) CreateLoanRequest(ctx context.Context, tx pgx.Tx, req *domain.LoanRequest) error {
	query := `
		INSERT INTO loan_requests (
			id, organization_id, requester_user_id, item_id, quantity,
			start_time, end_time, purpose, notes, status,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query,
			req.ID, req.OrganizationID, req.RequesterUserID, req.ItemID, req.Quantity,
			req.StartTime, req.EndTime, req.Purpose, req.Notes, string(req.Status),
			req.CreatedAt, req.UpdatedAt,
		)
	} else {
		_, err = r.pool.Exec(ctx, query,
			req.ID, req.OrganizationID, req.RequesterUserID, req.ItemID, req.Quantity,
			req.StartTime, req.EndTime, req.Purpose, req.Notes, string(req.Status),
			req.CreatedAt, req.UpdatedAt,
		)
	}
	return mapPgError(err)
}

func (r *Repository) GetLoanRequest(ctx context.Context, id uuid.UUID) (*domain.LoanRequest, error) {
	query := `
		SELECT
			id, organization_id, requester_user_id, item_id, quantity,
			start_time, end_time, purpose, notes, status,
			reviewer_user_id, review_notes, reviewed_at, created_at, updated_at
		FROM loan_requests
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)
	var req domain.LoanRequest
	var statusStr string
	err := row.Scan(
		&req.ID, &req.OrganizationID, &req.RequesterUserID, &req.ItemID, &req.Quantity,
		&req.StartTime, &req.EndTime, &req.Purpose, &req.Notes, &statusStr,
		&req.ReviewerUserID, &req.ReviewNotes, &req.ReviewedAt, &req.CreatedAt, &req.UpdatedAt,
	)
	if err != nil {
		return nil, mapPgError(err)
	}
	req.Status = domain.RequestStatus(statusStr)
	return &req, nil
}
func (r *Repository) GetLoanRequestByID(ctx context.Context, id uuid.UUID) (*domain.LoanRequest, error) {
	return r.GetLoanRequest(ctx, id)
}

func (r *Repository) ListLoanRequests(ctx context.Context, orgID uuid.UUID, requesterID *uuid.UUID, status *domain.RequestStatus, limit, offset int) ([]domain.LoanRequest, error) {
	query := `
		SELECT
			id, organization_id, requester_user_id, item_id, quantity,
			start_time, end_time, purpose, notes, status,
			reviewer_user_id, review_notes, reviewed_at, created_at, updated_at
		FROM loan_requests
		WHERE organization_id = $1
		  AND ($2::uuid IS NULL OR requester_user_id = $2)
		  AND ($3::varchar IS NULL OR status = $3)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`
	var statusVal *string
	if status != nil {
		s := string(*status)
		statusVal = &s
	}
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.pool.Query(ctx, query, orgID, requesterID, statusVal, limit, offset)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer rows.Close()

	var results []domain.LoanRequest
	for rows.Next() {
		var req domain.LoanRequest
		var statusStr string
		if err := rows.Scan(
			&req.ID, &req.OrganizationID, &req.RequesterUserID, &req.ItemID, &req.Quantity,
			&req.StartTime, &req.EndTime, &req.Purpose, &req.Notes, &statusStr,
			&req.ReviewerUserID, &req.ReviewNotes, &req.ReviewedAt, &req.CreatedAt, &req.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning loan request row: %w", err)
		}
		req.Status = domain.RequestStatus(statusStr)
		results = append(results, req)
	}
	return results, nil
}

func (r *Repository) UpdateLoanRequest(ctx context.Context, tx pgx.Tx, req *domain.LoanRequest) error {
	query := `
		UPDATE loan_requests
		SET status = $1,
		    reviewer_user_id = $2,
		    review_notes = $3,
		    reviewed_at = $4,
		    updated_at = $5
		WHERE id = $6
	`
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query,
			string(req.Status), req.ReviewerUserID, req.ReviewNotes, req.ReviewedAt, req.UpdatedAt, req.ID,
		)
	} else {
		_, err = r.pool.Exec(ctx, query,
			string(req.Status), req.ReviewerUserID, req.ReviewNotes, req.ReviewedAt, req.UpdatedAt, req.ID,
		)
	}
	return mapPgError(err)
}

// ---------------------------------------------------------------------------
// Reservations
// ---------------------------------------------------------------------------

func (r *Repository) CreateReservation(ctx context.Context, tx pgx.Tx, res *domain.Reservation) error {
	query := `
		INSERT INTO reservations (
			id, organization_id, loan_request_id, unit_id,
			start_time, end_time, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query,
			res.ID, res.OrganizationID, res.LoanRequestID, res.UnitID,
			res.StartTime, res.EndTime, string(res.Status), res.CreatedAt, res.UpdatedAt,
		)
	} else {
		_, err = r.pool.Exec(ctx, query,
			res.ID, res.OrganizationID, res.LoanRequestID, res.UnitID,
			res.StartTime, res.EndTime, string(res.Status), res.CreatedAt, res.UpdatedAt,
		)
	}
	return mapPgError(err)
}

func (r *Repository) GetReservation(ctx context.Context, id uuid.UUID) (*domain.Reservation, error) {
	query := `
		SELECT id, organization_id, loan_request_id, unit_id, start_time, end_time, status, created_at, updated_at
		FROM reservations
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)
	var res domain.Reservation
	var statusStr string
	err := row.Scan(
		&res.ID, &res.OrganizationID, &res.LoanRequestID, &res.UnitID,
		&res.StartTime, &res.EndTime, &statusStr, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, mapPgError(err)
	}
	res.Status = domain.ReservationStatus(statusStr)
	return &res, nil
}

func (r *Repository) GetReservationByLoanRequestID(ctx context.Context, loanRequestID uuid.UUID) (*domain.Reservation, error) {
	query := `
		SELECT id, organization_id, loan_request_id, unit_id, start_time, end_time, status, created_at, updated_at
		FROM reservations
		WHERE loan_request_id = $1
	`
	row := r.pool.QueryRow(ctx, query, loanRequestID)
	var res domain.Reservation
	var statusStr string
	err := row.Scan(
		&res.ID, &res.OrganizationID, &res.LoanRequestID, &res.UnitID,
		&res.StartTime, &res.EndTime, &statusStr, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, mapPgError(err)
	}
	res.Status = domain.ReservationStatus(statusStr)
	return &res, nil
}

func (r *Repository) UpdateReservationStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status domain.ReservationStatus) error {
	query := `
		UPDATE reservations
		SET status = $1, updated_at = $2
		WHERE id = $3
	`
	now := time.Now()
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query, string(status), now, id)
	} else {
		_, err = r.pool.Exec(ctx, query, string(status), now, id)
	}
	return mapPgError(err)
}

func (r *Repository) FindOverlappingUnitIDs(ctx context.Context, unitIDs []uuid.UUID, startTime, endTime time.Time) ([]uuid.UUID, error) {
	if len(unitIDs) == 0 {
		return nil, nil
	}
	query := `
		SELECT DISTINCT unit_id
		FROM reservations
		WHERE unit_id = ANY($1)
		  AND status = 'active'
		  AND tstzrange(start_time, end_time, '[)') && tstzrange($2, $3, '[)')
	`
	rows, err := r.pool.Query(ctx, query, unitIDs, startTime, endTime)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer rows.Close()

	var overlapping []uuid.UUID
	for rows.Next() {
		var uid uuid.UUID
		if err := rows.Scan(&uid); err != nil {
			return nil, fmt.Errorf("scanning overlapping unit: %w", err)
		}
		overlapping = append(overlapping, uid)
	}
	return overlapping, nil
}

// ---------------------------------------------------------------------------
// Loans
// ---------------------------------------------------------------------------

func (r *Repository) CreateLoan(ctx context.Context, tx pgx.Tx, loan *domain.Loan) error {
	query := `
		INSERT INTO loans (
			id, organization_id, loan_request_id, reservation_id, unit_id, borrower_user_id,
			status, checkout_time, expected_return_time, actual_return_time,
			return_condition, return_notes, processed_by_user_id,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	var condVal *string
	if loan.ReturnCondition != nil {
		c := string(*loan.ReturnCondition)
		condVal = &c
	}

	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query,
			loan.ID, loan.OrganizationID, loan.LoanRequestID, loan.ReservationID, loan.UnitID, loan.BorrowerUserID,
			string(loan.Status), loan.CheckoutTime, loan.ExpectedReturnTime, loan.ActualReturnTime,
			condVal, loan.ReturnNotes, loan.ProcessedByUserID,
			loan.CreatedAt, loan.UpdatedAt,
		)
	} else {
		_, err = r.pool.Exec(ctx, query,
			loan.ID, loan.OrganizationID, loan.LoanRequestID, loan.ReservationID, loan.UnitID, loan.BorrowerUserID,
			string(loan.Status), loan.CheckoutTime, loan.ExpectedReturnTime, loan.ActualReturnTime,
			condVal, loan.ReturnNotes, loan.ProcessedByUserID,
			loan.CreatedAt, loan.UpdatedAt,
		)
	}
	return mapPgError(err)
}

func (r *Repository) GetLoan(ctx context.Context, id uuid.UUID) (*domain.Loan, error) {
	query := `
		SELECT
			id, organization_id, loan_request_id, reservation_id, unit_id, borrower_user_id,
			status, checkout_time, expected_return_time, actual_return_time,
			return_condition, return_notes, processed_by_user_id,
			created_at, updated_at
		FROM loans
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)
	return scanLoan(row)
}
func (r *Repository) GetLoanByID(ctx context.Context, id uuid.UUID) (*domain.Loan, error) {
	return r.GetLoan(ctx, id)
}

func (r *Repository) GetLoanByLoanRequestID(ctx context.Context, loanRequestID uuid.UUID) (*domain.Loan, error) {
	query := `
		SELECT
			id, organization_id, loan_request_id, reservation_id, unit_id, borrower_user_id,
			status, checkout_time, expected_return_time, actual_return_time,
			return_condition, return_notes, processed_by_user_id,
			created_at, updated_at
		FROM loans
		WHERE loan_request_id = $1
	`
	row := r.pool.QueryRow(ctx, query, loanRequestID)
	return scanLoan(row)
}

func (r *Repository) ListLoans(ctx context.Context, orgID uuid.UUID, borrowerUserID *uuid.UUID, status *domain.LoanStatus, limit, offset int) ([]domain.Loan, error) {
	query := `
		SELECT
			id, organization_id, loan_request_id, reservation_id, unit_id, borrower_user_id,
			status, checkout_time, expected_return_time, actual_return_time,
			return_condition, return_notes, processed_by_user_id,
			created_at, updated_at
		FROM loans
		WHERE organization_id = $1
		  AND ($2::uuid IS NULL OR borrower_user_id = $2)
		  AND ($3::varchar IS NULL OR status = $3)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`
	var statusVal *string
	if status != nil {
		s := string(*status)
		statusVal = &s
	}
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.pool.Query(ctx, query, orgID, borrowerUserID, statusVal, limit, offset)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer rows.Close()

	var results []domain.Loan
	for rows.Next() {
		loan, err := scanLoan(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning loan row: %w", err)
		}
		results = append(results, *loan)
	}
	return results, nil
}

func (r *Repository) UpdateLoan(ctx context.Context, tx pgx.Tx, loan *domain.Loan) error {
	query := `
		UPDATE loans
		SET status = $1,
		    checkout_time = $2,
		    actual_return_time = $3,
		    return_condition = $4,
		    return_notes = $5,
		    processed_by_user_id = $6,
		    updated_at = $7
		WHERE id = $8
	`
	var condVal *string
	if loan.ReturnCondition != nil {
		c := string(*loan.ReturnCondition)
		condVal = &c
	}

	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query,
			string(loan.Status), loan.CheckoutTime, loan.ActualReturnTime,
			condVal, loan.ReturnNotes, loan.ProcessedByUserID, loan.UpdatedAt, loan.ID,
		)
	} else {
		_, err = r.pool.Exec(ctx, query,
			string(loan.Status), loan.CheckoutTime, loan.ActualReturnTime,
			condVal, loan.ReturnNotes, loan.ProcessedByUserID, loan.UpdatedAt, loan.ID,
		)
	}
	return mapPgError(err)
}

func (r *Repository) ListOverdueLoans(ctx context.Context, orgID uuid.UUID) ([]domain.Loan, error) {
	query := `
		SELECT
			id, organization_id, loan_request_id, reservation_id, unit_id, borrower_user_id,
			status, checkout_time, expected_return_time, actual_return_time,
			return_condition, return_notes, processed_by_user_id,
			created_at, updated_at
		FROM loans
		WHERE organization_id = $1 AND status = 'overdue'
		ORDER BY expected_return_time ASC
	`
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer rows.Close()

	var results []domain.Loan
	for rows.Next() {
		loan, err := scanLoan(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning overdue loan row: %w", err)
		}
		results = append(results, *loan)
	}
	return results, nil
}

func (r *Repository) FindOverdueCheckedOutLoans(ctx context.Context, now time.Time) ([]domain.Loan, error) {
	query := `
		SELECT
			id, organization_id, loan_request_id, reservation_id, unit_id, borrower_user_id,
			status, checkout_time, expected_return_time, actual_return_time,
			return_condition, return_notes, processed_by_user_id,
			created_at, updated_at
		FROM loans
		WHERE status = 'checked_out' AND expected_return_time < $1
		ORDER BY expected_return_time ASC
	`
	rows, err := r.pool.Query(ctx, query, now)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer rows.Close()

	var results []domain.Loan
	for rows.Next() {
		loan, err := scanLoan(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning overdue loan candidate row: %w", err)
		}
		results = append(results, *loan)
	}
	return results, nil
}
func (r *Repository) TransitionLoanToOverdue(ctx context.Context, loanID uuid.UUID, now time.Time) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, mapPgError(err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE loans
		SET status = 'overdue', updated_at = $2
		WHERE id = $1 AND status = 'checked_out'
		RETURNING organization_id, unit_id, borrower_user_id, expected_return_time
	`
	var orgID, unitID, borrowerID uuid.UUID
	var expectedReturn time.Time

	err = tx.QueryRow(ctx, query, loanID, now).Scan(&orgID, &unitID, &borrowerID, &expectedReturn)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, mapPgError(err)
	}

	corrID := uuid.New().String()
	historyQuery := `
		INSERT INTO status_history (
			id, entity_type, entity_id, from_status, to_status,
			changed_by_user_id, reason, correlation_id, created_at
		) VALUES ($1, 'loan', $2, 'checked_out', 'overdue', $3, NULL, $4, $5)
	`
	if _, err := tx.Exec(ctx, historyQuery, uuid.New(), loanID, uuid.Nil, corrID, now); err != nil {
		return false, mapPgError(err)
	}

	outboxEv := event.BuildOutboxEvent(
		event.EventLoanOverdue,
		loanID.String(),
		orgID,
		corrID,
		loanID.String(),
		map[string]any{
			"loan_id":              loanID.String(),
			"unit_id":              unitID.String(),
			"borrower_user_id":     borrowerID.String(),
			"expected_return_time": expectedReturn.Format(time.RFC3339Nano),
		},
	)

	dataBytes, err := json.Marshal(outboxEv.Data)
	if err != nil {
		return false, fmt.Errorf("marshaling outbox event data: %w", err)
	}

	outboxQuery := `
		INSERT INTO outbox_events (
			event_id, event_type, event_version, source, subject, organization_id,
			occurred_at, correlation_id, causation_id, data, status, retry_count, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'pending', 0, $11)
	`
	if _, err := tx.Exec(ctx, outboxQuery,
		outboxEv.EventID, outboxEv.EventType, outboxEv.EventVersion, outboxEv.Source,
		outboxEv.Subject, outboxEv.OrganizationID, outboxEv.OccurredAt, outboxEv.CorrelationID,
		outboxEv.CausationID, dataBytes, outboxEv.CreatedAt,
	); err != nil {
		return false, mapPgError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, mapPgError(err)
	}

	return true, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanLoan(row rowScanner) (*domain.Loan, error) {
	var loan domain.Loan
	var statusStr string
	var condStr *string

	err := row.Scan(
		&loan.ID, &loan.OrganizationID, &loan.LoanRequestID, &loan.ReservationID, &loan.UnitID, &loan.BorrowerUserID,
		&statusStr, &loan.CheckoutTime, &loan.ExpectedReturnTime, &loan.ActualReturnTime,
		&condStr, &loan.ReturnNotes, &loan.ProcessedByUserID,
		&loan.CreatedAt, &loan.UpdatedAt,
	)
	if err != nil {
		return nil, mapPgError(err)
	}
	loan.Status = domain.LoanStatus(statusStr)
	if condStr != nil {
		c := domain.ReturnCondition(*condStr)
		loan.ReturnCondition = &c
	}
	return &loan, nil
}

// ---------------------------------------------------------------------------
// Checkout Tokens
// ---------------------------------------------------------------------------

func (r *Repository) CreateCheckoutToken(ctx context.Context, tx pgx.Tx, token *domain.CheckoutToken) error {
	query := `
		INSERT INTO checkout_tokens (id, loan_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query, token.ID, token.LoanID, token.TokenHash, token.ExpiresAt, token.CreatedAt)
	} else {
		_, err = r.pool.Exec(ctx, query, token.ID, token.LoanID, token.TokenHash, token.ExpiresAt, token.CreatedAt)
	}
	return mapPgError(err)
}

func (r *Repository) GetCheckoutTokenByHash(ctx context.Context, tokenHash string) (*domain.CheckoutToken, error) {
	query := `
		SELECT id, loan_id, token_hash, expires_at, used_at, created_at
		FROM checkout_tokens
		WHERE token_hash = $1
	`
	row := r.pool.QueryRow(ctx, query, tokenHash)
	var token domain.CheckoutToken
	err := row.Scan(&token.ID, &token.LoanID, &token.TokenHash, &token.ExpiresAt, &token.UsedAt, &token.CreatedAt)
	if err != nil {
		return nil, mapPgError(err)
	}
	return &token, nil
}

func (r *Repository) MarkCheckoutTokenUsed(ctx context.Context, tx pgx.Tx, id uuid.UUID, usedAt time.Time) error {
	query := `
		UPDATE checkout_tokens
		SET used_at = $1
		WHERE id = $2 AND used_at IS NULL
	`
	var cmd pgconn.CommandTag
	var err error
	if tx != nil {
		cmd, err = tx.Exec(ctx, query, usedAt, id)
	} else {
		cmd, err = r.pool.Exec(ctx, query, usedAt, id)
	}
	if err != nil {
		return mapPgError(err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrTokenAlreadyUsed
	}
	return nil
}

// ---------------------------------------------------------------------------
// Status History
// ---------------------------------------------------------------------------

func (r *Repository) CreateStatusHistory(ctx context.Context, tx pgx.Tx, history *domain.StatusHistory) error {
	query := `
		INSERT INTO status_history (
			id, entity_type, entity_id, from_status, to_status,
			changed_by_user_id, reason, correlation_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query,
			history.ID, history.EntityType, history.EntityID, history.FromStatus, history.ToStatus,
			history.ChangedByUserID, history.Reason, history.CorrelationID, history.CreatedAt,
		)
	} else {
		_, err = r.pool.Exec(ctx, query,
			history.ID, history.EntityType, history.EntityID, history.FromStatus, history.ToStatus,
			history.ChangedByUserID, history.Reason, history.CorrelationID, history.CreatedAt,
		)
	}
	return mapPgError(err)
}

func (r *Repository) GetStatusHistory(ctx context.Context, entityType string, entityID uuid.UUID) ([]domain.StatusHistory, error) {
	query := `
		SELECT id, entity_type, entity_id, from_status, to_status, changed_by_user_id, reason, correlation_id, created_at
		FROM status_history
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, entityType, entityID)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer rows.Close()

	var list []domain.StatusHistory
	for rows.Next() {
		var sh domain.StatusHistory
		if err := rows.Scan(
			&sh.ID, &sh.EntityType, &sh.EntityID, &sh.FromStatus, &sh.ToStatus,
			&sh.ChangedByUserID, &sh.Reason, &sh.CorrelationID, &sh.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning status history row: %w", err)
		}
		list = append(list, sh)
	}
	return list, nil
}

// ---------------------------------------------------------------------------
// Outbox Events
// ---------------------------------------------------------------------------

func (r *Repository) CreateOutboxEvent(ctx context.Context, tx pgx.Tx, event *domain.OutboxEvent) error {
	dataBytes, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("marshaling outbox event data: %w", err)
	}

	query := `
		INSERT INTO outbox_events (
			event_id, event_type, event_version, source, subject, organization_id,
			occurred_at, correlation_id, causation_id, data, status, retry_count, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'pending', 0, $11)
		RETURNING id
	`
	var row pgx.Row
	if tx != nil {
		row = tx.QueryRow(ctx, query,
			event.EventID, event.EventType, event.EventVersion, event.Source, event.Subject, event.OrganizationID,
			event.OccurredAt, event.CorrelationID, event.CausationID, dataBytes, event.CreatedAt,
		)
	} else {
		row = r.pool.QueryRow(ctx, query,
			event.EventID, event.EventType, event.EventVersion, event.Source, event.Subject, event.OrganizationID,
			event.OccurredAt, event.CorrelationID, event.CausationID, dataBytes, event.CreatedAt,
		)
	}
	if err := row.Scan(&event.ID); err != nil {
		return mapPgError(err)
	}
	return nil
}

func (r *Repository) ClaimPendingOutboxEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer tx.Rollback(ctx)

	query := `
		SELECT
			id, event_id, event_type, event_version, source, subject, organization_id,
			occurred_at, correlation_id, causation_id, data, status, retry_count,
			last_error, created_at, published_at
		FROM outbox_events
		WHERE status = 'pending' OR (status = 'in_flight' AND created_at < NOW() - INTERVAL '5 minutes')
		ORDER BY id ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`
	rows, err := tx.Query(ctx, query, limit)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer rows.Close()

	var events []domain.OutboxEvent
	var ids []int64
	for rows.Next() {
		var e domain.OutboxEvent
		var dataBytes []byte
		if err := rows.Scan(
			&e.ID, &e.EventID, &e.EventType, &e.EventVersion, &e.Source, &e.Subject, &e.OrganizationID,
			&e.OccurredAt, &e.CorrelationID, &e.CausationID, &dataBytes, &e.Status, &e.RetryCount,
			&e.LastError, &e.CreatedAt, &e.PublishedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning outbox event: %w", err)
		}
		if len(dataBytes) > 0 {
			if err := json.Unmarshal(dataBytes, &e.Data); err != nil {
				return nil, fmt.Errorf("unmarshaling outbox event data: %w", err)
			}
		}
		e.Status = "in_flight"
		events = append(events, e)
		ids = append(ids, e.ID)
	}
	rows.Close()

	if len(ids) > 0 {
		updateQuery := `
			UPDATE outbox_events
			SET status = 'in_flight'
			WHERE id = ANY($1)
		`
		if _, err := tx.Exec(ctx, updateQuery, ids); err != nil {
			return nil, mapPgError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, mapPgError(err)
	}

	return events, nil
}

func (r *Repository) GetPendingOutboxEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	return r.ClaimPendingOutboxEvents(ctx, limit)
}

func (r *Repository) MarkOutboxEventPublished(ctx context.Context, id int64) error {
	query := `
		UPDATE outbox_events
		SET status = 'published', published_at = $1
		WHERE id = $2
	`
	_, err := r.pool.Exec(ctx, query, time.Now(), id)
	return mapPgError(err)
}

func (r *Repository) MarkOutboxEventFailed(ctx context.Context, id int64, errStr string) error {
	query := `
		UPDATE outbox_events
		SET status = CASE WHEN retry_count + 1 >= 5 THEN 'failed' ELSE 'pending' END,
		    retry_count = retry_count + 1,
		    last_error = $1
		WHERE id = $2
	`
	_, err := r.pool.Exec(ctx, query, errStr, id)
	return mapPgError(err)
}

func (r *Repository) ResetOutboxEventsToPending(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	query := `
		UPDATE outbox_events
		SET status = 'pending'
		WHERE id = ANY($1) AND status = 'in_flight'
	`
	_, err := r.pool.Exec(ctx, query, ids)
	return mapPgError(err)
}

// ---------------------------------------------------------------------------
// Idempotency Keys
// ---------------------------------------------------------------------------

func (r *Repository) GetIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*domain.IdempotencyRecord, error) {
	query := `
		SELECT key, organization_id, user_id, request_path, request_hash, response_code, response_body, created_at, expires_at
		FROM idempotency_keys
		WHERE organization_id = $1 AND key = $2
	`
	row := r.pool.QueryRow(ctx, query, orgID, key)
	var rec domain.IdempotencyRecord
	err := row.Scan(
		&rec.Key, &rec.OrganizationID, &rec.UserID, &rec.RequestPath, &rec.RequestHash,
		&rec.ResponseCode, &rec.ResponseBody, &rec.CreatedAt, &rec.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, mapPgError(err)
	}
	return &rec, nil
}

func (r *Repository) SaveIdempotencyKey(ctx context.Context, record *domain.IdempotencyRecord) error {
	query := `
		INSERT INTO idempotency_keys (
			key, organization_id, user_id, request_path, request_hash,
			response_code, response_body, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query,
		record.Key, record.OrganizationID, record.UserID, record.RequestPath, record.RequestHash,
		record.ResponseCode, record.ResponseBody, record.CreatedAt, record.ExpiresAt,
	)
	return mapPgError(err)
}
