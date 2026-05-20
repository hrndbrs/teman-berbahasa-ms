package user

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/middleware"
)

type Handler struct {
	svc      *UserService
	validate *validator.Validate
}

func NewHandler(svc *UserService) *Handler {
	return &Handler{svc: svc, validate: validator.New()}
}

func (h *Handler) Register(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireRole("admin"))
		r.Get("/users", h.listUsers)
		r.Post("/users", h.createUser)
		r.Patch("/users/{id}", h.updateUser)
	})
	r.Get("/users/{id}", h.getUser)
}

// ── request types ─────────────────────────────────────────────────────────────

type createUserReq struct {
	FirstName string  `json:"first_name" validate:"required"`
	LastName  string  `json:"last_name"  validate:"required"`
	Email     string  `json:"email"      validate:"required,email"`
	Role      string  `json:"role"       validate:"required,oneof=admin teacher staff"`
	Phone     *string `json:"phone"`
}

type updateUserReq struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Email     *string `json:"email"   validate:"omitempty,email"`
	Role      *string `json:"role"    validate:"omitempty,oneof=admin teacher staff"`
	Phone     *string `json:"phone"`
	Status    *string `json:"status"  validate:"omitempty,oneof=active inactive"`
}

// ── response type ─────────────────────────────────────────────────────────────

type userResp struct {
	ID        string  `json:"id"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Email     string  `json:"email"`
	Role      string  `json:"role"`
	Phone     *string `json:"phone"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func toResp(u User) userResp {
	return userResp{
		ID:        u.ID.String(),
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Email:     u.Email,
		Role:      u.Role,
		Phone:     u.Phone,
		Status:    u.Status,
		CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// ── handlers ──────────────────────────────────────────────────────────────────

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	var role, status *string
	if v := r.URL.Query().Get("role"); v != "" {
		role = &v
	}
	if v := r.URL.Query().Get("status"); v != "" {
		status = &v
	}

	resp, err := h.svc.ListUsers(r.Context(), ListParams{
		Page: page, PerPage: perPage, Role: role, Status: status,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "list users error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}

	type listOut struct {
		Data       []userResp     `json:"data"`
		Pagination PaginationMeta `json:"pagination"`
	}
	out := listOut{Pagination: resp.Pagination, Data: []userResp{}}
	for _, u := range resp.Data {
		out.Data = append(out.Data, toResp(u))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var req createUserReq
	if !h.decode(w, r, &req) {
		return
	}
	u, err := h.svc.CreateUser(r.Context(), CreateUserRequest{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     strings.TrimSpace(req.Email),
		Role:      req.Role,
		Phone:     req.Phone,
	})
	if err != nil {
		if errors.Is(err, ErrEmailConflict) {
			writeJSON(w, http.StatusConflict, errEnvelope("CONFLICT", "email already in use"))
			return
		}
		slog.ErrorContext(r.Context(), "create user error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}
	writeJSON(w, http.StatusCreated, toResp(*u))
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "id")
	callerID := middleware.UserIDFromCtx(r.Context())
	callerRole := middleware.UserRoleFromCtx(r.Context())

	u, err := h.svc.GetUser(r.Context(), callerID, callerRole, targetID)
	if err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			writeJSON(w, http.StatusForbidden, errEnvelope("FORBIDDEN", "access denied"))
		case errors.Is(err, ErrNotFound):
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "user not found"))
		default:
			slog.ErrorContext(r.Context(), "get user error", "error", err)
			writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		}
		return
	}
	writeJSON(w, http.StatusOK, toResp(*u))
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "id")
	var req updateUserReq
	if !h.decode(w, r, &req) {
		return
	}
	u, err := h.svc.UpdateUser(r.Context(), targetID, UpdateUserRequest{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Role:      req.Role,
		Phone:     req.Phone,
		Status:    req.Status,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailConflict):
			writeJSON(w, http.StatusConflict, errEnvelope("CONFLICT", "email already in use"))
		case errors.Is(err, ErrNotFound):
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "user not found"))
		default:
			slog.ErrorContext(r.Context(), "update user error", "error", err)
			writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		}
		return
	}
	writeJSON(w, http.StatusOK, toResp(*u))
}

// ── helpers ───────────────────────────────────────────────────────────────────

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
