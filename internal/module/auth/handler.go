package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/middleware"
)

type Handler struct {
	svc      *AuthService
	validate *validator.Validate
}

func NewHandler(svc *AuthService) *Handler {
	return &Handler{svc: svc, validate: validator.New()}
}

func (h *Handler) Register(r chi.Router) {
	r.Post("/auth/login", h.login)
	r.Post("/auth/refresh", h.refresh)
	r.Post("/auth/logout", h.logout)
	r.Post("/auth/forgot-password", h.forgotPassword)
	r.Post("/auth/reset-password", h.resetPassword)
}

func (h *Handler) RegisterProtected(r chi.Router) {
	r.Get("/auth/me", h.getMe)
}

type loginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type forgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token"        validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !h.decode(w, r, &req) {
		return
	}
	resp, err := h.svc.Login(r.Context(), strings.TrimSpace(req.Email), req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrAccountLocked):
			writeJSON(w, http.StatusUnauthorized, errEnvelope("ACCOUNT_LOCKED", "account is locked due to too many failed login attempts"))
		case errors.Is(err, ErrInvalidCredentials):
			writeJSON(w, http.StatusUnauthorized, errEnvelope("UNAUTHORIZED", "invalid email or password"))
		default:
			slog.ErrorContext(r.Context(), "login error", "error", err)
			writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	info, err := h.svc.Me(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errEnvelope("UNAUTHORIZED", "invalid or expired token"))
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if !h.decode(w, r, &req) {
		return
	}
	pair, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			writeJSON(w, http.StatusUnauthorized, errEnvelope("UNAUTHORIZED", "invalid or expired refresh token"))
			return
		}
		slog.ErrorContext(r.Context(), "refresh error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}
	writeJSON(w, http.StatusOK, pair)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if !h.decode(w, r, &req) {
		return
	}
	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		slog.ErrorContext(r.Context(), "logout error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if !h.decode(w, r, &req) {
		return
	}
	_ = h.svc.ForgotPassword(r.Context(), strings.TrimSpace(req.Email))
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "If that email exists, a reset link has been sent.",
	})
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if !h.decode(w, r, &req) {
		return
	}
	err := h.svc.ResetPassword(r.Context(), req.Token, req.NewPassword)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			writeJSON(w, http.StatusBadRequest, errEnvelope("BAD_REQUEST", "invalid or expired reset token"))
			return
		}
		slog.ErrorContext(r.Context(), "reset password error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Password updated. Please log in again.",
	})
}

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	info, err := h.svc.GetMe(r.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			writeJSON(w, http.StatusUnauthorized, errEnvelope("UNAUTHORIZED", "invalid or expired token"))
			return
		}
		slog.ErrorContext(r.Context(), "get me error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (h *Handler) decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, errEnvelope("BAD_REQUEST", "invalid request body"))
		return false
	}
	if err := h.validate.Struct(v); err != nil {
		fields := map[string]string{}
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			for _, fe := range ve {
				fields[strings.ToLower(fe.Field())] = fe.Tag()
			}
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error": map[string]any{
				"code":    "VALIDATION_ERROR",
				"message": "validation failed",
				"fields":  fields,
			},
		})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func errEnvelope(code, message string) map[string]any {
	return map[string]any{
		"error": map[string]string{"code": code, "message": message},
	}
}
