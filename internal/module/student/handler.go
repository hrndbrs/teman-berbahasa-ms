package student

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/middleware"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/pagination"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/patch"
)

type Handler struct {
	svc      *StudentService
	validate *validator.Validate
}

func NewHandler(svc *StudentService) *Handler {
	return &Handler{svc: svc, validate: validator.New()}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/students", h.listStudents)
	r.Get("/students/{id}", h.getStudent)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireRole("admin", "staff"))
		r.Post("/students", h.createStudent)
		r.Patch("/students/{id}", h.updateStudent)
	})
}

// ── request types ─────────────────────────────────────────────────────────────

type createStudentReq struct {
	FirstName        string  `json:"first_name"         validate:"required"`
	LastName         string  `json:"last_name"          validate:"required"`
	Email            string  `json:"email"              validate:"required,email"`
	Phone            *string `json:"phone"`
	DateOfBirth      *string `json:"date_of_birth"      validate:"omitempty,datetime=2006-01-02"`
	Gender           *string `json:"gender"             validate:"omitempty,oneof=male female other"`
	Address          *string `json:"address"`
	ParentName       *string `json:"parent_name"`
	ParentPhone      *string `json:"parent_phone"`
	RegistrationDate *string `json:"registration_date"  validate:"omitempty,datetime=2006-01-02"`
}

type rawUpdateStudentReq struct {
	FirstName   *string                  `json:"first_name"`
	LastName    *string                  `json:"last_name"`
	Email       *string                  `json:"email"           validate:"omitempty,email"`
	Phone       *patch.Patchable[string] `json:"phone"`
	DateOfBirth *patch.Patchable[string] `json:"date_of_birth"`
	Gender      *patch.Patchable[string] `json:"gender"`
	Address     *patch.Patchable[string] `json:"address"`
	ParentName  *patch.Patchable[string] `json:"parent_name"`
	ParentPhone *patch.Patchable[string] `json:"parent_phone"`
	Status      *string                  `json:"status"`
}

type updateStudentReq struct {
	FirstName   *string `json:"first_name"`
	LastName    *string `json:"last_name"`
	Email       *string `json:"email"         validate:"omitempty,email"`
	Phone       *string `json:"phone"`
	DateOfBirth *string `json:"date_of_birth" validate:"omitempty,datetime=2006-01-02"`
	Gender      *string `json:"gender"        validate:"omitempty,oneof=male female other"`
	Address     *string `json:"address"`
	ParentName  *string `json:"parent_name"`
	ParentPhone *string `json:"parent_phone"`
	Status      *string `json:"status"        validate:"omitempty,oneof=active inactive graduated"`
}

// ── response types ────────────────────────────────────────────────────────────

type studentResp struct {
	ID               string  `json:"id"`
	FirstName        string  `json:"first_name"`
	LastName         string  `json:"last_name"`
	Email            string  `json:"email"`
	Phone            *string `json:"phone"`
	DateOfBirth      *string `json:"date_of_birth"`
	Gender           *string `json:"gender"`
	Address          *string `json:"address"`
	ParentName       *string `json:"parent_name"`
	ParentPhone      *string `json:"parent_phone"`
	RegistrationDate string  `json:"registration_date"`
	Status           string  `json:"status"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

type listItemResp struct {
	studentResp
	EnrolledBatches []string `json:"enrolled_batches"`
}

type batchRefResp struct {
	ID        string `json:"id"`
	BatchName string `json:"batch_name"`
	BatchCode string `json:"batch_code"`
	Status    string `json:"status"`
}

type courseRefResp struct {
	ID         string  `json:"id"`
	CourseName string  `json:"course_name"`
	CourseCode string  `json:"course_code"`
	Level      *string `json:"level"`
}

type enrollmentResp struct {
	ID             string        `json:"id"`
	Status         string        `json:"status"`
	PaymentStatus  string        `json:"payment_status"`
	EnrollmentDate string        `json:"enrollment_date"`
	Batch          batchRefResp  `json:"batch"`
	Course         courseRefResp `json:"course"`
}

type detailResp struct {
	studentResp
	Enrollments []enrollmentResp `json:"enrollments"`
}

// ── converters ────────────────────────────────────────────────────────────────

func toStudentResp(s Student) studentResp {
	var dob *string
	if s.DateOfBirth != nil {
		v := s.DateOfBirth.Format("2006-01-02")
		dob = &v
	}
	return studentResp{
		ID:               s.ID.String(),
		FirstName:        s.FirstName,
		LastName:         s.LastName,
		Email:            s.Email,
		Phone:            s.Phone,
		DateOfBirth:      dob,
		Gender:           s.Gender,
		Address:          s.Address,
		ParentName:       s.ParentName,
		ParentPhone:      s.ParentPhone,
		RegistrationDate: s.RegistrationDate.Format("2006-01-02"),
		Status:           s.Status,
		CreatedAt:        s.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        s.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toEnrollmentResp(e EnrollmentSummary) enrollmentResp {
	return enrollmentResp{
		ID:             e.ID.String(),
		Status:         e.Status,
		PaymentStatus:  e.PaymentStatus,
		EnrollmentDate: e.EnrollmentDate.Format("2006-01-02"),
		Batch: batchRefResp{
			ID:        e.Batch.ID.String(),
			BatchName: e.Batch.BatchName,
			BatchCode: e.Batch.BatchCode,
			Status:    e.Batch.Status,
		},
		Course: courseRefResp{
			ID:         e.Course.ID.String(),
			CourseName: e.Course.CourseName,
			CourseCode: e.Course.CourseCode,
			Level:      e.Course.Level,
		},
	}
}

// ── handlers ──────────────────────────────────────────────────────────────────

func (h *Handler) listStudents(w http.ResponseWriter, r *http.Request) {
	page, perPage := pagination.ParsePage(r.URL.Query().Get("page"), r.URL.Query().Get("per_page"))
	var status, search *string
	if v := r.URL.Query().Get("status"); v != "" {
		status = &v
	}
	if v := r.URL.Query().Get("search"); v != "" {
		v = strings.TrimSpace(v)
		search = &v
	}

	resp, err := h.svc.ListStudents(r.Context(), ListParams{
		Page: page, PerPage: perPage, Status: status, Search: search,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "list students error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}

	type listOut struct {
		Data       []listItemResp `json:"data"`
		Pagination PaginationMeta `json:"pagination"`
	}
	out := listOut{Pagination: resp.Pagination, Data: []listItemResp{}}
	for _, item := range resp.Data {
		out.Data = append(out.Data, listItemResp{
			studentResp:     toStudentResp(item.Student),
			EnrolledBatches: item.EnrolledBatches,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) getStudent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	detail, err := h.svc.GetStudent(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "student not found"))
			return
		}
		slog.ErrorContext(r.Context(), "get student error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}

	enrollments := make([]enrollmentResp, len(detail.Enrollments))
	for i, e := range detail.Enrollments {
		enrollments[i] = toEnrollmentResp(e)
	}
	writeJSON(w, http.StatusOK, detailResp{
		studentResp: toStudentResp(detail.Student),
		Enrollments: enrollments,
	})
}

func (h *Handler) createStudent(w http.ResponseWriter, r *http.Request) {
	var req createStudentReq
	if !h.decode(w, r, &req) {
		return
	}
	dob, err := parseDatePtr(req.DateOfBirth)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "invalid date_of_birth format"))
		return
	}
	regDate, err := parseDatePtr(req.RegistrationDate)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "invalid registration_date format"))
		return
	}

	s, err := h.svc.CreateStudent(r.Context(), CreateStudentRequest{
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		Email:            req.Email,
		Phone:            req.Phone,
		DateOfBirth:      dob,
		Gender:           req.Gender,
		Address:          req.Address,
		ParentName:       req.ParentName,
		ParentPhone:      req.ParentPhone,
		RegistrationDate: regDate,
	})
	if err != nil {
		if errors.Is(err, ErrEmailConflict) {
			writeJSON(w, http.StatusConflict, errEnvelope("CONFLICT", "email already in use"))
			return
		}
		slog.ErrorContext(r.Context(), "create student error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}
	writeJSON(w, http.StatusCreated, toStudentResp(*s))
}

func (h *Handler) updateStudent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var raw rawUpdateStudentReq
	if !h.decode(w, r, &raw) {
		return
	}

	var dob *patch.Patchable[time.Time]
	if raw.DateOfBirth.Set() {
		dob = &patch.Patchable[time.Time]{Sent: true, IsNull: raw.DateOfBirth.IsNull}
		if !raw.DateOfBirth.IsNull && raw.DateOfBirth.Value != nil {
			t, err := time.Parse("2006-01-02", *raw.DateOfBirth.Value)
			if err != nil {
				writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "invalid date_of_birth format"))
				return
			}
			dob.Value = &t
		}
	}

	s, err := h.svc.UpdateStudent(r.Context(), id, UpdateStudentRequest{
		FirstName: raw.FirstName,
		LastName:  raw.LastName,

		Email:       raw.Email,
		Phone:       raw.Phone,
		DateOfBirth: dob,
		Gender:      raw.Gender,
		Address:     raw.Address,
		ParentName:  raw.ParentName,
		ParentPhone: raw.ParentPhone,

		Status: raw.Status,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "student not found"))
		case errors.Is(err, ErrEmailConflict):
			writeJSON(w, http.StatusConflict, errEnvelope("CONFLICT", "email already in use"))
		default:
			slog.ErrorContext(r.Context(), "update student error", "error", err)
			writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		}
		return
	}
	writeJSON(w, http.StatusOK, toStudentResp(*s))
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (h *Handler) decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge,
				errEnvelope("REQUEST_TOO_LARGE", "request body exceeds 1MB"))
			return false
		}
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

func parseDatePtr(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
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
