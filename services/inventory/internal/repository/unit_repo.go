package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"borrowhub/services/inventory/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UnitRepository struct {
	pool *pgxpool.Pool
}

func NewUnitRepository(pool *pgxpool.Pool) *UnitRepository {
	return &UnitRepository{pool: pool}
}

func (r *UnitRepository) Create(ctx context.Context, u *domain.Unit, createdBy uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	insertUnit := `
		INSERT INTO units (id, organization_id, item_id, barcode, serial_number, status, condition, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err = tx.Exec(
		ctx,
		insertUnit,
		u.ID,
		u.OrganizationID,
		u.ItemID,
		u.Barcode,
		u.SerialNumber,
		u.Status,
		u.Condition,
		u.Notes,
		u.CreatedAt,
		u.UpdatedAt,
	)
	if err != nil {
		return mapPgError(err)
	}

	insertLog := `
		INSERT INTO condition_logs (id, unit_id, previous_condition, new_condition, reason, changed_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	logID, err := uuid.NewV7()
	if err != nil {
		logID = uuid.New()
	}
	initialReason := "Initial condition on creation"
	_, err = tx.Exec(
		ctx,
		insertLog,
		logID,
		u.ID,
		nil,
		u.Condition,
		initialReason,
		createdBy,
		u.CreatedAt,
	)
	if err != nil {
		return mapPgError(err)
	}

	return tx.Commit(ctx)
}

func (r *UnitRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Unit, error) {
	query := `
		SELECT id, organization_id, item_id, barcode, serial_number, status, condition, notes, created_at, updated_at
		FROM units
		WHERE id = $1
	`
	var u domain.Unit
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.OrganizationID,
		&u.ItemID,
		&u.Barcode,
		&u.SerialNumber,
		&u.Status,
		&u.Condition,
		&u.Notes,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return nil, mapPgError(err)
	}
	return &u, nil
}

func (r *UnitRepository) ListByItem(ctx context.Context, itemID uuid.UUID) ([]*domain.Unit, error) {
	query := `
		SELECT id, organization_id, item_id, barcode, serial_number, status, condition, notes, created_at, updated_at
		FROM units
		WHERE item_id = $1
		ORDER BY barcode ASC
	`
	rows, err := r.pool.Query(ctx, query, itemID)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer rows.Close()

	var units []*domain.Unit
	for rows.Next() {
		var u domain.Unit
		if err := rows.Scan(
			&u.ID,
			&u.OrganizationID,
			&u.ItemID,
			&u.Barcode,
			&u.SerialNumber,
			&u.Status,
			&u.Condition,
			&u.Notes,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, mapPgError(err)
		}
		units = append(units, &u)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPgError(err)
	}
	if units == nil {
		units = []*domain.Unit{}
	}
	return units, nil
}

func (r *UnitRepository) ListEligibleByItem(ctx context.Context, itemID uuid.UUID) ([]*domain.Unit, error) {
	query := `
		SELECT id, organization_id, item_id, barcode, serial_number, status, condition, notes, created_at, updated_at
		FROM units
		WHERE item_id = $1 AND status = 'available'
		ORDER BY barcode ASC
	`
	rows, err := r.pool.Query(ctx, query, itemID)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer rows.Close()

	var units []*domain.Unit
	for rows.Next() {
		var u domain.Unit
		if err := rows.Scan(
			&u.ID,
			&u.OrganizationID,
			&u.ItemID,
			&u.Barcode,
			&u.SerialNumber,
			&u.Status,
			&u.Condition,
			&u.Notes,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, mapPgError(err)
		}
		units = append(units, &u)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPgError(err)
	}
	if units == nil {
		units = []*domain.Unit{}
	}
	return units, nil
}

func (r *UnitRepository) Update(
	ctx context.Context,
	unitID uuid.UUID,
	input domain.UpdateUnitInput,
	changedBy *uuid.UUID,
) (*domain.Unit, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	querySelect := `
		SELECT id, organization_id, item_id, barcode, serial_number, status, condition, notes, created_at, updated_at
		FROM units
		WHERE id = $1
		FOR UPDATE
	`
	var u domain.Unit
	err = tx.QueryRow(ctx, querySelect, unitID).Scan(
		&u.ID,
		&u.OrganizationID,
		&u.ItemID,
		&u.Barcode,
		&u.SerialNumber,
		&u.Status,
		&u.Condition,
		&u.Notes,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return nil, mapPgError(err)
	}

	currentStatus := u.Status
	currentCondition := u.Condition

	newStatus := currentStatus
	statusChanged := false
	if input.Status != nil {
		if !domain.CanAdminTransition(currentStatus, *input.Status) {
			return nil, domain.ErrInvalidTransition
		}
		newStatus = *input.Status
		statusChanged = (newStatus != currentStatus)
	}

	newCondition := currentCondition
	conditionChanged := false
	if input.Condition != nil {
		newCondition = *input.Condition
		conditionChanged = (newCondition != currentCondition)
	}

	if input.Barcode != nil {
		u.Barcode = domain.NormalizeBarcode(*input.Barcode)
	}
	if input.SerialNumber != nil {
		s := strings.TrimSpace(*input.SerialNumber)
		if s == "" {
			u.SerialNumber = nil
		} else {
			u.SerialNumber = &s
		}
	}
	if input.Notes != nil {
		u.Notes = input.Notes
	}

	u.Status = newStatus
	u.Condition = newCondition
	u.UpdatedAt = time.Now()

	queryUpdate := `
		UPDATE units
		SET barcode = $2, serial_number = $3, status = $4, condition = $5, notes = $6, updated_at = $7
		WHERE id = $1
	`
	cmd, err := tx.Exec(
		ctx,
		queryUpdate,
		u.ID,
		u.Barcode,
		u.SerialNumber,
		u.Status,
		u.Condition,
		u.Notes,
		u.UpdatedAt,
	)
	if err != nil {
		return nil, mapPgError(err)
	}
	if cmd.RowsAffected() == 0 {
		return nil, domain.ErrNotFound
	}

	reasonProvided := input.Reason != nil && *input.Reason != ""
	if (conditionChanged || statusChanged || reasonProvided) && changedBy != nil {
		logID, err := uuid.NewV7()
		if err != nil {
			logID = uuid.New()
		}

		prevCondition := &currentCondition
		var logReason *string
		if statusChanged {
			if reasonProvided {
				rStr := fmt.Sprintf("[status: %s] %s", newStatus, *input.Reason)
				logReason = &rStr
			} else {
				rStr := fmt.Sprintf("[status: %s]", newStatus)
				logReason = &rStr
			}
		} else if reasonProvided {
			logReason = input.Reason
		}

		insertLog := `
			INSERT INTO condition_logs (id, unit_id, previous_condition, new_condition, reason, changed_by, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`
		_, err = tx.Exec(
			ctx,
			insertLog,
			logID,
			u.ID,
			prevCondition,
			u.Condition,
			logReason,
			*changedBy,
			time.Now(),
		)
		if err != nil {
			return nil, mapPgError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &u, nil
}

func (r *UnitRepository) TransitionStatus(ctx context.Context, unitID uuid.UUID, fromStatus, toStatus domain.UnitStatus) (*domain.Unit, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	querySelect := `
		SELECT id, organization_id, item_id, barcode, serial_number, status, condition, notes, created_at, updated_at
		FROM units
		WHERE id = $1
		FOR UPDATE
	`
	var u domain.Unit
	err = tx.QueryRow(ctx, querySelect, unitID).Scan(
		&u.ID,
		&u.OrganizationID,
		&u.ItemID,
		&u.Barcode,
		&u.SerialNumber,
		&u.Status,
		&u.Condition,
		&u.Notes,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return nil, mapPgError(err)
	}

	if u.Status != fromStatus {
		return nil, domain.ErrStatusMismatch
	}

	if !domain.CanTransition(fromStatus, toStatus) {
		return nil, domain.ErrInvalidTransition
	}

	now := time.Now()
	queryUpdate := `
		UPDATE units
		SET status = $2, updated_at = $3
		WHERE id = $1
		RETURNING updated_at
	`
	if err := tx.QueryRow(ctx, queryUpdate, unitID, toStatus, now).Scan(&u.UpdatedAt); err != nil {
		return nil, mapPgError(err)
	}
	u.Status = toStatus

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &u, nil
}

func (r *UnitRepository) GetConditionLogs(ctx context.Context, unitID uuid.UUID) ([]*domain.ConditionLog, error) {
	query := `
		SELECT id, unit_id, previous_condition, new_condition, reason, changed_by, created_at
		FROM condition_logs
		WHERE unit_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, unitID)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer rows.Close()

	var logs []*domain.ConditionLog
	for rows.Next() {
		var log domain.ConditionLog
		if err := rows.Scan(
			&log.ID,
			&log.UnitID,
			&log.PreviousCondition,
			&log.NewCondition,
			&log.Reason,
			&log.ChangedBy,
			&log.CreatedAt,
		); err != nil {
			return nil, mapPgError(err)
		}
		logs = append(logs, &log)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPgError(err)
	}
	if logs == nil {
		logs = []*domain.ConditionLog{}
	}
	return logs, nil
}
