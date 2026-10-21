package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"borrowhub/services/identity/internal/auth"
	"borrowhub/services/identity/internal/domain"
	"borrowhub/services/identity/internal/service"
	"github.com/google/uuid"
)

type RouterDependencies struct {
	AuthService   *service.AuthService
	OrgService    *service.OrgService
	MemberService *service.MemberService
	TokenManager  *auth.JWTManager
	ReadyCheck    func() bool
	IsSecure      bool
	InternalToken string
}

type handler struct {
	deps RouterDependencies
}

func NewRouter(deps RouterDependencies) http.Handler {
	h := &handler{deps: deps}
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health/live", h.handleHealthLive)
	mux.HandleFunc("GET /health/ready", h.handleHealthReady)

	// Public Auth
	mux.HandleFunc("POST /api/v1/auth/register", h.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", h.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", h.handleRefresh)
	mux.HandleFunc("POST /api/v1/auth/logout", h.handleLogout)

	// Internal downstream verification
	mux.HandleFunc("GET /api/v1/internal/organizations/{organizationId}/members/{userId}", h.handleInternalGetMember)

	// Protected routes
	authWrap := AuthMiddleware(deps.TokenManager)

	mux.Handle("GET /api/v1/me", authWrap(http.HandlerFunc(h.handleGetMe)))
	mux.Handle("PATCH /api/v1/me", authWrap(http.HandlerFunc(h.handleUpdateMe)))

	mux.Handle("POST /api/v1/organizations", authWrap(http.HandlerFunc(h.handleCreateOrg)))
	mux.Handle("GET /api/v1/organizations", authWrap(http.HandlerFunc(h.handleListOrgs)))
	mux.Handle("GET /api/v1/organizations/{organizationId}", authWrap(http.HandlerFunc(h.handleGetOrg)))
	mux.Handle("PATCH /api/v1/organizations/{organizationId}", authWrap(http.HandlerFunc(h.handleUpdateOrg)))

	mux.Handle("GET /api/v1/organizations/{organizationId}/members", authWrap(http.HandlerFunc(h.handleListMembers)))
	mux.Handle("POST /api/v1/organizations/{organizationId}/invitations", authWrap(http.HandlerFunc(h.handleInviteMember)))
	mux.Handle("POST /api/v1/invitations/{token}/accept", authWrap(http.HandlerFunc(h.handleAcceptInvitation)))

	mux.Handle("PATCH /api/v1/organizations/{organizationId}/members/{memberId}", authWrap(http.HandlerFunc(h.handleChangeMemberRole)))
	mux.Handle("DELETE /api/v1/organizations/{organizationId}/members/{memberId}", authWrap(http.HandlerFunc(h.handleRemoveMember)))

	// Fallback 404
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, r, http.StatusNotFound, "RESOURCE_NOT_FOUND", "route not found", nil)
	})

	return mux
}

func (h *handler) handleHealthLive(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) handleHealthReady(w http.ResponseWriter, r *http.Request) {
	if h.deps.ReadyCheck != nil && !h.deps.ReadyCheck() {
		WriteError(
			w,
			r,
			http.StatusServiceUnavailable,
			"SERVICE_NOT_READY",
			"service is not ready",
			nil,
		)
		return
	}
	WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}

type registerRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	DisplayName   string `json:"display_name"`
	TermsAccepted bool   `json:"terms_accepted"`
}

func (h *handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(w, r, &req); err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil)
		return
	}

	user, tokens, err := h.deps.AuthService.Register(r.Context(), service.RegisterInput{
		Email:         req.Email,
		Password:      req.Password,
		DisplayName:   req.DisplayName,
		TermsAccepted: req.TermsAccepted,
		UserAgent:     r.UserAgent(),
		IPAddress:     r.RemoteAddr,
	})
	if err != nil {
		HandleError(w, r, err)
		return
	}

	SetAuthCookies(w, tokens.AccessToken, tokens.RefreshToken, h.deps.IsSecure)

	WriteJSON(w, r, http.StatusCreated, map[string]any{
		"user":   user,
		"tokens": tokens,
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil)
		return
	}

	user, tokens, err := h.deps.AuthService.Login(r.Context(), service.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: r.UserAgent(),
		IPAddress: r.RemoteAddr,
	})
	if err != nil {
		HandleError(w, r, err)
		return
	}

	SetAuthCookies(w, tokens.AccessToken, tokens.RefreshToken, h.deps.IsSecure)

	WriteJSON(w, r, http.StatusOK, map[string]any{
		"user":   user,
		"tokens": tokens,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *handler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var rawToken string

	var req refreshRequest
	if err := decodeJSON(w, r, &req); err == nil && req.RefreshToken != "" {
		rawToken = req.RefreshToken
	} else if cookie, err := r.Cookie("refresh_token"); err == nil {
		rawToken = cookie.Value
	}

	if rawToken == "" {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Missing refresh token", nil)
		return
	}

	tokens, err := h.deps.AuthService.Refresh(r.Context(), rawToken, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	SetAuthCookies(w, tokens.AccessToken, tokens.RefreshToken, h.deps.IsSecure)

	WriteJSON(w, r, http.StatusOK, tokens)
}

func (h *handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	var rawToken string

	var req refreshRequest
	if err := decodeJSON(w, r, &req); err == nil && req.RefreshToken != "" {
		rawToken = req.RefreshToken
	} else if cookie, err := r.Cookie("refresh_token"); err == nil {
		rawToken = cookie.Value
	}

	_ = h.deps.AuthService.Logout(r.Context(), rawToken)
	ClearAuthCookies(w, h.deps.IsSecure)

	WriteJSON(w, r, http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}

func (h *handler) handleGetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	user, orgs, err := h.deps.AuthService.GetMe(r.Context(), userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, map[string]any{
		"user":          user,
		"organizations": orgs,
	})
}

type updateMeRequest struct {
	DisplayName     *string `json:"display_name"`
	CurrentPassword *string `json:"current_password"`
	NewPassword     *string `json:"new_password"`
}

func (h *handler) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	var req updateMeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil)
		return
	}

	user, err := h.deps.AuthService.UpdateMe(r.Context(), userID, service.UpdateMeInput{
		DisplayName:     req.DisplayName,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, user)
}

type createOrgRequest struct {
	Name                string `json:"name"`
	Slug                string `json:"slug"`
	Timezone            string `json:"timezone"`
	Visibility          string `json:"visibility"`
	MaxLoanDurationDays int    `json:"max_loan_duration_days"`
}

func (h *handler) handleCreateOrg(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	var req createOrgRequest
	if err := decodeJSON(w, r, &req); err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil)
		return
	}

	org, err := h.deps.OrgService.Create(r.Context(), userID, service.CreateOrgInput{
		Name:                req.Name,
		Slug:                req.Slug,
		Timezone:            req.Timezone,
		Visibility:          req.Visibility,
		MaxLoanDurationDays: req.MaxLoanDurationDays,
	})
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusCreated, org)
}

func (h *handler) handleListOrgs(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	orgs, err := h.deps.OrgService.ListUserOrgs(r.Context(), userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	if orgs == nil {
		orgs = []domain.UserOrganization{}
	}

	WriteJSON(w, r, http.StatusOK, orgs)
}

func (h *handler) handleGetOrg(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	orgID, err := parseUUID("organizationId", r.PathValue("organizationId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	org, err := h.deps.OrgService.Get(r.Context(), orgID, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, org)
}

type updateOrgRequest struct {
	Name                *string `json:"name"`
	Timezone            *string `json:"timezone"`
	Visibility          *string `json:"visibility"`
	MaxLoanDurationDays *int    `json:"max_loan_duration_days"`
}

func (h *handler) handleUpdateOrg(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	orgID, err := parseUUID("organizationId", r.PathValue("organizationId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	var req updateOrgRequest
	if err := decodeJSON(w, r, &req); err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil)
		return
	}

	org, err := h.deps.OrgService.Update(r.Context(), orgID, userID, service.UpdateOrgInput{
		Name:                req.Name,
		Timezone:            req.Timezone,
		Visibility:          req.Visibility,
		MaxLoanDurationDays: req.MaxLoanDurationDays,
	})
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, org)
}

func (h *handler) handleListMembers(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	orgID, err := parseUUID("organizationId", r.PathValue("organizationId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	members, err := h.deps.MemberService.ListMembers(r.Context(), orgID, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	if members == nil {
		members = []domain.MembershipDetail{}
	}

	WriteJSON(w, r, http.StatusOK, members)
}

type inviteMemberRequest struct {
	Email string      `json:"email"`
	Role  domain.Role `json:"role"`
}

func (h *handler) handleInviteMember(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	orgID, err := parseUUID("organizationId", r.PathValue("organizationId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	var req inviteMemberRequest
	if err := decodeJSON(w, r, &req); err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil)
		return
	}

	invitation, token, err := h.deps.MemberService.Invite(r.Context(), orgID, userID, service.InviteInput{
		Email: req.Email,
		Role:  req.Role,
	})
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusCreated, map[string]any{
		"invitation": invitation,
		"token":      token,
	})
}

func (h *handler) handleAcceptInvitation(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	token := strings.TrimSpace(r.PathValue("token"))
	if token == "" {
		WriteError(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "token is required", map[string]any{
			"token": "cannot be empty",
		})
		return
	}

	membership, err := h.deps.MemberService.AcceptInvitation(r.Context(), token, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, membership)
}

type changeRoleRequest struct {
	Role domain.Role `json:"role"`
}

func (h *handler) handleChangeMemberRole(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	orgID, err := parseUUID("organizationId", r.PathValue("organizationId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	memberID, err := parseUUID("memberId", r.PathValue("memberId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	var req changeRoleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body", nil)
		return
	}

	if err := h.deps.MemberService.ChangeRole(r.Context(), orgID, userID, memberID, req.Role); err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, map[string]string{
		"message": "role updated successfully",
	})
}

func (h *handler) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	orgID, err := parseUUID("organizationId", r.PathValue("organizationId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	memberID, err := parseUUID("memberId", r.PathValue("memberId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	if err := h.deps.MemberService.RemoveMember(r.Context(), orgID, userID, memberID); err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, map[string]string{
		"message": "member removed successfully",
	})
}

func (h *handler) handleInternalGetMember(w http.ResponseWriter, r *http.Request) {
	if h.deps.InternalToken != "" {
		token := r.Header.Get("X-Internal-Token")
		if token == "" || token != h.deps.InternalToken {
			WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid internal token", nil)
			return
		}
	} else {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid internal token", nil)
		return
	}

	orgID, err := parseUUID("organizationId", r.PathValue("organizationId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}
	targetUserID, err := parseUUID("userId", r.PathValue("userId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	mem, err := h.deps.MemberService.GetMemberForUser(r.Context(), orgID, targetUserID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, map[string]any{
		"is_member":       true,
		"role":            mem.Role,
		"organization_id": mem.OrganizationID,
		"user_id":         mem.UserID,
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if r.Body == nil {
		return errors.New("empty request body")
	}
	defer r.Body.Close()

	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}

	// Ensure no trailing bytes beyond EOF
	if dec.Decode(&struct{}{}) != io.EOF {
		return errors.New("unexpected trailing content")
	}

	return nil
}

func parseUUID(param string, val string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(val))
	if err != nil {
		return uuid.Nil, domain.NewValidationError(param+" must be a valid UUID", map[string]any{
			param: "invalid UUID",
		})
	}
	return id, nil
}
