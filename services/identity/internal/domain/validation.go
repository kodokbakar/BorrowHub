package domain

import (
	"net/mail"
	"regexp"
	"strings"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func ValidateEmail(email string) (string, error) {
	norm := NormalizeEmail(email)
	if norm == "" {
		return "", NewValidationError("email is required", map[string]any{"email": "cannot be empty"})
	}
	addr, err := mail.ParseAddress(norm)
	if err != nil || addr.Address != norm || !strings.Contains(norm, ".") {
		return "", NewValidationError("invalid email format", map[string]any{"email": "must be a valid email address"})
	}
	return norm, nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return NewValidationError("password must be at least 8 characters long", map[string]any{"password": "minimum 8 characters"})
	}
	return nil
}

func ValidateSlug(slug string) error {
	trimmed := strings.TrimSpace(slug)
	if len(trimmed) < 2 || len(trimmed) > 50 {
		return NewValidationError("slug must be between 2 and 50 characters", map[string]any{"slug": "length must be between 2 and 50"})
	}
	if !slugRegex.MatchString(trimmed) {
		return NewValidationError("slug must consist of lowercase alphanumeric characters and single hyphens", map[string]any{"slug": "invalid format"})
	}
	return nil
}

func ValidateDisplayName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return NewValidationError("display name is required", map[string]any{"display_name": "cannot be empty"})
	}
	if len(trimmed) > 100 {
		return NewValidationError("display name cannot exceed 100 characters", map[string]any{"display_name": "maximum 100 characters"})
	}
	return nil
}

func ValidateOrgName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return NewValidationError("organization name is required", map[string]any{"name": "cannot be empty"})
	}
	if len(trimmed) > 100 {
		return NewValidationError("organization name cannot exceed 100 characters", map[string]any{"name": "maximum 100 characters"})
	}
	return nil
}

func ValidateRole(role Role) error {
	if !role.IsValid() {
		return NewValidationError("invalid role", map[string]any{"role": "must be owner, admin, or member"})
	}
	return nil
}

func ValidateInvitationRole(role Role) error {
	if role != RoleAdmin && role != RoleMember {
		return NewValidationError("invalid invitation role", map[string]any{"role": "must be admin or member"})
	}
	return nil
}

func ValidateMaxLoanDuration(days int) error {
	if days <= 0 {
		return NewValidationError("max loan duration must be positive", map[string]any{"max_loan_duration_days": "must be greater than 0"})
	}
	return nil
}
