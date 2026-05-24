package schedule

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
	"github.com/hrndbrs/teman-berbahasa-ms/internal/patch"
)

type Handler struct {
	svc      *ScheduleService
	validate *validator.Validate
}

func NewHandler(svc *ScheduleService) *Handler {
	return &Handler{svc: svc, validate: validator.New()}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/batches/{id}/schedules", h.listSchedulesByBatch)
	r.Get("/schedules/weekly", h.getWeekly)
	r.Get("/schedules/{id}", h.getSchedule)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireRole("admin", "staff"))
		r.Post("/batches/{id}/schedules", h.createSchedule)
		r.Patch("/schedules/{id}", h.updateSchedule)
		r.Delete("/schedules/{id}", h.deleteSchedule)
		r.Delete("/schedule-overrides/{id}", h.deleteOverride)
	})

	r.Post("/schedules/{id}/overrides", h.createOverride)
	r.Get("/schedule-overrides/{id}", h.getOverride)
	r.Patch("/schedule-overrides/{id}", h.updateOverride)
}

// ── request types ─────────────────────────────────────────────────────────────

type createScheduleReq struct {
	InstructorUserID *string `json:"instructor_user_id" validate:"omitempty,uuid"`
	DayOfWeek        string  `json:"day_of_week"        validate:"required,oneof=monday tuesday wednesday thursday friday saturday sunday"`
	StartTime        string  `json:"start_time"         validate:"required"`
	EndTime          string  `json:"end_time"           validate:"required"`
	Room             *string `json:"room"`
	Recurrence       string  `json:"recurrence"         validate:"required,oneof=weekly one-time"`
	EffectiveFrom    string  `json:"effective_from"     validate:"required"`
	EffectiveUntil   *string `json:"effective_until"`
}

type updateScheduleReq struct {
	InstructorUserID *patch.Patchable[string] `json:"instructor_user_id"`
	DayOfWeek        *string                  `json:"day_of_week"  validate:"omitempty,oneof=monday tuesday wednesday thursday friday saturday sunday"`
	StartTime        *string                  `json:"start_time"`
	EndTime          *string                  `json:"end_time"`
	Room             *patch.Patchable[string] `json:"room"`
	Recurrence       *string                  `json:"recurrence"   validate:"omitempty,oneof=weekly one-time"`
	EffectiveFrom    *string                  `json:"effective_from"`
	EffectiveUntil   *patch.Patchable[string] `json:"effective_until"`
}

type createOverrideReq struct {
	OriginalDate        string  `json:"original_date"          validate:"required"`
	OverrideType        string  `json:"override_type"          validate:"required,oneof=reschedule instructor_change"`
	NewDate             *string `json:"new_date"`
	NewStartTime        *string `json:"new_start_time"`
	NewEndTime          *string `json:"new_end_time"`
	NewRoom             *string `json:"new_room"`
	NewInstructorUserID *string `json:"new_instructor_user_id" validate:"omitempty,uuid"`
	Reason              *string `json:"reason"`
}

type updateOverrideReq struct {
	NewDate             *string `json:"new_date"`
	NewStartTime        *string `json:"new_start_time"`
	NewEndTime          *string `json:"new_end_time"`
	NewRoom             *string `json:"new_room"`
	NewInstructorUserID *string `json:"new_instructor_user_id" validate:"omitempty,uuid"`
	Reason              *string `json:"reason"`
}

// ── response types ────────────────────────────────────────────────────────────

type scheduleResp struct {
	ID             string         `json:"id"`
	BatchID        string         `json:"batch_id"`
	CourseID       string         `json:"course_id"`
	Instructor     *instructorRef `json:"instructor"`
	DayOfWeek      string         `json:"day_of_week"`
	StartTime      string         `json:"start_time"`
	EndTime        string         `json:"end_time"`
	Room           *string        `json:"room"`
	Recurrence     string         `json:"recurrence"`
	EffectiveFrom  string         `json:"effective_from"`
	EffectiveUntil *string        `json:"effective_until"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
}

type instructorRef struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type overrideResp struct {
	ID            string         `json:"id"`
	ScheduleID    string         `json:"schedule_id"`
	OriginalDate  string         `json:"original_date"`
	OverrideType  string         `json:"override_type"`
	NewDate       *string        `json:"new_date"`
	NewStartTime  *string        `json:"new_start_time"`
	NewEndTime    *string        `json:"new_end_time"`
	NewRoom       *string        `json:"new_room"`
	NewInstructor *instructorRef `json:"new_instructor"`
	Reason        *string        `json:"reason"`
	CreatedBy     string         `json:"created_by_user_id"`
	CreatedAt     string         `json:"created_at"`
}

type sessionResp struct {
	ScheduleID          string         `json:"schedule_id"`
	Date                string         `json:"date"`
	OriginalDate        *string        `json:"original_date"`
	DayOfWeek           string         `json:"day_of_week"`
	StartTime           string         `json:"start_time"`
	EndTime             string         `json:"end_time"`
	Room                *string        `json:"room"`
	Status              string         `json:"status"`
	EffectiveInstructor InstructorRef  `json:"effective_instructor"`
	Batch               BatchRef       `json:"batch"`
	Course              CourseRef      `json:"course"`
	Override            *overrideResp  `json:"override"`
}

type weeklyResp struct {
	WeekStart     string        `json:"week_start"`
	WeekEnd       string        `json:"week_end"`
	TotalSessions int           `json:"total_sessions"`
	TotalHours    float64       `json:"total_hours"`
	Sessions      []sessionResp `json:"data"`
}

func toScheduleResp(s Schedule) scheduleResp {
	r := scheduleResp{
		ID:            s.ID.String(),
		BatchID:       s.BatchID.String(),
		CourseID:      s.CourseID.String(),
		DayOfWeek:     s.DayOfWeek,
		StartTime:     s.StartTime,
		EndTime:       s.EndTime,
		Room:          s.Room,
		Recurrence:    s.Recurrence,
		EffectiveFrom: s.EffectiveFrom.Format("2006-01-02"),
		CreatedAt:     s.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     s.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if s.EffectiveUntil != nil {
		d := s.EffectiveUntil.Format("2006-01-02")
		r.EffectiveUntil = &d
	}
	if s.InstructorUserID != nil {
		r.Instructor = &instructorRef{ID: s.InstructorUserID.String()}
	}
	return r
}

func toOverrideResp(o ScheduleOverride) overrideResp {
	r := overrideResp{
		ID:           o.ID.String(),
		ScheduleID:   o.ScheduleID.String(),
		OriginalDate: o.OriginalDate.Format("2006-01-02"),
		OverrideType: o.OverrideType,
		NewStartTime: o.NewStartTime,
		NewEndTime:   o.NewEndTime,
		NewRoom:      o.NewRoom,
		Reason:       o.Reason,
		CreatedBy:    o.CreatedByUserID.String(),
		CreatedAt:    o.CreatedAt.UTC().Format(time.RFC3339),
	}
	if o.NewDate != nil {
		d := o.NewDate.Format("2006-01-02")
		r.NewDate = &d
	}
	if o.NewInstructorUserID != nil {
		r.NewInstructor = &instructorRef{
			ID:        o.NewInstructorUserID.String(),
			FirstName: derefStr(o.NewInstructorFirstName),
			LastName:  derefStr(o.NewInstructorLastName),
		}
	}
	return r
}

func toSessionResp(s Session) sessionResp {
	r := sessionResp{
		ScheduleID:          s.ScheduleID.String(),
		Date:                s.Date.Format("2006-01-02"),
		DayOfWeek:           s.DayOfWeek,
		StartTime:           s.StartTime,
		EndTime:             s.EndTime,
		Room:                s.Room,
		Status:              s.Status,
		EffectiveInstructor: s.EffectiveInstructor,
		Batch:               s.Batch,
		Course:              s.Course,
	}
	if s.OriginalDate != nil {
		d := s.OriginalDate.Format("2006-01-02")
		r.OriginalDate = &d
	}
	if s.Override != nil {
		o := toOverrideResp(*s.Override)
		r.Override = &o
	}
	return r
}

// ── handlers ──────────────────────────────────────────────────────────────────

func (h *Handler) listSchedulesByBatch(w http.ResponseWriter, r *http.Request) {
	batchID := chi.URLParam(r, "id")
	scheds, err := h.svc.ListSchedulesByBatch(r.Context(), batchID)
	if err != nil {
		if errors.Is(err, ErrBatchNotFound) {
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "batch not found"))
			return
		}
		slog.ErrorContext(r.Context(), "list schedules error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}
	out := make([]scheduleResp, len(scheds))
	for i, s := range scheds {
		out[i] = toScheduleResp(s)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) getSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s, err := h.svc.GetScheduleByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrScheduleNotFound) {
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "schedule not found"))
			return
		}
		slog.ErrorContext(r.Context(), "get schedule error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}
	writeJSON(w, http.StatusOK, toScheduleResp(*s))
}

func (h *Handler) createSchedule(w http.ResponseWriter, r *http.Request) {
	batchIDStr := chi.URLParam(r, "id")
	batchID, err := uuid.Parse(batchIDStr)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "batch not found"))
		return
	}

	var req createScheduleReq
	if !h.decode(w, r, &req) {
		return
	}

	from, err := time.Parse("2006-01-02", strings.TrimSpace(req.EffectiveFrom))
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "invalid effective_from date"))
		return
	}
	var until *time.Time
	if req.EffectiveUntil != nil {
		t, err := time.Parse("2006-01-02", strings.TrimSpace(*req.EffectiveUntil))
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "invalid effective_until date"))
			return
		}
		until = &t
	}

	callerStr := middleware.UserIDFromCtx(r.Context())
	callerID, err := uuid.Parse(callerStr)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errEnvelope("UNAUTHORIZED", "invalid user identity"))
		return
	}

	svcReq := CreateScheduleRequest{
		BatchID:         batchID,
		DayOfWeek:       req.DayOfWeek,
		StartTime:       strings.TrimSpace(req.StartTime),
		EndTime:         strings.TrimSpace(req.EndTime),
		Room:            req.Room,
		Recurrence:      req.Recurrence,
		EffectiveFrom:   from,
		EffectiveUntil:  until,
		CreatedByUserID: callerID,
	}
	if req.InstructorUserID != nil {
		id, err := uuid.Parse(*req.InstructorUserID)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "invalid instructor_user_id"))
			return
		}
		svcReq.InstructorUserID = &id
	}

	s, err := h.svc.CreateSchedule(r.Context(), svcReq)
	if err != nil {
		switch {
		case errors.Is(err, ErrBatchNotFound):
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "batch not found"))
		case errors.Is(err, ErrSessionCapExceeded):
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", err.Error()))
		default:
			slog.ErrorContext(r.Context(), "create schedule error", "error", err)
			writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		}
		return
	}
	writeJSON(w, http.StatusCreated, toScheduleResp(*s))
}

func (h *Handler) updateSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateScheduleReq
	if !h.decode(w, r, &req) {
		return
	}

	svcReq := UpdateScheduleRequest{
		DayOfWeek:  req.DayOfWeek,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Recurrence: req.Recurrence,
	}

	if instrPatch := req.InstructorUserID; instrPatch != nil && instrPatch.Set() {
		if !instrPatch.IsNull && instrPatch.Value != nil {
			id, err := uuid.Parse(*instrPatch.Value)
			if err != nil {
				writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "invalid instructor_user_id"))
				return
			}
			svcReq.InstructorUserID = &id
		} else {
			var zero uuid.UUID
			svcReq.InstructorUserID = &zero
		}
	}

	if req.Room != nil && req.Room.Set() {
		if !req.Room.IsNull && req.Room.Value != nil {
			svcReq.Room = req.Room.Value
		} else {
			svcReq.Room = nil
		}
	}

	if req.EffectiveFrom != nil {
		t, err := time.Parse("2006-01-02", strings.TrimSpace(*req.EffectiveFrom))
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "invalid effective_from"))
			return
		}
		svcReq.EffectiveFrom = &t
	}
	if req.EffectiveUntil != nil && req.EffectiveUntil.Set() {
		if !req.EffectiveUntil.IsNull && req.EffectiveUntil.Value != nil {
			t, err := time.Parse("2006-01-02", strings.TrimSpace(*req.EffectiveUntil.Value))
			if err != nil {
				writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "invalid effective_until"))
				return
			}
			svcReq.EffectiveUntil = &t
		}
	}

	s, err := h.svc.UpdateSchedule(r.Context(), id, svcReq)
	if err != nil {
		switch {
		case errors.Is(err, ErrScheduleNotFound):
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "schedule not found"))
		case errors.Is(err, ErrSessionCapExceeded):
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", err.Error()))
		default:
			slog.ErrorContext(r.Context(), "update schedule error", "error", err)
			writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		}
		return
	}
	writeJSON(w, http.StatusOK, toScheduleResp(*s))
}

func (h *Handler) deleteSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteSchedule(r.Context(), id); err != nil {
		if errors.Is(err, ErrScheduleNotFound) {
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "schedule not found"))
			return
		}
		slog.ErrorContext(r.Context(), "delete schedule error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getWeekly(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	weekStart := time.Now()
	if v := q.Get("week_start"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errEnvelope("BAD_REQUEST", "invalid week_start date"))
			return
		}
		weekStart = t
	}

	params := WeeklyParams{WeekStart: weekStart}
	if v := q.Get("level"); v != "" {
		params.Level = &v
	}
	if v := q.Get("batch_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errEnvelope("BAD_REQUEST", "invalid batch_id"))
			return
		}
		params.BatchID = &id
	}
	if v := q.Get("course_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errEnvelope("BAD_REQUEST", "invalid course_id"))
			return
		}
		params.CourseID = &id
	}

	resp, err := h.svc.GetWeekly(r.Context(), params)
	if err != nil {
		slog.ErrorContext(r.Context(), "get weekly schedules error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}

	sessions := make([]sessionResp, len(resp.Sessions))
	for i, s := range resp.Sessions {
		sessions[i] = toSessionResp(s)
	}
	writeJSON(w, http.StatusOK, weeklyResp{
		WeekStart:     resp.WeekStart.Format("2006-01-02"),
		WeekEnd:       resp.WeekEnd.Format("2006-01-02"),
		TotalSessions: resp.TotalSessions,
		TotalHours:    resp.TotalHours,
		Sessions:      sessions,
	})
}

func (h *Handler) createOverride(w http.ResponseWriter, r *http.Request) {
	schedIDStr := chi.URLParam(r, "id")
	schedID, err := uuid.Parse(schedIDStr)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "schedule not found"))
		return
	}

	var req createOverrideReq
	if !h.decode(w, r, &req) {
		return
	}

	callerStr := middleware.UserIDFromCtx(r.Context())
	callerID, err := uuid.Parse(callerStr)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errEnvelope("UNAUTHORIZED", "invalid user identity"))
		return
	}
	role := middleware.UserRoleFromCtx(r.Context())

	origDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.OriginalDate))
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "invalid original_date"))
		return
	}
	var newDate *time.Time
	if req.NewDate != nil {
		t, err := time.Parse("2006-01-02", strings.TrimSpace(*req.NewDate))
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "invalid new_date"))
			return
		}
		newDate = &t
	}

	svcReq := CreateOverrideRequest{
		ScheduleID:      schedID,
		OriginalDate:    origDate,
		OverrideType:    req.OverrideType,
		NewDate:         newDate,
		NewStartTime:    req.NewStartTime,
		NewEndTime:      req.NewEndTime,
		NewRoom:         req.NewRoom,
		Reason:          req.Reason,
		CreatedByUserID: callerID,
	}
	if req.NewInstructorUserID != nil {
		id, err := uuid.Parse(*req.NewInstructorUserID)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "invalid new_instructor_user_id"))
			return
		}
		svcReq.NewInstructorUserID = &id
	}

	o, err := h.svc.CreateOverride(r.Context(), svcReq, role)
	if err != nil {
		switch {
		case errors.Is(err, ErrScheduleNotFound):
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "schedule not found"))
		case errors.Is(err, ErrForbidden):
			writeJSON(w, http.StatusForbidden, errEnvelope("FORBIDDEN", err.Error()))
		case errors.Is(err, ErrOriginalDateOutOfRange):
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", err.Error()))
		case errors.Is(err, ErrRescheduleMissingDate):
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", err.Error()))
		case errors.Is(err, ErrInstructorChangeInvalid):
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", err.Error()))
		default:
			slog.ErrorContext(r.Context(), "create override error", "error", err)
			writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		}
		return
	}
	writeJSON(w, http.StatusCreated, toOverrideResp(*o))
}

func (h *Handler) getOverride(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	o, err := h.svc.GetOverrideByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrOverrideNotFound) {
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "override not found"))
			return
		}
		slog.ErrorContext(r.Context(), "get override error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}
	writeJSON(w, http.StatusOK, toOverrideResp(*o))
}

func (h *Handler) updateOverride(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateOverrideReq
	if !h.decode(w, r, &req) {
		return
	}

	callerStr := middleware.UserIDFromCtx(r.Context())
	callerID, err := uuid.Parse(callerStr)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errEnvelope("UNAUTHORIZED", "invalid user identity"))
		return
	}
	role := middleware.UserRoleFromCtx(r.Context())

	svcReq := UpdateOverrideRequest{
		NewStartTime: req.NewStartTime,
		NewEndTime:   req.NewEndTime,
		NewRoom:      req.NewRoom,
		Reason:       req.Reason,
	}
	if req.NewDate != nil {
		t, err := time.Parse("2006-01-02", strings.TrimSpace(*req.NewDate))
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "invalid new_date"))
			return
		}
		svcReq.NewDate = &t
	}
	if req.NewInstructorUserID != nil {
		id, err := uuid.Parse(*req.NewInstructorUserID)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", "invalid new_instructor_user_id"))
			return
		}
		svcReq.NewInstructorUserID = &id
	}

	o, err := h.svc.UpdateOverride(r.Context(), id, svcReq, callerID, role)
	if err != nil {
		switch {
		case errors.Is(err, ErrOverrideNotFound), errors.Is(err, ErrScheduleNotFound):
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "override not found"))
		case errors.Is(err, ErrForbidden):
			writeJSON(w, http.StatusForbidden, errEnvelope("FORBIDDEN", err.Error()))
		case errors.Is(err, ErrInstructorChangeInvalid), errors.Is(err, ErrRescheduleMissingDate):
			writeJSON(w, http.StatusUnprocessableEntity, errEnvelope("VALIDATION_ERROR", err.Error()))
		default:
			slog.ErrorContext(r.Context(), "update override error", "error", err)
			writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		}
		return
	}
	writeJSON(w, http.StatusOK, toOverrideResp(*o))
}

func (h *Handler) deleteOverride(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteOverride(r.Context(), id); err != nil {
		if errors.Is(err, ErrOverrideNotFound) {
			writeJSON(w, http.StatusNotFound, errEnvelope("NOT_FOUND", "override not found"))
			return
		}
		slog.ErrorContext(r.Context(), "delete override error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errEnvelope("INTERNAL_ERROR", "internal server error"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (h *Handler) decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, errEnvelope("REQUEST_TOO_LARGE", "request body exceeds 1MB"))
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
