package course

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
	svc      *CourseService
	validate *validator.Validate
}

func NewHandler(svc *CourseService) *Handler {
	return &Handler{svc: svc, validate: validator.New()}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/courses", h.listCourses)
	r.Get("/courses/{id}", h.getCourse)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireRole("admin"))
		r.Post("/courses", h.createCourse)
		r.Patch("/courses/{id}", h.updateCourse)
		r.Patch("/courses/{id}/archive", h.archiveCourse)
	})
}

// ── request types ─────────────────────────────────────────────────────────────

type createCourseReq struct {
	CourseName    string  `json:"course_name"    validate:"required"`
	CourseCode    string  `json:"course_code"    validate:"required"`
	Description   *string `json:"description"`
	Subject       *string `json:"subject"`
	Level         *string `json:"level"          validate:"omitempty,oneof=beginner intermediate advanced"`
	DurationWeeks *int32  `json:"duration_weeks"`
	Price         *string `json:"price"`
	MaxCapacity   *int32  `json:"max_capacity"`
}

type updateCourseReq struct {
	CourseName    *string `json:"course_name"`
	CourseCode    *string `json:"course_code"`
	Description   *string `json:"description"`
	Subject       *string `json:"subject"`
	Level         *string `json:"level"          validate:"omitempty,oneof=beginner intermediate advanced"`
	DurationWeeks *int32  `json:"duration_weeks"`
	Price         *string `json:"price"`
	MaxCapacity   *int32  `json:"max_capacity"`
}

// ── response types ────────────────────────────────────────────────────────────

type courseResp struct {
	ID            string  `json:"id"`
	CourseName    string  `json:"course_name"`
	CourseCode    string  `json:"course_code"`
	Description   *string `json:"description"`
	Subject       *string `json:"subject"`
	Level         *string `json:"level"`
	DurationWeeks *int32  `json:"duration_weeks"`
	Price         *string `json:"price"`
	MaxCapacity   *int32  `json:"max_capacity"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type courseWithStatsResp struct {
	courseResp
	BatchCount        int64 `json:"batch_count"`
	OngoingBatchCount int64 `json:"ongoing_batch_count"`
	EnrolledCount     int64 `json:"enrolled_count"`
}

type archiveResp struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

func toCourseResp(c Course) courseResp {
	return courseResp{
		ID:            c.ID.String(),
		CourseName:    c.CourseName,
		CourseCode:    c.CourseCode,
		Description:   c.Description,
		Subject:       c.Subject,
		Level:         c.Level,
		DurationWeeks: c.DurationWeeks,
		Price:         c.Price,
		MaxCapacity:   c.MaxCapacity,
		Status:        c.Status,
		CreatedAt:     c.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     c.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toCourseWithStatsResp(c CourseWithStats) courseWithStatsResp {
	return courseWithStatsResp{
		courseResp:        toCourseResp(c.Course),
		BatchCount:        c.BatchCount,
		OngoingBatchCount: c.OngoingBatchCount,
		EnrolledCount:     c.EnrolledCount,
	}
}

// ── handlers ──────────────────────────────────────────────────────────────────

func (h *Handler) listCourses(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	params := ListParams{Page: page, PerPage: perPage}
	if v := r.URL.Query().Get("status"); v != "" {
		params.Status = &v
	}
	if v := r.URL.Query().Get("level"); v != "" {
		params.Level = &v
	}
	if v := r.URL.Query().Get("search"); v != "" {
		params.Search = &v
	}

	resp, err := h.svc.List(r.Context(), params)
	if err != nil {
		slog.ErrorContext(r.Context(), "list courses error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}

	type listOut struct {
		Data       []courseWithStatsResp `json:"data"`
		Pagination PaginationMeta        `json:"pagination"`
	}
	out := listOut{Pagination: resp.Pagination, Data: []courseWithStatsResp{}}
	for _, c := range resp.Data {
		out.Data = append(out.Data, toCourseWithStatsResp(c))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) getCourse(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "course not found"))
			return
		}
		slog.ErrorContext(r.Context(), "get course error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}
	writeJSON(w, http.StatusOK, toCourseWithStatsResp(*c))
}

func (h *Handler) createCourse(w http.ResponseWriter, r *http.Request) {
	var req createCourseReq
	if !h.decode(w, r, &req) {
		return
	}
	c, err := h.svc.Create(r.Context(), CreateCourseRequest{
		CourseName:    req.CourseName,
		CourseCode:    req.CourseCode,
		Description:   req.Description,
		Subject:       req.Subject,
		Level:         req.Level,
		DurationWeeks: req.DurationWeeks,
		Price:         req.Price,
		MaxCapacity:   req.MaxCapacity,
	})
	if err != nil {
		if errors.Is(err, ErrCourseCodeConflict) {
			writeJSON(w, http.StatusConflict, errEnvelope("CONFLICT", "course code already in use"))
			return
		}
		slog.ErrorContext(r.Context(), "create course error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}
	writeJSON(w, http.StatusCreated, toCourseResp(*c))
}

func (h *Handler) updateCourse(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateCourseReq
	if !h.decode(w, r, &req) {
		return
	}
	c, err := h.svc.Update(r.Context(), id, UpdateCourseRequest{
		CourseName:    req.CourseName,
		CourseCode:    req.CourseCode,
		Description:   req.Description,
		Subject:       req.Subject,
		Level:         req.Level,
		DurationWeeks: req.DurationWeeks,
		Price:         req.Price,
		MaxCapacity:   req.MaxCapacity,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "course not found"))
		case errors.Is(err, ErrCourseCodeConflict):
			writeJSON(w, http.StatusConflict, errEnvelope("CONFLICT", "course code already in use"))
		default:
			slog.ErrorContext(r.Context(), "update course error", "error", err)
			writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		}
		return
	}
	writeJSON(w, http.StatusOK, toCourseResp(*c))
}

func (h *Handler) archiveCourse(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	result, err := h.svc.Archive(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "course not found"))
		case errors.Is(err, ErrHasOngoingBatches):
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "cannot archive a course with ongoing batches"))
		default:
			slog.ErrorContext(r.Context(), "archive course error", "error", err)
			writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		}
		return
	}
	writeJSON(w, http.StatusOK, archiveResp{
		ID:        result.ID.String(),
		Status:    result.Status,
		UpdatedAt: result.UpdatedAt.UTC().Format(time.RFC3339),
	})
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
