package domain

import (
	"errors"
	"testing"
)

func TestValidation(t *testing.T) {
	// Email
	validEmail, err := ValidateEmail("  User@Example.COM ")
	if err != nil {
		t.Fatalf("unexpected error for valid email: %v", err)
	}
	if validEmail != "user@example.com" {
		t.Fatalf("expected normalized email user@example.com, got %s", validEmail)
	}

	invalidEmails := []string{"", "plainaddress", "@missinguser.com", "user@", "user@domain"}
	for _, em := range invalidEmails {
		if _, err := ValidateEmail(em); err == nil {
			t.Fatalf("expected error for invalid email %q", em)
		}
	}

	// Password
	if err := ValidatePassword("short"); err == nil {
		t.Fatal("expected error for short password")
	}
	if err := ValidatePassword("securePassword123"); err != nil {
		t.Fatalf("unexpected error for valid password: %v", err)
	}

	// Slug
	validSlugs := []string{"acme", "acme-corp", "team-123"}
	for _, s := range validSlugs {
		if err := ValidateSlug(s); err != nil {
			t.Fatalf("unexpected error for valid slug %q: %v", s, err)
		}
	}

	invalidSlugs := []string{"", "a", "Acme", "-acme", "acme-", "acme--corp", "acme_corp"}
	for _, s := range invalidSlugs {
		if err := ValidateSlug(s); err == nil {
			t.Fatalf("expected error for invalid slug %q", s)
		}
	}

	// Roles
	if !RoleOwner.IsValid() || !RoleAdmin.IsValid() || !RoleMember.IsValid() {
		t.Fatal("expected roles to be valid")
	}
	if Role("guest").IsValid() {
		t.Fatal("expected guest to be invalid role")
	}
	if !RoleOwner.CanManageMembers() || !RoleAdmin.CanManageMembers() || RoleMember.CanManageMembers() {
		t.Fatal("expected correct member management permissions")
	}

	if err := ValidateInvitationRole(RoleOwner); err == nil {
		t.Fatal("expected owner to be invalid invitation role")
	}
	if err := ValidateInvitationRole(RoleAdmin); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateInvitationRole(RoleMember); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Names & Duration
	if err := ValidateDisplayName(""); err == nil {
		t.Fatal("expected error for empty display name")
	}
	if err := ValidateOrgName(""); err == nil {
		t.Fatal("expected error for empty org name")
	}
	if err := ValidateMaxLoanDuration(0); err == nil {
		t.Fatal("expected error for 0 max loan duration")
	}
	if err := ValidateMaxLoanDuration(14); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var valErr *ValidationError
	if !errors.As(NewValidationError("msg", nil), &valErr) {
		t.Fatal("expected errors.As to match ValidationError")
	}
}
