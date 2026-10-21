//go:build integration

package repository

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"borrowhub/services/lending/internal/domain"
	"borrowhub/services/lending/internal/event"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresConnection(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	expectedDatabase := os.Getenv("EXPECTED_DATABASE")

	if databaseURL == "" || expectedDatabase == "" {
		t.Skip("DATABASE_URL and EXPECTED_DATABASE are required for integration tests")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())

	var database string

	if err := conn.QueryRow(
		ctx,
		"SELECT current_database()",
	).Scan(&database); err != nil {
		t.Fatal(err)
	}

	if database != expectedDatabase {
		t.Fatalf(
			"connected to %q, expected %q",
			database,
			expectedDatabase,
		)
	}
}

func TestLendingRepository_Integration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Migrations Up / Down Idempotency
	if err := MigrateUp(databaseURL); err != nil {
		t.Fatalf("failed migrate up: %v", err)
	}
	if err := MigrateDown(databaseURL); err != nil {
		t.Fatalf("failed migrate down: %v", err)
	}
	if err := MigrateUp(databaseURL); err != nil {
		t.Fatalf("failed re-run migrate up: %v", err)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed connecting to pgxpool: %v", err)
	}
	defer pool.Close()

	repo := NewRepository(pool)

	orgID := uuid.New()
	userID := uuid.New()
	adminID := uuid.New()
	itemID := uuid.New()
	unitID := uuid.New()

	baseTime := time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)

	// 2. Loan Request Creation & Retrieval
	req1 := &domain.LoanRequest{
		ID:              uuid.New(),
		OrganizationID:  orgID,
		RequesterUserID: userID,
		ItemID:          itemID,
		Quantity:        1,
		StartTime:       baseTime,
		EndTime:         baseTime.Add(4 * time.Hour),
		Purpose:         "Geological survey",
		Status:          domain.RequestStatusPending,
		CreatedAt:       baseTime,
		UpdatedAt:       baseTime,
	}
	if err := repo.CreateLoanRequest(ctx, nil, req1); err != nil {
		t.Fatalf("create loan request 1: %v", err)
	}

	fetchedReq, err := repo.GetLoanRequest(ctx, req1.ID)
	if err != nil {
		t.Fatalf("get loan request: %v", err)
	}
	if fetchedReq.Purpose != req1.Purpose || fetchedReq.Status != domain.RequestStatusPending {
		t.Fatalf("unexpected fetched loan request: %+v", fetchedReq)
	}

	// 3. PostgreSQL Exclusion Constraint on Reservations
	// R1: [10:00, 14:00) active
	res1 := &domain.Reservation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		LoanRequestID:  req1.ID,
		UnitID:         unitID,
		StartTime:      baseTime,
		EndTime:        baseTime.Add(4 * time.Hour),
		Status:         domain.ReservationStatusActive,
		CreatedAt:      baseTime,
		UpdatedAt:      baseTime,
	}
	if err := repo.CreateReservation(ctx, nil, res1); err != nil {
		t.Fatalf("create reservation 1: %v", err)
	}

	// R2: Overlaps with R1 on same unit: [12:00, 16:00)
	req2 := &domain.LoanRequest{
		ID:              uuid.New(),
		OrganizationID:  orgID,
		RequesterUserID: userID,
		ItemID:          itemID,
		Quantity:        1,
		StartTime:       baseTime.Add(2 * time.Hour),
		EndTime:         baseTime.Add(6 * time.Hour),
		Purpose:         "Overlapping request",
		Status:          domain.RequestStatusPending,
		CreatedAt:       baseTime,
		UpdatedAt:       baseTime,
	}
	if err := repo.CreateLoanRequest(ctx, nil, req2); err != nil {
		t.Fatalf("create loan request 2: %v", err)
	}

	res2 := &domain.Reservation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		LoanRequestID:  req2.ID,
		UnitID:         unitID,
		StartTime:      baseTime.Add(2 * time.Hour),
		EndTime:        baseTime.Add(6 * time.Hour),
		Status:         domain.ReservationStatusActive,
		CreatedAt:      baseTime,
		UpdatedAt:      baseTime,
	}
	err = repo.CreateReservation(ctx, nil, res2)
	if !errors.Is(err, domain.ErrReservationConflict) {
		t.Fatalf("expected ErrReservationConflict (23P01) for overlapping reservation, got: %v", err)
	}

	// Abutting reservation R3: [14:00, 18:00) on same unit should SUCCEED
	req3 := &domain.LoanRequest{
		ID:              uuid.New(),
		OrganizationID:  orgID,
		RequesterUserID: userID,
		ItemID:          itemID,
		Quantity:        1,
		StartTime:       baseTime.Add(4 * time.Hour),
		EndTime:         baseTime.Add(8 * time.Hour),
		Purpose:         "Abutting request",
		Status:          domain.RequestStatusPending,
		CreatedAt:       baseTime,
		UpdatedAt:       baseTime,
	}
	if err := repo.CreateLoanRequest(ctx, nil, req3); err != nil {
		t.Fatalf("create loan request 3: %v", err)
	}

	res3 := &domain.Reservation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		LoanRequestID:  req3.ID,
		UnitID:         unitID,
		StartTime:      baseTime.Add(4 * time.Hour),
		EndTime:        baseTime.Add(8 * time.Hour),
		Status:         domain.ReservationStatusActive,
		CreatedAt:      baseTime,
		UpdatedAt:      baseTime,
	}
	if err := repo.CreateReservation(ctx, nil, res3); err != nil {
		t.Fatalf("abutting reservation should succeed, got: %v", err)
	}

	// 4. Concurrent Approval Race Condition Guarded by PostgreSQL Exclusion Constraint
	unitRace := uuid.New()
	var wg sync.WaitGroup
	results := make([]error, 2)

	for i := range 2 {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			raceReq := &domain.LoanRequest{
				ID:              uuid.New(),
				OrganizationID:  orgID,
				RequesterUserID: userID,
				ItemID:          itemID,
				Quantity:        1,
				StartTime:       baseTime,
				EndTime:         baseTime.Add(2 * time.Hour),
				Purpose:         "Race request",
				Status:          domain.RequestStatusPending,
				CreatedAt:       baseTime,
				UpdatedAt:       baseTime,
			}
			_ = repo.CreateLoanRequest(ctx, nil, raceReq)

			raceRes := &domain.Reservation{
				ID:             uuid.New(),
				OrganizationID: orgID,
				LoanRequestID:  raceReq.ID,
				UnitID:         unitRace,
				StartTime:      baseTime,
				EndTime:        baseTime.Add(2 * time.Hour),
				Status:         domain.ReservationStatusActive,
				CreatedAt:      baseTime,
				UpdatedAt:      baseTime,
			}
			results[idx] = repo.CreateReservation(ctx, nil, raceRes)
		}()
	}
	wg.Wait()

	// Exactly one must succeed, and one must fail with ErrReservationConflict
	successes := 0
	conflicts := 0
	for _, resErr := range results {
		if resErr == nil {
			successes++
		} else if errors.Is(resErr, domain.ErrReservationConflict) {
			conflicts++
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("expected 1 success and 1 conflict in concurrent race, got %d successes and %d conflicts (errors: %v, %v)",
			successes, conflicts, results[0], results[1])
	}

	// 5. Transactional Outbox Pattern
	// Successful transaction
	loanID := uuid.New()
	txErr := repo.RunInTx(ctx, func(tx pgx.Tx) error {
		req := &domain.LoanRequest{
			ID:              uuid.New(),
			OrganizationID:  orgID,
			RequesterUserID: userID,
			ItemID:          itemID,
			Quantity:        1,
			StartTime:       baseTime.Add(24 * time.Hour),
			EndTime:         baseTime.Add(28 * time.Hour),
			Purpose:         "Outbox transaction test",
			Status:          domain.RequestStatusApproved,
			CreatedAt:       baseTime,
			UpdatedAt:       baseTime,
		}
		if err := repo.CreateLoanRequest(ctx, tx, req); err != nil {
			return err
		}

		res := &domain.Reservation{
			ID:             uuid.New(),
			OrganizationID: orgID,
			LoanRequestID:  req.ID,
			UnitID:         unitID,
			StartTime:      req.StartTime,
			EndTime:        req.EndTime,
			Status:         domain.ReservationStatusActive,
			CreatedAt:      baseTime,
			UpdatedAt:      baseTime,
		}
		if err := repo.CreateReservation(ctx, tx, res); err != nil {
			return err
		}

		loan := &domain.Loan{
			ID:                 loanID,
			OrganizationID:     orgID,
			LoanRequestID:      req.ID,
			ReservationID:      res.ID,
			UnitID:             unitID,
			BorrowerUserID:     userID,
			Status:             domain.LoanStatusPendingCheckout,
			ExpectedReturnTime: req.EndTime,
			CreatedAt:          baseTime,
			UpdatedAt:          baseTime,
		}
		if err := repo.CreateLoan(ctx, tx, loan); err != nil {
			return err
		}

		ev := event.BuildOutboxEvent(
			event.EventLoanApproved,
			loan.ID.String(),
			orgID,
			"corr-tx-1",
			req.ID.String(),
			map[string]any{"loan_id": loan.ID.String()},
		)
		return repo.CreateOutboxEvent(ctx, tx, &ev)
	})
	if txErr != nil {
		t.Fatalf("transaction commit failed: %v", txErr)
	}

	// Verify outbox event persisted
	pendingEvents, err := repo.GetPendingOutboxEvents(ctx, 10)
	if err != nil {
		t.Fatalf("get pending outbox: %v", err)
	}
	if len(pendingEvents) == 0 {
		t.Fatal("expected at least 1 pending outbox event")
	}

	// Mark event published
	targetEventID := pendingEvents[0].ID
	if err := repo.MarkOutboxEventPublished(ctx, targetEventID); err != nil {
		t.Fatalf("mark published failed: %v", err)
	}

	// Transaction Rollback Atomicity
	rollbackErr := repo.RunInTx(ctx, func(tx pgx.Tx) error {
		failReq := &domain.LoanRequest{
			ID:              uuid.New(),
			OrganizationID:  orgID,
			RequesterUserID: userID,
			ItemID:          itemID,
			Quantity:        1,
			StartTime:       baseTime.Add(48 * time.Hour),
			EndTime:         baseTime.Add(52 * time.Hour),
			Purpose:         "Rollback test",
			Status:          domain.RequestStatusPending,
			CreatedAt:       baseTime,
			UpdatedAt:       baseTime,
		}
		_ = repo.CreateLoanRequest(ctx, tx, failReq)
		ev := event.BuildOutboxEvent(
			event.EventLoanRequested,
			failReq.ID.String(),
			orgID,
			"corr-fail",
			failReq.ID.String(),
			map[string]any{"fail": true},
		)
		_ = repo.CreateOutboxEvent(ctx, tx, &ev)

		// Intentional rollback trigger
		return errors.New("simulated error to trigger rollback")
	})
	if rollbackErr == nil {
		t.Fatal("expected rollback error")
	}

	// 6. Checkout Token Lifecycle
	_, tokenHash, err := domain.GenerateSecureToken()
	if err != nil {
		t.Fatalf("generating secure token: %v", err)
	}
	tokenID := uuid.New()
	token := &domain.CheckoutToken{
		ID:        tokenID,
		LoanID:    loanID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	if err := repo.CreateCheckoutToken(ctx, nil, token); err != nil {
		t.Fatalf("create checkout token: %v", err)
	}

	fetchedToken, err := repo.GetCheckoutTokenByHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("get checkout token: %v", err)
	}
	if fetchedToken.ID != tokenID {
		t.Fatalf("expected token ID %s, got %s", tokenID, fetchedToken.ID)
	}

	// Mark used
	if err := repo.MarkCheckoutTokenUsed(ctx, nil, tokenID, time.Now()); err != nil {
		t.Fatalf("mark token used: %v", err)
	}

	// Second use fails with ErrTokenAlreadyUsed
	err = repo.MarkCheckoutTokenUsed(ctx, nil, tokenID, time.Now())
	if !errors.Is(err, domain.ErrTokenAlreadyUsed) {
		t.Fatalf("expected ErrTokenAlreadyUsed on second use, got: %v", err)
	}

	// 7. Status History
	sh := &domain.StatusHistory{
		ID:              uuid.New(),
		EntityType:      "loan",
		EntityID:        loanID,
		FromStatus:      "pending_checkout",
		ToStatus:        "checked_out",
		ChangedByUserID: adminID,
		Reason:          nil,
		CorrelationID:   "corr-sh-1",
		CreatedAt:       time.Now(),
	}
	if err := repo.CreateStatusHistory(ctx, nil, sh); err != nil {
		t.Fatalf("create status history: %v", err)
	}
	historyList, err := repo.GetStatusHistory(ctx, "loan", loanID)
	if err != nil {
		t.Fatalf("get status history: %v", err)
	}
	if len(historyList) == 0 {
		t.Fatal("expected at least 1 status history record")
	}

	// 8. Idempotency Key Storage & Lookup
	idempKey := "test-idemp-" + uuid.New().String()
	idempRec := &domain.IdempotencyRecord{
		Key:            idempKey,
		OrganizationID: orgID,
		UserID:         adminID,
		RequestPath:    "/api/v1/loan-requests",
		RequestHash:    "hash123",
		ResponseCode:   201,
		ResponseBody:   []byte(`{"status":"created"}`),
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}
	if err := repo.SaveIdempotencyKey(ctx, idempRec); err != nil {
		t.Fatalf("save idempotency key: %v", err)
	}

	fetchedIdemp, err := repo.GetIdempotencyKey(ctx, orgID, idempKey)
	if err != nil {
		t.Fatalf("get idempotency key: %v", err)
	}
	if fetchedIdemp == nil || fetchedIdemp.RequestHash != "hash123" {
		t.Fatalf("unexpected fetched idempotency record: %+v", fetchedIdemp)
	}
	// 9. ListLoans with Borrower Filter
	userLoans, err := repo.ListLoans(ctx, orgID, &userID, nil, 10, 0)
	if err != nil {
		t.Fatalf("list loans for user: %v", err)
	}
	if len(userLoans) == 0 {
		t.Fatal("expected at least 1 loan for user")
	}
	otherUser := uuid.New()
	emptyLoans, err := repo.ListLoans(ctx, orgID, &otherUser, nil, 10, 0)
	if err != nil {
		t.Fatalf("list loans for other user: %v", err)
	}
	if len(emptyLoans) != 0 {
		t.Fatalf("expected 0 loans for other user, got %d", len(emptyLoans))
	}

	// 10. TransitionLoanToOverdue Atomic CAS & Concurrency
	overdueLoanID := uuid.New()
	overdueLoan := &domain.Loan{
		ID:                 overdueLoanID,
		OrganizationID:     orgID,
		LoanRequestID:      req1.ID,
		ReservationID:      res1.ID,
		UnitID:             unitID,
		BorrowerUserID:     userID,
		Status:             domain.LoanStatusCheckedOut,
		CheckoutTime:       &baseTime,
		ExpectedReturnTime: baseTime.Add(time.Hour),
		CreatedAt:          baseTime,
		UpdatedAt:          baseTime,
	}
	if err := repo.CreateLoan(ctx, nil, overdueLoan); err != nil {
		t.Fatalf("create overdue loan test: %v", err)
	}

	// Concurrent CAS test
	const numWorkers = 5
	var casWg sync.WaitGroup
	resultsCh := make(chan bool, numWorkers)
	for range numWorkers {
		casWg.Add(1)
		go func() {
			defer casWg.Done()
			success, cErr := repo.TransitionLoanToOverdue(ctx, overdueLoanID, time.Now())
			if cErr != nil {
				t.Errorf("concurrent TransitionLoanToOverdue error: %v", cErr)
				return
			}
			resultsCh <- success
		}()
	}
	casWg.Wait()
	close(resultsCh)

	trueCount := 0
	for s := range resultsCh {
		if s {
			trueCount++
		}
	}
	if trueCount != 1 {
		t.Fatalf("expected exactly 1 worker to transition to overdue, got %d", trueCount)
	}

	// Verify loan status in DB is overdue
	dbLoan, err := repo.GetLoan(ctx, overdueLoanID)
	if err != nil {
		t.Fatalf("get overdue loan: %v", err)
	}
	if dbLoan.Status != domain.LoanStatusOverdue {
		t.Fatalf("expected overdue status, got %s", dbLoan.Status)
	}

	// 11. ClaimPendingOutboxEvents Transactional Claiming
	// Insert pending events
	ev1 := event.BuildOutboxEvent(event.EventLoanRequested, "subj-1", orgID, "corr-1", "cause-1", map[string]any{"test": 1})
	ev2 := event.BuildOutboxEvent(event.EventLoanRequested, "subj-2", orgID, "corr-2", "cause-2", map[string]any{"test": 2})
	if err := repo.CreateOutboxEvent(ctx, nil, &ev1); err != nil {
		t.Fatalf("create outbox ev1: %v", err)
	}
	if err := repo.CreateOutboxEvent(ctx, nil, &ev2); err != nil {
		t.Fatalf("create outbox ev2: %v", err)
	}

	claimed, err := repo.ClaimPendingOutboxEvents(ctx, 10)
	if err != nil {
		t.Fatalf("claim pending outbox events: %v", err)
	}
	if len(claimed) == 0 {
		t.Fatal("expected at least 1 claimed event")
	}
	for _, c := range claimed {
		if c.Status != "in_flight" {
			t.Fatalf("expected claimed event status in_flight, got %s", c.Status)
		}
	}

	// Second claim should return no more pending events
	claimedSecond, err := repo.ClaimPendingOutboxEvents(ctx, 10)
	if err != nil {
		t.Fatalf("second claim outbox: %v", err)
	}
	for _, c := range claimedSecond {
		if c.ID == ev1.ID || c.ID == ev2.ID {
			t.Fatalf("already claimed event %d returned again", c.ID)
		}
	}

	// Mark failed should set back to pending
	if err := repo.MarkOutboxEventFailed(ctx, ev1.ID, "temporary broker network glitch"); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	reclaimed, err := repo.ClaimPendingOutboxEvents(ctx, 10)
	if err != nil {
		t.Fatalf("reclaim outbox: %v", err)
	}
	foundReclaimed := false
	for _, c := range reclaimed {
		if c.ID == ev1.ID {
			foundReclaimed = true
			break
		}
	}
	if !foundReclaimed {
		t.Fatal("expected failed event to be reclaimed after being set back to pending")
	}
}
