package batch

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
	"github.com/google/uuid"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/middleware"
)

type Handler struct {
	svc      *BatchService
	validate *validator.Validate
}

func NewHandler(svc *BatchService) *Handler {
	return &Handler{svc: svc, validate: validator.New()}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/batches", h.listBatches)
	r.Get("/batches/{id}", h.getBatch)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireRole("admin", "staff"))
		r.Post("/batches", h.createBatch)
		r.Patch("/batches/{id}", h.updateBatch)
		r.Patch("/batches/{id}/status", h.transitionStatus)
		r.Delete("/batches/{id}", h.deleteBatch)
	})
}

// ── request types ─────────────────────────────────────────────────────────────

type createBatchReq struct {
	CourseID         string  `json:"course_id"          validate:"required,uuid"`
	InstructorUserID string  `json:"instructor_user_id" validate:"required,uuid"`
	BatchName        string  `json:"batch_name"         validate:"required"`
	BatchCode        string  `json:"batch_code"         validate:"required"`
	StartDate        *string `json:"start_date"`
	EndDate          *string `json:"end_date"`
	AcademicYear     *string `json:"academic_year"`
}

type updateBatchReq struct {
	InstructorUserID *string `json:"instructor_user_id" validate:"omitempty,uuid"`
	BatchName        *string `json:"batch_name"`
	BatchCode        *string `json:"batch_code"`
	StartDate        *string `json:"start_date"`
	EndDate          *string `json:"end_date"`
	AcademicYear     *string `json:"academic_year"`
}

type transitionStatusReq struct {
	Status string `json:"status" validate:"required,oneof=upcoming ongoing completed"`
}

// ── response types ────────────────────────────────────────────────────────────

type courseRef struct {
	ID         string `json:"id"`
	CourseName string `json:"course_name"`
	CourseCode string `json:"course_code"`
}

type instructorRef struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type batchResp struct {
	ID            string        `json:"id"`
	BatchName     string        `json:"batch_name"`
	BatchCode     string        `json:"batch_code"`
	StartDate     *string       `json:"start_date"`
	EndDate       *string       `json:"end_date"`
	AcademicYear  *string       `json:"academic_year"`
	Status        string        `json:"status"`
	Course        courseRef     `json:"course"`
	Instructor    instructorRef `json:"instructor"`
	EnrolledCount int64         `json:"enrolled_count"`
	CreatedAt     string        `json:"created_at"`
	UpdatedAt     string        `json:"updated_at"`
}

type statusResp struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

func toBatchResp(b BatchWithDetails) batchResp {
	return batchResp{
		ID:           b.ID.String(),
		BatchName:    b.BatchName,
		BatchCode:    b.BatchCode,
		StartDate:    formatDate(b.StartDate),
		EndDate:      formatDate(b.EndDate),
		AcademicYear: b.AcademicYear,
		Status:       b.Status,
		Course: courseRef{
			ID:         b.CourseID.String(),
			CourseName: b.CourseName,
			CourseCode: b.CourseCode,
		},
		Instructor: instructorRef{
			ID:        b.InstructorUserID.String(),
			FirstName: b.InstructorFirstName,
			LastName:  b.InstructorLastName,
		},
		EnrolledCount: b.EnrolledCount,
		CreatedAt:     b.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     b.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// ── handlers ──────────────────────────────────────────────────────────────────

func (h *Handler) listBatches(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	params := ListParams{Page: page, PerPage: perPage}
	if v := r.URL.Query().Get("status"); v != "" {
		params.Status = &v
	}
	if v := r.URL.Query().Get("search"); v != "" {
		params.Search = &v
	}
	if v := r.URL.Query().Get("course_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errEnvelope("BAD_REQUEST", "invalid course_id"))
			return
		}
		params.CourseID = &id
	}

	resp, err := h.svc.List(r.Context(), params)
	if err != nil {
		slog.ErrorContext(r.Context(), "list batches error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}

	type listOut struct {
		Data       []batchResp    `json:"data"`
		Pagination PaginationMeta `json:"pagination"`
	}
	out := listOut{Pagination: resp.Pagination, Data: []batchResp{}}
	for _, b := range resp.Data {
		out.Data = append(out.Data, toBatchResp(b))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) getBatch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	b, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "batch not found"))
			return
		}
		slog.ErrorContext(r.Context(), "get batch error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}
	writeJSON(w, http.StatusOK, toBatchResp(*b))
}

func (h *Handler) createBatch(w http.ResponseWriter, r *http.Request) {
	var req createBatchReq
	if !h.decode(w, r, &req) {
		return
	}

	courseID, _ := uuid.Parse(req.CourseID)
	instructorID, _ := uuid.Parse(req.InstructorUserID)

	createdByStr := middleware.UserIDFromCtx(r.Context())
	createdByID, err := uuid.Parse(createdByStr)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errEnvelope("UNAUTHORIZED", "invalid user identity"))
		return
	}

	b, err := h.svc.Create(r.Context(), CreateBatchRequest{
		CourseID:         courseID,
		InstructorUserID: instructorID,
		CreatedByUserID:  createdByID,
		BatchName:        req.BatchName,
		BatchCode:        req.BatchCode,
		StartDate:        parseDate(req.StartDate),
		EndDate:          parseDate(req.EndDate),
		AcademicYear:     req.AcademicYear,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrBatchCodeConflict):
			writeJSON(w, http.StatusConflict, errEnvelope("CONFLICT", "batch code already in use for this course"))
		case errors.Is(err, ErrInvalidInstructor):
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "instructor must be an active teacher"))
		default:
			slog.ErrorContext(r.Context(), "create batch error", "error", err)
			writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		}
		return
	}
	writeJSON(w, http.StatusCreated, toBatchResp(*b))
}

func (h *Handler) updateBatch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateBatchReq
	if !h.decode(w, r, &req) {
		return
	}

	updateReq := UpdateBatchRequest{
		BatchName:    req.BatchName,
		BatchCode:    req.BatchCode,
		StartDate:    parseDate(req.StartDate),
		EndDate:      parseDate(req.EndDate),
		AcademicYear: req.AcademicYear,
	}
	if req.InstructorUserID != nil {
		parsed, err := uuid.Parse(*req.InstructorUserID)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "invalid instructor_user_id"))
			return
		}
		updateReq.InstructorUserID = &parsed
	}

	b, err := h.svc.Update(r.Context(), id, updateReq)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "batch not found"))
		case errors.Is(err, ErrBatchCodeConflict):
			writeJSON(w, http.StatusConflict, errEnvelope("CONFLICT", "batch code already in use for this course"))
		case errors.Is(err, ErrInvalidInstructor):
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "instructor must be an active teacher"))
		default:
			slog.ErrorContext(r.Context(), "update batch error", "error", err)
			writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		}
		return
	}
	writeJSON(w, http.StatusOK, toBatchResp(*b))
}

func (h *Handler) transitionStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req transitionStatusReq
	if !h.decode(w, r, &req) {
		return
	}

	result, err := h.svc.TransitionStatus(r.Context(), id, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "batch not found"))
		case errors.Is(err, ErrInvalidTransition):
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "invalid status transition"))
		default:
			slog.ErrorContext(r.Context(), "transition batch status error", "error", err)
			writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		}
		return
	}
	writeJSON(w, http.StatusOK, statusResp{
		ID:        result.ID.String(),
		Status:    result.Status,
		UpdatedAt: result.UpdatedAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) deleteBatch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := h.svc.Delete(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "batch not found"))
		case errors.Is(err, ErrHasActiveEnrollments):
			writeJSON(w, http.StatusConflict, errEnvelope("CONFLICT", "cannot delete batch with active enrollments"))
		default:
			slog.ErrorContext(r.Context(), "delete batch error", "error", err)
			writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func parseDate(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil
	}
	return &t
}

func formatDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
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
