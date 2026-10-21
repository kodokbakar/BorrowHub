package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"borrowhub/services/identity/internal/auth"
	"borrowhub/services/identity/internal/domain"
	"github.com/google/uuid"
)

const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 30 * 24 * time.Hour
)

type AuthResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type RegisterInput struct {
	Email         string
	Password      string
	DisplayName   string
	TermsAccepted bool
	UserAgent     string
	IPAddress     string
}

type LoginInput struct {
	Email     string
	Password  string
	UserAgent string
	IPAddress string
}

type UpdateMeInput struct {
	DisplayName     *string
	CurrentPassword *string
	NewPassword     *string
}

type AuthService struct {
	users    UserRepository
	orgs     OrganizationRepository
	sessions SessionRepository
	hasher   PasswordHasher
	tokens   TokenManager
}

func NewAuthService(
	users UserRepository,
	orgs OrganizationRepository,
	sessions SessionRepository,
	hasher PasswordHasher,
	tokens TokenManager,
) *AuthService {
	return &AuthService{
		users:    users,
		orgs:     orgs,
		sessions: sessions,
		hasher:   hasher,
		tokens:   tokens,
	}
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*domain.User, *AuthResult, error) {
	if !input.TermsAccepted {
		return nil, nil, domain.ErrTermsNotAccepted
	}

	normEmail, err := domain.ValidateEmail(input.Email)
	if err != nil {
		return nil, nil, err
	}

	if err := domain.ValidatePassword(input.Password); err != nil {
		return nil, nil, err
	}

	if err := domain.ValidateDisplayName(input.DisplayName); err != nil {
		return nil, nil, err
	}

	hash, err := s.hasher.HashPassword(input.Password)
	if err != nil {
		return nil, nil, err
	}

	userID, err := uuid.NewV7()
	if err != nil {
		userID = uuid.New()
	}

	now := time.Now()
	user := &domain.User{
		ID:              userID,
		Email:           normEmail,
		PasswordHash:    hash,
		DisplayName:     strings.TrimSpace(input.DisplayName),
		Status:          domain.UserStatusActive,
		TermsAcceptedAt: now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	authResult, err := s.issueTokensAndSession(ctx, user, input.UserAgent, input.IPAddress)
	if err != nil {
		return nil, nil, err
	}

	return user, authResult, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (*domain.User, *AuthResult, error) {
	normEmail, err := domain.ValidateEmail(input.Email)
	if err != nil {
		s.hasher.DummyVerify(input.Password)
		return nil, nil, domain.ErrInvalidCredentials
	}

	user, err := s.users.GetByEmail(ctx, normEmail)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrUserNotFound) {
			s.hasher.DummyVerify(input.Password)
			return nil, nil, domain.ErrInvalidCredentials
		}
		return nil, nil, err
	}

	if user.Status != domain.UserStatusActive {
		s.hasher.DummyVerify(input.Password)
		return nil, nil, domain.ErrInvalidCredentials
	}

	valid, err := s.hasher.VerifyPassword(input.Password, user.PasswordHash)
	if err != nil || !valid {
		return nil, nil, domain.ErrInvalidCredentials
	}

	authResult, err := s.issueTokensAndSession(ctx, user, input.UserAgent, input.IPAddress)
	if err != nil {
		return nil, nil, err
	}

	return user, authResult, nil
}

func (s *AuthService) Refresh(
	ctx context.Context,
	rawRefreshToken string,
	userAgent string,
	ipAddress string,
) (*AuthResult, error) {
	if strings.TrimSpace(rawRefreshToken) == "" {
		return nil, domain.ErrUnauthorized
	}

	hash := auth.HashToken(rawRefreshToken)
	session, err := s.sessions.GetByRefreshTokenHash(ctx, hash)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	now := time.Now()
	if session.RevokedAt != nil {
		_ = s.sessions.RevokeAllForUser(ctx, session.UserID, now)
		return nil, domain.ErrUnauthorized
	}
	if now.After(session.ExpiresAt) {
		return nil, domain.ErrUnauthorized
	}

	revoked, err := s.sessions.Revoke(ctx, session.ID, now)
	if err != nil {
		return nil, err
	}
	if !revoked {
		_ = s.sessions.RevokeAllForUser(ctx, session.UserID, now)
		return nil, domain.ErrUnauthorized
	}
	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil || user.Status != domain.UserStatusActive {
		return nil, domain.ErrUnauthorized
	}

	return s.issueTokensAndSession(ctx, user, userAgent, ipAddress)
}

func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	if strings.TrimSpace(rawRefreshToken) == "" {
		return nil
	}

	hash := auth.HashToken(rawRefreshToken)
	session, err := s.sessions.GetByRefreshTokenHash(ctx, hash)
	if err != nil {
		return nil
	}

	_, err = s.sessions.Revoke(ctx, session.ID, time.Now())
	return err
}

func (s *AuthService) GetMe(ctx context.Context, userID uuid.UUID) (*domain.User, []domain.UserOrganization, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	orgs, err := s.orgs.ListByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	return user, orgs, nil
}

func (s *AuthService) UpdateMe(ctx context.Context, userID uuid.UUID, input UpdateMeInput) (*domain.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if input.DisplayName != nil {
		if err := domain.ValidateDisplayName(*input.DisplayName); err != nil {
			return nil, err
		}
		user.DisplayName = strings.TrimSpace(*input.DisplayName)
	}

	if input.NewPassword != nil {
		if input.CurrentPassword == nil || *input.CurrentPassword == "" {
			return nil, domain.NewValidationError("current password is required to set new password", map[string]any{
				"current_password": "is required",
			})
		}

		valid, err := s.hasher.VerifyPassword(*input.CurrentPassword, user.PasswordHash)
		if err != nil || !valid {
			return nil, domain.ErrInvalidCredentials
		}

		if err := domain.ValidatePassword(*input.NewPassword); err != nil {
			return nil, err
		}

		newHash, err := s.hasher.HashPassword(*input.NewPassword)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = newHash

		if err := s.sessions.RevokeAllForUser(ctx, userID, time.Now()); err != nil {
			return nil, err
		}
	}

	user.UpdatedAt = time.Now()
	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) issueTokensAndSession(
	ctx context.Context,
	user *domain.User,
	userAgent string,
	ipAddress string,
) (*AuthResult, error) {
	accessToken, err := s.tokens.GenerateAccessToken(user.ID.String(), user.Email, AccessTokenTTL)
	if err != nil {
		return nil, err
	}

	rawRefreshToken, err := auth.GenerateSecureToken(32)
	if err != nil {
		return nil, err
	}

	tokenHash := auth.HashToken(rawRefreshToken)
	sessionID, err := uuid.NewV7()
	if err != nil {
		sessionID = uuid.New()
	}

	now := time.Now()
	session := &domain.Session{
		ID:               sessionID,
		UserID:           user.ID,
		RefreshTokenHash: tokenHash,
		UserAgent:        userAgent,
		IPAddress:        ipAddress,
		ExpiresAt:        now.Add(RefreshTokenTTL),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(AccessTokenTTL.Seconds()),
	}, nil
}
