package service

import (
	"borrowhub/services/lending/internal/auth"
	"borrowhub/services/lending/internal/client"
	"borrowhub/services/lending/internal/domain"
	"borrowhub/services/lending/internal/event"
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"log/slog"
	"time"
)

type ApprovedRequestResult struct {
	LoanRequest domain.LoanRequest `json:"loan_request"`
	Loan        domain.Loan        `json:"loan"`
	PlainToken  string             `json:"checkout_token"`
}

type LendingService struct {
	repo            LendingRepository
	identityClient  auth.IdentityClient
	inventoryClient client.InventoryClient
}

func NewLendingService(
	repo LendingRepository,
	identityClient auth.IdentityClient,
	inventoryClient client.InventoryClient,
) *LendingService {
	return &LendingService{
		repo:            repo,
		identityClient:  identityClient,
		inventoryClient: inventoryClient,
	}
}

func (s *LendingService) CheckUserMembership(ctx context.Context, orgID, userID uuid.UUID) (*auth.MemberInfo, error) {
	member, err := s.identityClient.GetMember(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrForbidden
		}
		return nil, fmt.Errorf("checking membership: %w", err)
	}
	if !member.IsMember {
		return nil, domain.ErrForbidden
	}
	return member, nil
}

// ---------------------------------------------------------------------------
// Availability
// ---------------------------------------------------------------------------

func (s *LendingService) GetItemAvailability(
	ctx context.Context,
	orgID, userID, itemID uuid.UUID,
	startTime, endTime time.Time,
) (*domain.ItemAvailability, error) {
	if _, err := s.CheckUserMembership(ctx, orgID, userID); err != nil {
		return nil, err
	}

	if startTime.IsZero() || endTime.IsZero() || !startTime.Before(endTime) {
		return nil, domain.NewValidationError("invalid time window", map[string]any{
			"start_time": "must be before end_time",
		})
	}

	eligible, err := s.inventoryClient.GetEligibleUnits(ctx, itemID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("fetching eligible units: %w", err)
	}

	totalUnits := len(eligible)
	if totalUnits == 0 {
		return &domain.ItemAvailability{
			ItemID:         itemID,
			StartTime:      startTime,
			EndTime:        endTime,
			TotalUnits:     0,
			AvailableUnits: 0,
			IsAvailable:    false,
		}, nil
	}

	unitIDs := make([]uuid.UUID, 0, totalUnits)
	for _, u := range eligible {
		unitIDs = append(unitIDs, u.ID)
	}

	overlapping, err := s.repo.FindOverlappingUnitIDs(ctx, unitIDs, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("checking overlapping reservations: %w", err)
	}

	overlapMap := make(map[uuid.UUID]bool, len(overlapping))
	for _, id := range overlapping {
		overlapMap[id] = true
	}

	availableIDs := make([]uuid.UUID, 0, totalUnits)
	for _, id := range unitIDs {
		if !overlapMap[id] {
			availableIDs = append(availableIDs, id)
		}
	}

	return &domain.ItemAvailability{
		ItemID:         itemID,
		StartTime:      startTime,
		EndTime:        endTime,
		TotalUnits:     totalUnits,
		AvailableUnits: len(availableIDs),
		IsAvailable:    len(availableIDs) > 0,
		EligibleUnits:  availableIDs,
	}, nil
}

// ---------------------------------------------------------------------------
// Loan Requests
// ---------------------------------------------------------------------------

func (s *LendingService) CreateLoanRequest(
	ctx context.Context,
	orgID, userID uuid.UUID,
	input domain.CreateLoanRequestInput,
	correlationID string,
) (*domain.LoanRequest, error) {
	if _, err := s.CheckUserMembership(ctx, orgID, userID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if err := domain.ValidateCreateLoanRequestInput(input, now); err != nil {
		return nil, err
	}

	// Verify availability
	avail, err := s.GetItemAvailability(ctx, orgID, userID, input.ItemID, input.StartTime, input.EndTime)
	if err != nil {
		return nil, err
	}
	if avail.AvailableUnits < input.Quantity {
		return nil, domain.ErrNoUnitsAvailable
	}

	req := domain.LoanRequest{
		ID:              uuid.New(),
		OrganizationID:  orgID,
		RequesterUserID: userID,
		ItemID:          input.ItemID,
		Quantity:        input.Quantity,
		StartTime:       input.StartTime,
		EndTime:         input.EndTime,
		Purpose:         input.Purpose,
		Notes:           input.Notes,
		Status:          domain.RequestStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	err = s.repo.RunInTx(ctx, func(tx pgx.Tx) error {
		if err := s.repo.CreateLoanRequest(ctx, tx, &req); err != nil {
			return err
		}

		statusHist := domain.StatusHistory{
			ID:              uuid.New(),
			EntityType:      "loan_request",
			EntityID:        req.ID,
			FromStatus:      "",
			ToStatus:        string(domain.RequestStatusPending),
			ChangedByUserID: userID,
			Reason:          nil,
			CorrelationID:   correlationID,
			CreatedAt:       now,
		}
		if err := s.repo.CreateStatusHistory(ctx, tx, &statusHist); err != nil {
			return err
		}

		outboxEv := event.BuildOutboxEvent(
			event.EventLoanRequested,
			req.ID.String(),
			orgID,
			correlationID,
			req.ID.String(),
			map[string]any{
				"loan_request_id":   req.ID.String(),
				"item_id":           req.ItemID.String(),
				"requester_user_id": req.RequesterUserID.String(),
				"quantity":          req.Quantity,
				"start_time":        req.StartTime.Format(time.RFC3339Nano),
				"end_time":          req.EndTime.Format(time.RFC3339Nano),
				"purpose":           req.Purpose,
			},
		)
		return s.repo.CreateOutboxEvent(ctx, tx, &outboxEv)
	})

	if err != nil {
		return nil, fmt.Errorf("creating loan request: %w", err)
	}

	return &req, nil
}

func (s *LendingService) ListLoanRequests(
	ctx context.Context,
	orgID, userID uuid.UUID,
	role string,
	requesterFilter *uuid.UUID,
	statusFilter *domain.RequestStatus,
	limit, offset int,
) ([]domain.LoanRequest, error) {
	if _, err := s.CheckUserMembership(ctx, orgID, userID); err != nil {
		return nil, err
	}

	if !domain.IsAdminOrOwner(role) {
		requesterFilter = &userID
	}

	return s.repo.ListLoanRequests(ctx, orgID, requesterFilter, statusFilter, limit, offset)
}

func (s *LendingService) GetLoanRequest(
	ctx context.Context,
	orgID, userID uuid.UUID,
	role string,
	requestID uuid.UUID,
) (*domain.LoanRequest, error) {
	if _, err := s.CheckUserMembership(ctx, orgID, userID); err != nil {
		return nil, err
	}

	req, err := s.repo.GetLoanRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if req.OrganizationID != orgID {
		return nil, domain.ErrNotFound
	}

	if !domain.IsAdminOrOwner(role) && req.RequesterUserID != userID {
		return nil, domain.ErrForbidden
	}

	return req, nil
}
func (s *LendingService) GetLoanRequestByID(ctx context.Context, requestID uuid.UUID) (*domain.LoanRequest, error) {
	return s.repo.GetLoanRequest(ctx, requestID)
}

func (s *LendingService) ApproveLoanRequest(
	ctx context.Context,
	orgID, adminID uuid.UUID,
	role string,
	requestID uuid.UUID,
	reviewNotes *string,
	correlationID string,
) (*ApprovedRequestResult, error) {
	if _, err := s.CheckUserMembership(ctx, orgID, adminID); err != nil {
		return nil, err
	}

	req, err := s.repo.GetLoanRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if req.OrganizationID != orgID {
		return nil, domain.ErrNotFound
	}

	if err := domain.ValidateRequestTransition(req.Status, domain.RequestStatusApproved, role, false); err != nil {
		return nil, err
	}

	// Find available unit
	avail, err := s.GetItemAvailability(ctx, orgID, adminID, req.ItemID, req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}
	if len(avail.EligibleUnits) == 0 {
		return nil, domain.ErrNoUnitsAvailable
	}
	allocatedUnitID := avail.EligibleUnits[0]

	// CAS inventory unit: available -> reserved
	_, err = s.inventoryClient.TransitionUnitStatus(ctx, allocatedUnitID, "available", "reserved")
	if err != nil {
		return nil, fmt.Errorf("transitioning unit to reserved: %w", err)
	}

	plainToken, tokenHash, err := domain.GenerateSecureToken()
	if err != nil {
		if _, revErr := s.inventoryClient.TransitionUnitStatus(ctx, allocatedUnitID, "reserved", "available"); revErr != nil {
			slog.WarnContext(ctx, "failed to revert inventory unit status after token generation error", "unit_id", allocatedUnitID, "error", revErr)
		}
		return nil, fmt.Errorf("generating token: %w", err)
	}

	now := time.Now().UTC()
	req.Status = domain.RequestStatusApproved
	req.ReviewerUserID = &adminID
	req.ReviewNotes = reviewNotes
	req.ReviewedAt = &now
	req.UpdatedAt = now

	res := domain.Reservation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		LoanRequestID:  req.ID,
		UnitID:         allocatedUnitID,
		StartTime:      req.StartTime,
		EndTime:        req.EndTime,
		Status:         domain.ReservationStatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	loan := domain.Loan{
		ID:                 uuid.New(),
		OrganizationID:     orgID,
		LoanRequestID:      req.ID,
		ReservationID:      res.ID,
		UnitID:             allocatedUnitID,
		BorrowerUserID:     req.RequesterUserID,
		Status:             domain.LoanStatusPendingCheckout,
		ExpectedReturnTime: req.EndTime,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	token := domain.CheckoutToken{
		ID:        uuid.New(),
		LoanID:    loan.ID,
		TokenHash: tokenHash,
		ExpiresAt: req.EndTime,
		CreatedAt: now,
	}

	err = s.repo.RunInTx(ctx, func(tx pgx.Tx) error {
		if err := s.repo.UpdateLoanRequest(ctx, tx, req); err != nil {
			return err
		}
		if err := s.repo.CreateReservation(ctx, tx, &res); err != nil {
			return err
		}
		if err := s.repo.CreateLoan(ctx, tx, &loan); err != nil {
			return err
		}
		if err := s.repo.CreateCheckoutToken(ctx, tx, &token); err != nil {
			return err
		}

		shReq := domain.StatusHistory{
			ID:              uuid.New(),
			EntityType:      "loan_request",
			EntityID:        req.ID,
			FromStatus:      string(domain.RequestStatusPending),
			ToStatus:        string(domain.RequestStatusApproved),
			ChangedByUserID: adminID,
			Reason:          reviewNotes,
			CorrelationID:   correlationID,
			CreatedAt:       now,
		}
		if err := s.repo.CreateStatusHistory(ctx, tx, &shReq); err != nil {
			return err
		}

		shLoan := domain.StatusHistory{
			ID:              uuid.New(),
			EntityType:      "loan",
			EntityID:        loan.ID,
			FromStatus:      "",
			ToStatus:        string(domain.LoanStatusPendingCheckout),
			ChangedByUserID: adminID,
			Reason:          nil,
			CorrelationID:   correlationID,
			CreatedAt:       now,
		}
		if err := s.repo.CreateStatusHistory(ctx, tx, &shLoan); err != nil {
			return err
		}

		outboxEv := event.BuildOutboxEvent(
			event.EventLoanApproved,
			loan.ID.String(),
			orgID,
			correlationID,
			req.ID.String(),
			map[string]any{
				"loan_request_id":  req.ID.String(),
				"loan_id":          loan.ID.String(),
				"reservation_id":   res.ID.String(),
				"unit_id":          allocatedUnitID.String(),
				"borrower_user_id": req.RequesterUserID.String(),
				"admin_user_id":    adminID.String(),
			},
		)
		return s.repo.CreateOutboxEvent(ctx, tx, &outboxEv)
	})

	if err != nil {
		if _, revErr := s.inventoryClient.TransitionUnitStatus(ctx, allocatedUnitID, "reserved", "available"); revErr != nil {
			slog.WarnContext(ctx, "failed to revert inventory unit status after approval error", "unit_id", allocatedUnitID, "error", revErr)
		}
		return nil, fmt.Errorf("approving loan request: %w", err)
	}

	return &ApprovedRequestResult{
		LoanRequest: *req,
		Loan:        loan,
		PlainToken:  plainToken,
	}, nil
}

func (s *LendingService) RejectLoanRequest(
	ctx context.Context,
	orgID, adminID uuid.UUID,
	role string,
	requestID uuid.UUID,
	reviewNotes *string,
	correlationID string,
) (*domain.LoanRequest, error) {
	if _, err := s.CheckUserMembership(ctx, orgID, adminID); err != nil {
		return nil, err
	}

	req, err := s.repo.GetLoanRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if req.OrganizationID != orgID {
		return nil, domain.ErrNotFound
	}

	if err := domain.ValidateRequestTransition(req.Status, domain.RequestStatusRejected, role, false); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	req.Status = domain.RequestStatusRejected
	req.ReviewerUserID = &adminID
	req.ReviewNotes = reviewNotes
	req.ReviewedAt = &now
	req.UpdatedAt = now

	err = s.repo.RunInTx(ctx, func(tx pgx.Tx) error {
		if err := s.repo.UpdateLoanRequest(ctx, tx, req); err != nil {
			return err
		}

		sh := domain.StatusHistory{
			ID:              uuid.New(),
			EntityType:      "loan_request",
			EntityID:        req.ID,
			FromStatus:      string(domain.RequestStatusPending),
			ToStatus:        string(domain.RequestStatusRejected),
			ChangedByUserID: adminID,
			Reason:          reviewNotes,
			CorrelationID:   correlationID,
			CreatedAt:       now,
		}
		if err := s.repo.CreateStatusHistory(ctx, tx, &sh); err != nil {
			return err
		}

		outboxEv := event.BuildOutboxEvent(
			event.EventLoanRejected,
			req.ID.String(),
			orgID,
			correlationID,
			req.ID.String(),
			map[string]any{
				"loan_request_id": req.ID.String(),
				"admin_user_id":   adminID.String(),
				"review_notes":    reviewNotes,
			},
		)
		return s.repo.CreateOutboxEvent(ctx, tx, &outboxEv)
	})

	if err != nil {
		return nil, fmt.Errorf("rejecting loan request: %w", err)
	}

	return req, nil
}

func (s *LendingService) CancelLoanRequest(
	ctx context.Context,
	orgID, userID uuid.UUID,
	role string,
	requestID uuid.UUID,
	reason *string,
	correlationID string,
) (*domain.LoanRequest, error) {
	if _, err := s.CheckUserMembership(ctx, orgID, userID); err != nil {
		return nil, err
	}

	req, err := s.repo.GetLoanRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if req.OrganizationID != orgID {
		return nil, domain.ErrNotFound
	}

	isRequester := req.RequesterUserID == userID
	if err := domain.ValidateRequestTransition(req.Status, domain.RequestStatusCancelled, role, isRequester); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	oldStatus := req.Status
	req.Status = domain.RequestStatusCancelled
	req.UpdatedAt = now

	if oldStatus == domain.RequestStatusPending {
		err = s.repo.RunInTx(ctx, func(tx pgx.Tx) error {
			if err := s.repo.UpdateLoanRequest(ctx, tx, req); err != nil {
				return err
			}

			sh := domain.StatusHistory{
				ID:              uuid.New(),
				EntityType:      "loan_request",
				EntityID:        req.ID,
				FromStatus:      string(domain.RequestStatusPending),
				ToStatus:        string(domain.RequestStatusCancelled),
				ChangedByUserID: userID,
				Reason:          reason,
				CorrelationID:   correlationID,
				CreatedAt:       now,
			}
			if err := s.repo.CreateStatusHistory(ctx, tx, &sh); err != nil {
				return err
			}

			outboxEv := event.BuildOutboxEvent(
				event.EventLoanCancelled,
				req.ID.String(),
				orgID,
				correlationID,
				req.ID.String(),
				map[string]any{
					"loan_request_id": req.ID.String(),
					"user_id":         userID.String(),
					"reason":          reason,
				},
			)
			return s.repo.CreateOutboxEvent(ctx, tx, &outboxEv)
		})
		if err != nil {
			return nil, fmt.Errorf("cancelling pending loan request: %w", err)
		}
		return req, nil
	}

	// Approved request cancellation
	res, err := s.repo.GetReservationByLoanRequestID(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("fetching reservation: %w", err)
	}

	loan, err := s.repo.GetLoanByLoanRequestID(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("fetching loan: %w", err)
	}

	if loan.Status != domain.LoanStatusPendingCheckout {
		return nil, fmt.Errorf("%w: cannot cancel loan with status %s", domain.ErrInvalidTransition, loan.Status)
	}

	loan.Status = domain.LoanStatusCancelled
	loan.UpdatedAt = now

	err = s.repo.RunInTx(ctx, func(tx pgx.Tx) error {
		if err := s.repo.UpdateLoanRequest(ctx, tx, req); err != nil {
			return err
		}
		if err := s.repo.UpdateReservationStatus(ctx, tx, res.ID, domain.ReservationStatusCancelled); err != nil {
			return err
		}
		if err := s.repo.UpdateLoan(ctx, tx, loan); err != nil {
			return err
		}

		shReq := domain.StatusHistory{
			ID:              uuid.New(),
			EntityType:      "loan_request",
			EntityID:        req.ID,
			FromStatus:      string(domain.RequestStatusApproved),
			ToStatus:        string(domain.RequestStatusCancelled),
			ChangedByUserID: userID,
			Reason:          reason,
			CorrelationID:   correlationID,
			CreatedAt:       now,
		}
		if err := s.repo.CreateStatusHistory(ctx, tx, &shReq); err != nil {
			return err
		}

		shLoan := domain.StatusHistory{
			ID:              uuid.New(),
			EntityType:      "loan",
			EntityID:        loan.ID,
			FromStatus:      string(domain.LoanStatusPendingCheckout),
			ToStatus:        string(domain.LoanStatusCancelled),
			ChangedByUserID: userID,
			Reason:          reason,
			CorrelationID:   correlationID,
			CreatedAt:       now,
		}
		if err := s.repo.CreateStatusHistory(ctx, tx, &shLoan); err != nil {
			return err
		}

		outboxEv := event.BuildOutboxEvent(
			event.EventLoanCancelled,
			req.ID.String(),
			orgID,
			correlationID,
			req.ID.String(),
			map[string]any{
				"loan_request_id": req.ID.String(),
				"loan_id":         loan.ID.String(),
				"unit_id":         res.UnitID.String(),
				"user_id":         userID.String(),
				"reason":          reason,
			},
		)
		return s.repo.CreateOutboxEvent(ctx, tx, &outboxEv)
	})

	if err != nil {
		return nil, fmt.Errorf("cancelling approved loan request: %w", err)
	}

	// Revert inventory status: reserved -> available
	if _, revErr := s.inventoryClient.TransitionUnitStatus(ctx, res.UnitID, "reserved", "available"); revErr != nil {
		slog.WarnContext(ctx, "failed to revert inventory unit status after loan request cancellation", "unit_id", res.UnitID, "error", revErr)
	}
	return req, nil
}

// ---------------------------------------------------------------------------
// Loans & Checkout / Return
// ---------------------------------------------------------------------------

func (s *LendingService) GetLoan(
	ctx context.Context,
	orgID, userID uuid.UUID,
	role string,
	loanID uuid.UUID,
) (*domain.Loan, error) {
	if _, err := s.CheckUserMembership(ctx, orgID, userID); err != nil {
		return nil, err
	}

	loan, err := s.repo.GetLoan(ctx, loanID)
	if err != nil {
		return nil, err
	}
	if loan.OrganizationID != orgID {
		return nil, domain.ErrNotFound
	}

	if !domain.IsAdminOrOwner(role) && loan.BorrowerUserID != userID {
		return nil, domain.ErrForbidden
	}

	return loan, nil
}
func (s *LendingService) GetLoanByID(ctx context.Context, loanID uuid.UUID) (*domain.Loan, error) {
	return s.repo.GetLoan(ctx, loanID)
}

func (s *LendingService) ListLoans(
	ctx context.Context,
	orgID, userID uuid.UUID,
	role string,
	status *domain.LoanStatus,
	limit, offset int,
) ([]domain.Loan, error) {
	if _, err := s.CheckUserMembership(ctx, orgID, userID); err != nil {
		return nil, err
	}

	var borrowerFilter *uuid.UUID
	if !domain.IsAdminOrOwner(role) {
		borrowerFilter = &userID
	}

	return s.repo.ListLoans(ctx, orgID, borrowerFilter, status, limit, offset)
}

func (s *LendingService) CheckoutLoan(
	ctx context.Context,
	orgID, adminID uuid.UUID,
	role string,
	loanID uuid.UUID,
	plainToken string,
	correlationID string,
) (*domain.Loan, error) {
	if _, err := s.CheckUserMembership(ctx, orgID, adminID); err != nil {
		return nil, err
	}

	if !domain.IsAdminOrOwner(role) {
		return nil, domain.ErrUnauthorizedAction
	}

	loan, err := s.repo.GetLoan(ctx, loanID)
	if err != nil {
		return nil, err
	}
	if loan.OrganizationID != orgID {
		return nil, domain.ErrNotFound
	}

	if err := domain.ValidateLoanTransition(loan.Status, domain.LoanStatusCheckedOut); err != nil {
		return nil, err
	}

	// Verify token
	tokenHash := domain.HashToken(plainToken)
	token, err := s.repo.GetCheckoutTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrInvalidToken
		}
		return nil, err
	}
	if token.LoanID != loan.ID {
		return nil, domain.ErrInvalidToken
	}
	if token.UsedAt != nil {
		return nil, domain.ErrTokenAlreadyUsed
	}
	if time.Now().After(token.ExpiresAt) {
		return nil, domain.ErrTokenExpired
	}

	// CAS inventory unit: reserved -> borrowed
	_, err = s.inventoryClient.TransitionUnitStatus(ctx, loan.UnitID, "reserved", "borrowed")
	if err != nil {
		return nil, fmt.Errorf("transitioning unit to borrowed: %w", err)
	}

	now := time.Now().UTC()
	loan.Status = domain.LoanStatusCheckedOut
	loan.CheckoutTime = &now
	loan.ProcessedByUserID = &adminID
	loan.UpdatedAt = now

	err = s.repo.RunInTx(ctx, func(tx pgx.Tx) error {
		if err := s.repo.MarkCheckoutTokenUsed(ctx, tx, token.ID, now); err != nil {
			return err
		}
		if err := s.repo.UpdateLoan(ctx, tx, loan); err != nil {
			return err
		}

		sh := domain.StatusHistory{
			ID:              uuid.New(),
			EntityType:      "loan",
			EntityID:        loan.ID,
			FromStatus:      string(domain.LoanStatusPendingCheckout),
			ToStatus:        string(domain.LoanStatusCheckedOut),
			ChangedByUserID: adminID,
			Reason:          nil,
			CorrelationID:   correlationID,
			CreatedAt:       now,
		}
		if err := s.repo.CreateStatusHistory(ctx, tx, &sh); err != nil {
			return err
		}

		outboxEv := event.BuildOutboxEvent(
			event.EventLoanCheckedOut,
			loan.ID.String(),
			orgID,
			correlationID,
			loan.ID.String(),
			map[string]any{
				"loan_id":          loan.ID.String(),
				"loan_request_id":  loan.LoanRequestID.String(),
				"unit_id":          loan.UnitID.String(),
				"borrower_user_id": loan.BorrowerUserID.String(),
				"admin_user_id":    adminID.String(),
				"checkout_time":    now.Format(time.RFC3339Nano),
			},
		)
		return s.repo.CreateOutboxEvent(ctx, tx, &outboxEv)
	})

	if err != nil {
		if _, revErr := s.inventoryClient.TransitionUnitStatus(ctx, loan.UnitID, "borrowed", "reserved"); revErr != nil {
			slog.WarnContext(ctx, "failed to revert inventory unit status after checkout error", "unit_id", loan.UnitID, "error", revErr)
		}
		return nil, fmt.Errorf("checking out loan: %w", err)
	}

	return loan, nil
}

func (s *LendingService) ReturnLoan(
	ctx context.Context,
	orgID, adminID uuid.UUID,
	role string,
	loanID uuid.UUID,
	input domain.ReturnLoanInput,
	correlationID string,
) (*domain.Loan, error) {
	if _, err := s.CheckUserMembership(ctx, orgID, adminID); err != nil {
		return nil, err
	}

	if !domain.IsAdminOrOwner(role) {
		return nil, domain.ErrUnauthorizedAction
	}

	if err := domain.ValidateReturnLoanInput(input); err != nil {
		return nil, err
	}

	loan, err := s.repo.GetLoan(ctx, loanID)
	if err != nil {
		return nil, err
	}
	if loan.OrganizationID != orgID {
		return nil, domain.ErrNotFound
	}

	if err := domain.ValidateLoanTransition(loan.Status, domain.LoanStatusReturned); err != nil {
		return nil, err
	}

	targetUnitStatus := "available"
	if input.ReturnCondition.RequiresMaintenance() {
		targetUnitStatus = "maintenance"
	}

	// CAS inventory unit: borrowed -> target
	_, err = s.inventoryClient.TransitionUnitStatus(ctx, loan.UnitID, "borrowed", targetUnitStatus)
	if err != nil {
		return nil, fmt.Errorf("transitioning unit to %s: %w", targetUnitStatus, err)
	}

	now := time.Now().UTC()
	oldStatus := loan.Status
	loan.Status = domain.LoanStatusReturned
	loan.ActualReturnTime = &now
	loan.ReturnCondition = &input.ReturnCondition
	loan.ReturnNotes = input.ReturnNotes
	loan.ProcessedByUserID = &adminID
	loan.UpdatedAt = now

	err = s.repo.RunInTx(ctx, func(tx pgx.Tx) error {
		if err := s.repo.UpdateLoan(ctx, tx, loan); err != nil {
			return err
		}
		if err := s.repo.UpdateReservationStatus(ctx, tx, loan.ReservationID, domain.ReservationStatusFulfilled); err != nil {
			return err
		}

		sh := domain.StatusHistory{
			ID:              uuid.New(),
			EntityType:      "loan",
			EntityID:        loan.ID,
			FromStatus:      string(oldStatus),
			ToStatus:        string(domain.LoanStatusReturned),
			ChangedByUserID: adminID,
			Reason:          input.ReturnNotes,
			CorrelationID:   correlationID,
			CreatedAt:       now,
		}
		if err := s.repo.CreateStatusHistory(ctx, tx, &sh); err != nil {
			return err
		}

		outboxEv := event.BuildOutboxEvent(
			event.EventLoanReturned,
			loan.ID.String(),
			orgID,
			correlationID,
			loan.ID.String(),
			map[string]any{
				"loan_id":          loan.ID.String(),
				"unit_id":          loan.UnitID.String(),
				"borrower_user_id": loan.BorrowerUserID.String(),
				"admin_user_id":    adminID.String(),
				"return_condition": string(input.ReturnCondition),
				"return_notes":     input.ReturnNotes,
				"actual_return":    now.Format(time.RFC3339Nano),
			},
		)
		return s.repo.CreateOutboxEvent(ctx, tx, &outboxEv)
	})

	if err != nil {
		if _, revErr := s.inventoryClient.TransitionUnitStatus(ctx, loan.UnitID, targetUnitStatus, "borrowed"); revErr != nil {
			slog.WarnContext(ctx, "failed to revert inventory unit status after return error", "unit_id", loan.UnitID, "error", revErr)
		}
		return nil, fmt.Errorf("returning loan: %w", err)
	}

	return loan, nil
}

// ---------------------------------------------------------------------------
// Overdue Handling
// ---------------------------------------------------------------------------

func (s *LendingService) ProcessOverdue(ctx context.Context) (int, error) {
	now := time.Now().UTC()
	candidates, err := s.repo.FindOverdueCheckedOutLoans(ctx, now)
	if err != nil {
		return 0, fmt.Errorf("finding overdue candidates: %w", err)
	}

	processed := 0
	for _, l := range candidates {
		transitioned, err := s.repo.TransitionLoanToOverdue(ctx, l.ID, now)
		if err != nil {
			return processed, fmt.Errorf("transitioning loan %s to overdue: %w", l.ID, err)
		}
		if transitioned {
			processed++
		}
	}

	return processed, nil
}

func (s *LendingService) ListOverdueLoans(
	ctx context.Context,
	orgID, userID uuid.UUID,
	role string,
) ([]domain.Loan, error) {
	if _, err := s.CheckUserMembership(ctx, orgID, userID); err != nil {
		return nil, err
	}
	if !domain.IsAdminOrOwner(role) {
		return nil, domain.ErrUnauthorizedAction
	}
	return s.repo.ListOverdueLoans(ctx, orgID)
}
