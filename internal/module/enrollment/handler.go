package enrollment

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/middleware"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/pagination"
)

type Handler struct {
	svc      *EnrollmentService
	validate *validator.Validate
}

func NewHandler(svc *EnrollmentService) *Handler {
	return &Handler{svc: svc, validate: validator.New()}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/enrollments", h.listEnrollments)
	r.Get("/enrollments/{id}", h.getEnrollment)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireRole("admin", "staff"))
		r.Post("/enrollments", h.createEnrollment)
		r.Patch("/enrollments/{id}", h.updateEnrollment)
	})
}

// ── request types ─────────────────────────────────────────────────────────────

type createEnrollmentReq struct {
	StudentID string `json:"student_id" validate:"required,uuid"`
	BatchID   string `json:"batch_id"   validate:"required,uuid"`
}

type updateEnrollmentReq struct {
	Status        *string `json:"status"         validate:"omitempty,oneof=dropped completed"`
	PaymentStatus *string `json:"payment_status" validate:"omitempty,oneof=partial paid"`
	FinalGrade    *string `json:"final_grade"`
}

// ── response types ────────────────────────────────────────────────────────────

type studentRef struct {
	ID        string  `json:"id"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Email     *string `json:"email"`
}

type batchRef struct {
	ID        string `json:"id"`
	BatchName string `json:"batch_name"`
	BatchCode string `json:"batch_code"`
	Status    string `json:"status"`
}

type courseRef struct {
	ID         string `json:"id"`
	CourseName string `json:"course_name"`
	CourseCode string `json:"course_code"`
}

type enrollmentResp struct {
	ID             string     `json:"id"`
	Status         string     `json:"status"`
	PaymentStatus  string     `json:"payment_status"`
	FinalGrade     *string    `json:"final_grade"`
	EnrollmentDate string     `json:"enrollment_date"`
	Student        studentRef `json:"student"`
	Batch          batchRef   `json:"batch"`
	Course         courseRef  `json:"course"`
	CreatedAt      string     `json:"created_at"`
	UpdatedAt      string     `json:"updated_at"`
}

func toEnrollmentResp(e Enrollment) enrollmentResp {
	return enrollmentResp{
		ID:             e.ID.String(),
		Status:         e.Status,
		PaymentStatus:  e.PaymentStatus,
		FinalGrade:     e.FinalGrade,
		EnrollmentDate: e.EnrollmentDate.Format("2006-01-02"),
		Student: studentRef{
			ID:        e.Student.ID.String(),
			FirstName: e.Student.FirstName,
			LastName:  e.Student.LastName,
			Email:     e.Student.Email,
		},
		Batch: batchRef{
			ID:        e.Batch.ID.String(),
			BatchName: e.Batch.BatchName,
			BatchCode: e.Batch.BatchCode,
			Status:    e.Batch.Status,
		},
		Course: courseRef{
			ID:         e.Course.ID.String(),
			CourseName: e.Course.CourseName,
			CourseCode: e.Course.CourseCode,
		},
		CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: e.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// ── handlers ──────────────────────────────────────────────────────────────────

func (h *Handler) listEnrollments(w http.ResponseWriter, r *http.Request) {
	page, perPage := pagination.ParsePage(r.URL.Query().Get("page"), r.URL.Query().Get("per_page"))

	params := ListParams{Page: page, PerPage: perPage}
	if v := r.URL.Query().Get("status"); v != "" {
		params.Status = &v
	}
	if v := r.URL.Query().Get("payment_status"); v != "" {
		params.PaymentStatus = &v
	}
	if v := r.URL.Query().Get("batch_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errEnvelope("BAD_REQUEST", "invalid batch_id"))
			return
		}
		params.BatchID = &id
	}
	if v := r.URL.Query().Get("student_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errEnvelope("BAD_REQUEST", "invalid student_id"))
			return
		}
		params.StudentID = &id
	}

	resp, err := h.svc.List(r.Context(), params)
	if err != nil {
		slog.ErrorContext(r.Context(), "list enrollments error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}

	type listOut struct {
		Data       []enrollmentResp `json:"data"`
		Pagination PaginationMeta   `json:"pagination"`
	}
	out := listOut{Pagination: resp.Pagination, Data: []enrollmentResp{}}
	for _, e := range resp.Data {
		out.Data = append(out.Data, toEnrollmentResp(e))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) getEnrollment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	e, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "enrollment not found"))
			return
		}
		slog.ErrorContext(r.Context(), "get enrollment error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}
	writeJSON(w, http.StatusOK, toEnrollmentResp(*e))
}

func (h *Handler) createEnrollment(w http.ResponseWriter, r *http.Request) {
	var req createEnrollmentReq
	if !h.decode(w, r, &req) {
		return
	}

	studentID, err := uuid.Parse(strings.TrimSpace(req.StudentID))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errEnvelope("BAD_REQUEST", "invalid student_id"))
		return
	}
	batchID, err := uuid.Parse(strings.TrimSpace(req.BatchID))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errEnvelope("BAD_REQUEST", "invalid batch_id"))
		return
	}

	e, err := h.svc.Create(r.Context(), CreateRequest{
		StudentID: studentID,
		BatchID:   batchID,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrBatchNotFound):
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "batch not found"))
		case errors.Is(err, ErrStudentNotFound):
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "student not found"))
		case errors.Is(err, ErrBatchCompleted):
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "cannot enroll into a completed batch"))
		case errors.Is(err, ErrCapacityFull):
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "batch is at full capacity"))
		case errors.Is(err, ErrDuplicate):
			writeJSON(w, http.StatusConflict, errEnvelope("CONFLICT", "student already enrolled in this batch"))
		default:
			slog.ErrorContext(r.Context(), "create enrollment error", "error", err)
			writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		}
		return
	}
	writeJSON(w, http.StatusCreated, toEnrollmentResp(*e))
}

func (h *Handler) updateEnrollment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateEnrollmentReq
	if !h.decode(w, r, &req) {
		return
	}

	e, err := h.svc.Update(r.Context(), id, UpdateRequest{
		Status:        req.Status,
		PaymentStatus: req.PaymentStatus,
		FinalGrade:    req.FinalGrade,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "enrollment not found"))
		case errors.Is(err, ErrInvalidTransition):
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "invalid status or payment transition"))
		case errors.Is(err, ErrFinalGradeRequiresCompleted):
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "final grade can only be set on completed enrollments"))
		default:
			slog.ErrorContext(r.Context(), "update enrollment error", "error", err)
			writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		}
		return
	}
	writeJSON(w, http.StatusOK, toEnrollmentResp(*e))
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
