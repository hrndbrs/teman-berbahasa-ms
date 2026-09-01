package schedule

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrScheduleNotFound        = errors.New("schedule not found")
	ErrOverrideNotFound        = errors.New("schedule override not found")
	ErrSessionCapExceeded      = errors.New("schedule would exceed course session limit")
	ErrOriginalDateOutOfRange  = errors.New("original_date not within schedule effective range or wrong day")
	ErrRescheduleMissingDate   = errors.New("reschedule override requires new_date")
	ErrInstructorChangeInvalid = errors.New("instructor_change must not set new_date or new times")
	ErrForbidden               = errors.New("not allowed to modify this schedule")
	ErrBatchNotFound           = errors.New("batch not found")
)

// ── domain types ──────────────────────────────────────────────────────────────

type Schedule struct {
	ID               uuid.UUID
	BatchID          uuid.UUID
	CourseID         uuid.UUID
	InstructorUserID *uuid.UUID
	DayOfWeek        string
	StartTime        string // "09:00:00"
	EndTime          string
	Room             *string
	Recurrence       string // "weekly" | "one-time"
	EffectiveFrom    time.Time
	EffectiveUntil   *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ScheduleOverride struct {
	ID                  uuid.UUID
	ScheduleID          uuid.UUID
	OriginalDate        time.Time
	OverrideType        string // "reschedule" | "instructor_change"
	NewDate             *time.Time
	NewStartTime        *string
	NewEndTime          *string
	NewRoom             *string
	NewInstructorUserID *uuid.UUID
	Reason              *string
	CreatorUserID       uuid.UUID
	CreatedAt           time.Time
	// resolved names for responses
	NewInstructorFirstName *string
	NewInstructorLastName  *string
}

type InstructorRef struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type BatchRef struct {
	ID        string `json:"id"`
	BatchName string `json:"batch_name"`
	BatchCode string `json:"batch_code"`
}

type CourseRef struct {
	ID         string  `json:"id"`
	CourseName string  `json:"course_name"`
	CourseCode string  `json:"course_code"`
	Level      *string `json:"level"`
}

type Session struct {
	ScheduleID          uuid.UUID
	Date                time.Time
	OriginalDate        *time.Time // non-nil when rescheduled from a different date
	DayOfWeek           string
	StartTime           string
	EndTime             string
	Room                *string
	Status              string // "scheduled" | "rescheduled" | "instructor_changed"
	EffectiveInstructor InstructorRef
	Batch               BatchRef
	Course              CourseRef
	Override            *ScheduleOverride
}

// ScheduleForWeek is the enriched row passed to ExpandWeek.
type ScheduleForWeek struct {
	ID                       uuid.UUID
	BatchID                  uuid.UUID
	CourseID                 uuid.UUID
	ScheduleInstructorID     *uuid.UUID
	DayOfWeek                string
	StartTime                string
	EndTime                  string
	Room                     *string
	Recurrence               string
	EffectiveFrom            time.Time
	EffectiveUntil           *time.Time
	BatchName                string
	BatchCode                string
	BatchInstructorID        uuid.UUID
	BatchInstructorFirstName string
	BatchInstructorLastName  string
	SchedInstructorFirstName *string
	SchedInstructorLastName  *string
	CourseName               string
	CourseCode               string
	CourseLevel              *string
}

// OverrideForWeek is the enriched override row passed to ExpandWeek.
type OverrideForWeek struct {
	ID                     uuid.UUID
	ScheduleID             uuid.UUID
	OriginalDate           time.Time
	OverrideType           string
	NewDate                *time.Time
	NewStartTime           *string
	NewEndTime             *string
	NewRoom                *string
	NewInstructorID        *uuid.UUID
	NewInstructorFirstName *string
	NewInstructorLastName  *string
	Reason                 *string
	CreatorUserID          uuid.UUID
	CreatedAt              time.Time
}

// ScheduleForCount is used for session-count enforcement.
type ScheduleForCount struct {
	ID             uuid.UUID
	Recurrence     string
	DayOfWeek      string
	EffectiveFrom  time.Time
	EffectiveUntil *time.Time
}

// ── request types ─────────────────────────────────────────────────────────────

type CreateScheduleRequest struct {
	BatchID          uuid.UUID
	CourseID         uuid.UUID
	InstructorUserID *uuid.UUID
	DayOfWeek        string
	StartTime        string
	EndTime          string
	Room             *string
	Recurrence       string
	EffectiveFrom    time.Time
	EffectiveUntil   *time.Time
	CreatorUserID    uuid.UUID
}

type UpdateScheduleRequest struct {
	InstructorUserID *uuid.UUID
	DayOfWeek        *string
	StartTime        *string
	EndTime          *string
	Room             *string
	Recurrence       *string
	EffectiveFrom    *time.Time
	EffectiveUntil   *time.Time
}

type FullScheduleUpdate struct {
	InstructorUserID *uuid.UUID
	DayOfWeek        string
	StartTime        string
	EndTime          string
	Room             *string
	Recurrence       string
	EffectiveFrom    time.Time
	EffectiveUntil   *time.Time
}

type CreateOverrideRequest struct {
	ScheduleID          uuid.UUID
	OriginalDate        time.Time
	OverrideType        string
	NewDate             *time.Time
	NewStartTime        *string
	NewEndTime          *string
	NewRoom             *string
	NewInstructorUserID *uuid.UUID
	Reason              *string
	CreatorUserID       uuid.UUID
}

type UpdateOverrideRequest struct {
	NewDate             *time.Time
	NewStartTime        *string
	NewEndTime          *string
	NewRoom             *string
	NewInstructorUserID *uuid.UUID
	Reason              *string
}

type WeeklyParams struct {
	WeekStart time.Time
	Level     *string
	BatchID   *uuid.UUID
	CourseID  *uuid.UUID
}

// ── repository interface ──────────────────────────────────────────────────────

type ScheduleRepository interface {
	GetBatchForSchedule(ctx context.Context, batchID uuid.UUID) (BatchForSchedule, error)
	GetCourseSessionCount(ctx context.Context, courseID uuid.UUID) (int32, error)
	ListSchedulesByBatchForCount(ctx context.Context, batchID uuid.UUID) ([]ScheduleForCount, error)
	CreateSchedule(ctx context.Context, id uuid.UUID, req CreateScheduleRequest) (Schedule, error)
	GetScheduleByID(ctx context.Context, id uuid.UUID) (Schedule, error)
	ListSchedulesByBatch(ctx context.Context, batchID uuid.UUID) ([]Schedule, error)
	UpdateSchedule(ctx context.Context, id uuid.UUID, full FullScheduleUpdate) (Schedule, error)
	DeleteSchedule(ctx context.Context, id uuid.UUID) error
	GetSchedulesForWeek(ctx context.Context, params WeeklyParams) ([]ScheduleForWeek, error)
	GetOverridesForWeek(ctx context.Context, scheduleIDs []uuid.UUID, weekStart, weekEnd time.Time) ([]OverrideForWeek, error)
	UpsertOverride(ctx context.Context, id uuid.UUID, req CreateOverrideRequest) (ScheduleOverride, error)
	GetOverrideByID(ctx context.Context, id uuid.UUID) (ScheduleOverride, error)
	UpdateOverride(ctx context.Context, id uuid.UUID, req UpdateOverrideRequest) (ScheduleOverride, error)
	DeleteOverride(ctx context.Context, id uuid.UUID) error
}

// BatchForSchedule is a minimal batch row fetched by the schedule service.
type BatchForSchedule struct {
	ID               uuid.UUID
	CourseID         uuid.UUID
	InstructorUserID uuid.UUID
}

// ── service ───────────────────────────────────────────────────────────────────

type ScheduleService struct {
	repo ScheduleRepository
}

func NewService(repo ScheduleRepository) *ScheduleService {
	return &ScheduleService{repo: repo}
}

// ── schedule CRUD ─────────────────────────────────────────────────────────────

func (s *ScheduleService) CreateSchedule(ctx context.Context, req CreateScheduleRequest) (*Schedule, error) {
	batch, err := s.repo.GetBatchForSchedule(ctx, req.BatchID)
	if err != nil {
		return nil, err
	}
	req.CourseID = batch.CourseID
	_ = batch.InstructorUserID

	if req.Recurrence == "one-time" && req.EffectiveUntil == nil {
		req.EffectiveUntil = &req.EffectiveFrom
	}

	if req.EffectiveUntil != nil {
		sessionCount, err := s.repo.GetCourseSessionCount(ctx, batch.CourseID)
		if err != nil {
			return nil, err
		}
		if sessionCount > 0 {
			existing, err := s.repo.ListSchedulesByBatchForCount(ctx, req.BatchID)
			if err != nil {
				return nil, err
			}
			total := 0
			for _, e := range existing {
				total += CountExpandedSessions(e)
			}
			newCount := CountExpandedSessions(ScheduleForCount{
				Recurrence:     req.Recurrence,
				DayOfWeek:      req.DayOfWeek,
				EffectiveFrom:  req.EffectiveFrom,
				EffectiveUntil: req.EffectiveUntil,
			})
			if total+newCount > int(sessionCount) {
				return nil, ErrSessionCapExceeded
			}
		}
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	sched, err := s.repo.CreateSchedule(ctx, id, req)
	if err != nil {
		return nil, err
	}
	return &sched, nil
}

func (s *ScheduleService) GetScheduleByID(ctx context.Context, rawID string) (*Schedule, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, ErrScheduleNotFound
	}
	sched, err := s.repo.GetScheduleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &sched, nil
}

func (s *ScheduleService) ListSchedulesByBatch(ctx context.Context, rawBatchID string) ([]Schedule, error) {
	batchID, err := uuid.Parse(rawBatchID)
	if err != nil {
		return nil, ErrBatchNotFound
	}
	return s.repo.ListSchedulesByBatch(ctx, batchID)
}

func (s *ScheduleService) UpdateSchedule(ctx context.Context, rawID string, req UpdateScheduleRequest) (*Schedule, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, ErrScheduleNotFound
	}
	existing, err := s.repo.GetScheduleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	merged := mergeScheduleUpdate(existing, req)

	if merged.EffectiveUntil != nil {
		batch, err := s.repo.GetBatchForSchedule(ctx, existing.BatchID)
		if err != nil {
			return nil, err
		}
		sessionCount, err := s.repo.GetCourseSessionCount(ctx, batch.CourseID)
		if err != nil {
			return nil, err
		}
		if sessionCount > 0 {
			allSlots, err := s.repo.ListSchedulesByBatchForCount(ctx, existing.BatchID)
			if err != nil {
				return nil, err
			}
			total := 0
			for _, slot := range allSlots {
				if slot.ID == existing.ID {
					continue
				}
				total += CountExpandedSessions(slot)
			}
			newCount := CountExpandedSessions(ScheduleForCount{
				Recurrence:     merged.Recurrence,
				DayOfWeek:      merged.DayOfWeek,
				EffectiveFrom:  merged.EffectiveFrom,
				EffectiveUntil: merged.EffectiveUntil,
			})
			if total+newCount > int(sessionCount) {
				return nil, ErrSessionCapExceeded
			}
		}
	}

	sched, err := s.repo.UpdateSchedule(ctx, id, merged)
	if err != nil {
		return nil, err
	}
	return &sched, nil
}

func mergeScheduleUpdate(existing Schedule, req UpdateScheduleRequest) FullScheduleUpdate {
	u := FullScheduleUpdate{
		InstructorUserID: existing.InstructorUserID,
		DayOfWeek:        existing.DayOfWeek,
		StartTime:        existing.StartTime,
		EndTime:          existing.EndTime,
		Room:             existing.Room,
		Recurrence:       existing.Recurrence,
		EffectiveFrom:    existing.EffectiveFrom,
		EffectiveUntil:   existing.EffectiveUntil,
	}
	if req.InstructorUserID != nil {
		if *req.InstructorUserID == (uuid.UUID{}) {
			u.InstructorUserID = nil
		} else {
			u.InstructorUserID = req.InstructorUserID
		}
	}
	if req.DayOfWeek != nil {
		u.DayOfWeek = *req.DayOfWeek
	}
	if req.StartTime != nil {
		u.StartTime = *req.StartTime
	}
	if req.EndTime != nil {
		u.EndTime = *req.EndTime
	}
	if req.Room != nil {
		u.Room = req.Room
	}
	if req.Recurrence != nil {
		u.Recurrence = *req.Recurrence
	}
	if req.EffectiveFrom != nil {
		u.EffectiveFrom = *req.EffectiveFrom
	}
	if req.EffectiveUntil != nil {
		u.EffectiveUntil = req.EffectiveUntil
	}
	return u
}

func (s *ScheduleService) DeleteSchedule(ctx context.Context, rawID string) error {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return ErrScheduleNotFound
	}
	return s.repo.DeleteSchedule(ctx, id)
}

// ── override CRUD ─────────────────────────────────────────────────────────────

func (s *ScheduleService) CreateOverride(ctx context.Context, req CreateOverrideRequest, role string) (*ScheduleOverride, error) {
	sched, err := s.repo.GetScheduleByID(ctx, req.ScheduleID)
	if err != nil {
		return nil, err
	}
	batch, err := s.repo.GetBatchForSchedule(ctx, sched.BatchID)
	if err != nil {
		return nil, err
	}

	if role == "teacher" && batch.InstructorUserID != req.CreatorUserID {
		return nil, ErrForbidden
	}

	if err := validateOriginalDate(sched, req.OriginalDate); err != nil {
		return nil, err
	}
	if err := validateOverrideType(req.OverrideType, req.NewDate, req.NewStartTime, req.NewEndTime); err != nil {
		return nil, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	o, err := s.repo.UpsertOverride(ctx, id, req)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (s *ScheduleService) GetOverrideByID(ctx context.Context, rawID string) (*ScheduleOverride, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, ErrOverrideNotFound
	}
	o, err := s.repo.GetOverrideByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (s *ScheduleService) UpdateOverride(ctx context.Context, rawID string, req UpdateOverrideRequest, callerID uuid.UUID, role string) (*ScheduleOverride, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, ErrOverrideNotFound
	}
	existing, err := s.repo.GetOverrideByID(ctx, id)
	if err != nil {
		return nil, err
	}
	sched, err := s.repo.GetScheduleByID(ctx, existing.ScheduleID)
	if err != nil {
		return nil, err
	}
	batch, err := s.repo.GetBatchForSchedule(ctx, sched.BatchID)
	if err != nil {
		return nil, err
	}
	if role == "teacher" && batch.InstructorUserID != callerID {
		return nil, ErrForbidden
	}

	merged := mergeOverrideUpdate(existing, req)
	if err := validateOverrideType(existing.OverrideType, merged.NewDate, merged.NewStartTime, merged.NewEndTime); err != nil {
		return nil, err
	}

	o, err := s.repo.UpdateOverride(ctx, id, merged)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func mergeOverrideUpdate(existing ScheduleOverride, req UpdateOverrideRequest) UpdateOverrideRequest {
	merged := UpdateOverrideRequest{
		NewDate:             existing.NewDate,
		NewStartTime:        existing.NewStartTime,
		NewEndTime:          existing.NewEndTime,
		NewRoom:             existing.NewRoom,
		NewInstructorUserID: existing.NewInstructorUserID,
		Reason:              existing.Reason,
	}
	if req.NewDate != nil {
		merged.NewDate = req.NewDate
	}
	if req.NewStartTime != nil {
		merged.NewStartTime = req.NewStartTime
	}
	if req.NewEndTime != nil {
		merged.NewEndTime = req.NewEndTime
	}
	if req.NewRoom != nil {
		merged.NewRoom = req.NewRoom
	}
	if req.NewInstructorUserID != nil {
		merged.NewInstructorUserID = req.NewInstructorUserID
	}
	if req.Reason != nil {
		merged.Reason = req.Reason
	}
	return merged
}

func (s *ScheduleService) DeleteOverride(ctx context.Context, rawID string) error {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return ErrOverrideNotFound
	}
	return s.repo.DeleteOverride(ctx, id)
}

// ── weekly calendar ───────────────────────────────────────────────────────────

type WeeklyResponse struct {
	WeekStart     time.Time
	WeekEnd       time.Time
	TotalSessions int
	TotalHours    float64
	Sessions      []Session
}

func (s *ScheduleService) GetWeekly(ctx context.Context, params WeeklyParams) (*WeeklyResponse, error) {
	params.WeekStart = truncateToMonday(params.WeekStart)
	weekEnd := params.WeekStart.AddDate(0, 0, 6)

	schedRows, err := s.repo.GetSchedulesForWeek(ctx, params)
	if err != nil {
		return nil, err
	}

	var schedIDs []uuid.UUID
	for _, r := range schedRows {
		schedIDs = append(schedIDs, r.ID)
	}

	var overrideRows []OverrideForWeek
	if len(schedIDs) > 0 {
		overrideRows, err = s.repo.GetOverridesForWeek(ctx, schedIDs, params.WeekStart, weekEnd)
		if err != nil {
			return nil, err
		}
	}

	sessions := ExpandWeek(schedRows, overrideRows, params.WeekStart, weekEnd)

	totalHours := computeTotalHours(sessions)

	return &WeeklyResponse{
		WeekStart:     params.WeekStart,
		WeekEnd:       weekEnd,
		TotalSessions: len(sessions),
		TotalHours:    totalHours,
		Sessions:      sessions,
	}, nil
}

func truncateToMonday(t time.Time) time.Time {
	for t.Weekday() != time.Monday {
		t = t.AddDate(0, 0, -1)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func computeTotalHours(sessions []Session) float64 {
	total := 0.0
	for _, s := range sessions {
		start := parseTimeStr(s.StartTime)
		end := parseTimeStr(s.EndTime)
		total += end.Sub(start).Hours()
	}
	return total
}

func parseTimeStr(s string) time.Time {
	t, _ := time.Parse("15:04:05", s)
	return t
}

// ── validation helpers ────────────────────────────────────────────────────────

func validateOriginalDate(sched Schedule, d time.Time) error {
	if d.Before(sched.EffectiveFrom) {
		return ErrOriginalDateOutOfRange
	}
	if sched.EffectiveUntil != nil && d.After(*sched.EffectiveUntil) {
		return ErrOriginalDateOutOfRange
	}
	if weekdayName(d.Weekday()) != sched.DayOfWeek {
		return ErrOriginalDateOutOfRange
	}
	return nil
}

func validateOverrideType(overrideType string, newDate *time.Time, newStartTime, newEndTime *string) error {
	switch overrideType {
	case "reschedule":
		if newDate == nil {
			return ErrRescheduleMissingDate
		}
	case "instructor_change":
		if newDate != nil || newStartTime != nil || newEndTime != nil {
			return ErrInstructorChangeInvalid
		}
	}
	return nil
}
