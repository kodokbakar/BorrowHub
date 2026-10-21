//go:build integration

package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"borrowhub/services/identity/internal/domain"
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

func TestRepository_FullIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Migrate Up
	if err := MigrateUp(databaseURL); err != nil {
		t.Fatalf("failed to run migrate up: %v", err)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to connect pgxpool: %v", err)
	}
	defer pool.Close()

	repos := NewRepositories(pool)

	// Clean tables before test
	_, _ = pool.Exec(ctx, "DELETE FROM sessions; DELETE FROM invitations; DELETE FROM memberships; DELETE FROM organizations; DELETE FROM users;")

	// 2. User Creation & Unique Constraint
	userID, err := uuid.NewV7()
	if err != nil {
		userID = uuid.New()
	}
	now := time.Now().Truncate(time.Microsecond)
	user1 := &domain.User{
		ID:              userID,
		Email:           "integration@example.com",
		PasswordHash:    "dummy-hash",
		DisplayName:     "Integration User",
		Status:          domain.UserStatusActive,
		TermsAcceptedAt: now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := repos.Users.Create(ctx, user1); err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	// Fetch user by ID and Email
	byID, err := repos.Users.GetByID(ctx, userID)
	if err != nil || byID.Email != "integration@example.com" {
		t.Fatalf("get by ID failed: %v, %+v", err, byID)
	}
	byEmail, err := repos.Users.GetByEmail(ctx, "INTEGRATION@example.com") // citext test
	if err != nil || byEmail.ID != userID {
		t.Fatalf("get by email (case-insensitive) failed: %v", err)
	}

	// Duplicate email test
	user2 := &domain.User{
		ID:              uuid.Must(uuid.NewV7()),
		Email:           "Integration@Example.Com",
		PasswordHash:    "dummy-hash2",
		DisplayName:     "Another User",
		Status:          domain.UserStatusActive,
		TermsAcceptedAt: now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	err = repos.Users.Create(ctx, user2)
	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}

	// 3. Organization & CreateWithInitialOwner
	orgID, err := uuid.NewV7()
	if err != nil {
		orgID = uuid.New()
	}
	memID, err := uuid.NewV7()
	if err != nil {
		memID = uuid.New()
	}
	org := &domain.Organization{
		ID:                  orgID,
		Name:                "Integration Org",
		Slug:                "integration-org",
		Timezone:            "UTC",
		Visibility:          domain.OrgVisibilityPrivate,
		MaxLoanDurationDays: 14,
		Status:              domain.OrgStatusActive,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	ownerMem := &domain.Membership{
		ID:             memID,
		OrganizationID: orgID,
		UserID:         userID,
		Role:           domain.RoleOwner,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repos.Organizations.CreateWithInitialOwner(ctx, org, ownerMem); err != nil {
		t.Fatalf("failed create org with initial owner: %v", err)
	}

	// Duplicate slug test
	dupOrg := &domain.Organization{
		ID:                  uuid.Must(uuid.NewV7()),
		Name:                "Duplicate Org",
		Slug:                "integration-org",
		Timezone:            "UTC",
		Visibility:          domain.OrgVisibilityPrivate,
		MaxLoanDurationDays: 14,
		Status:              domain.OrgStatusActive,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := repos.Organizations.Create(ctx, dupOrg); !errors.Is(err, domain.ErrSlugAlreadyExists) {
		t.Fatalf("expected ErrSlugAlreadyExists, got %v", err)
	}

	// 4. List memberships and count owners
	ownersCount, err := repos.Memberships.CountOwners(ctx, orgID)
	if err != nil || ownersCount != 1 {
		t.Fatalf("expected 1 owner, got %d (err: %v)", ownersCount, err)
	}

	members, err := repos.Memberships.ListByOrgID(ctx, orgID)
	if err != nil || len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}
	if members[0].Email != "integration@example.com" {
		t.Fatalf("expected joined email to match, got %s", members[0].Email)
	}

	// 5. Session creation & retrieval
	sessID, err := uuid.NewV7()
	if err != nil {
		sessID = uuid.New()
	}
	sess := &domain.Session{
		ID:               sessID,
		UserID:           userID,
		RefreshTokenHash: "hash-1234567890",
		UserAgent:        "test-agent",
		IPAddress:        "127.0.0.1",
		ExpiresAt:        now.Add(24 * time.Hour),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := repos.Sessions.Create(ctx, sess); err != nil {
		t.Fatalf("failed create session: %v", err)
	}
	fetchedSess, err := repos.Sessions.GetByRefreshTokenHash(ctx, "hash-1234567890")
	if err != nil || fetchedSess.ID != sessID {
		t.Fatalf("failed get session: %v", err)
	}

	// 6. Cascade Delete on Organization
	// Deleting org should cascade delete membership
	_, err = pool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgID)
	if err != nil {
		t.Fatalf("failed to delete org: %v", err)
	}
	_, err = repos.Memberships.GetByID(ctx, memID)
	if !errors.Is(err, domain.ErrMembershipNotFound) {
		t.Fatalf("expected membership to be cascade-deleted, got: %v", err)
	}

	// 7. Migrate Down then Up
	if err := MigrateDown(databaseURL); err != nil {
		t.Fatalf("failed to migrate down: %v", err)
	}
	if err := MigrateUp(databaseURL); err != nil {
		t.Fatalf("failed to migrate up again: %v", err)
	}
}
